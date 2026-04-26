package pipeline

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"orc/internal/git"
	"orc/internal/handoff"
	"orc/internal/log"
	"orc/internal/review"
	"orc/internal/session"
	"orc/internal/spec"
	"orc/internal/state"
	"orc/internal/topo"
	"orc/internal/verify"
)

type Config struct {
	SpecPath   string
	Root       string
	ExtraSpecs []string
	PlanModel  string
	ExecModel  string
}

type Pipeline struct {
	Config Config
	logger *slog.Logger
}

func New(cfg Config) *Pipeline {
	return &Pipeline{
		Config: cfg,
		logger: log.Get("pipeline"),
	}
}

func (p *Pipeline) Run() int {
	cwd, err := os.Getwd()
	if err != nil {
		p.logger.Error("getting cwd", "error", err)
		return 1
	}
	root := p.Config.Root

	// Resolve spec paths relative to CWD
	specPath := filepath.Join(cwd, p.Config.SpecPath)
	var extraPaths []string
	for _, e := range p.Config.ExtraSpecs {
		extraPaths = append(extraPaths, filepath.Join(cwd, e))
	}

	projectSpec, err := spec.Load(specPath, extraPaths...)
	if err != nil {
		p.logger.Error("loading spec", "error", err)
		return 1
	}

	if errs := spec.Preflight(projectSpec); len(errs) > 0 {
		p.logger.Error("preflight validation failed")
		for _, e := range errs {
			p.logger.Error("  - " + e)
		}
		return 1
	}

	topoMilestones := make([]topo.Milestone, len(projectSpec.Milestones))
	ids := make([]string, len(projectSpec.Milestones))
	for i, ms := range projectSpec.Milestones {
		topoMilestones[i] = topo.Milestone{ID: ms.ID, DependsOn: ms.DependsOn}
		ids[i] = ms.ID
	}

	ordered, err := topo.Sort(topoMilestones)
	if err != nil {
		p.logger.Error("topological sort failed", "error", err)
		return 1
	}
	p.logger.Info("milestone order", "order", fmt.Sprintf("%v", ordered))

	// State in CWD, not root
	statePath := filepath.Join(cwd, "state.yaml")
	pipeState := state.New(ids, statePath)

	// Validate root, git init if needed
	if err := p.ensureRoot(pipeState); err != nil {
		p.logger.Error("root validation", "error", err)
		return 1
	}
	root = p.Config.Root

	// Model fallback: if only ExecModel is set, use it for plan too
	if p.Config.PlanModel == "" {
		p.Config.PlanModel = p.Config.ExecModel
	}

	seenHandoffPaths := make(map[string]bool)
	var allHandoffNotes []handoff.Note
	var allVerifyResults []verify.Result
	hasFailures := false

	for _, msID := range ordered {
		ms := spec.GetMilestone(projectSpec, msID)
		if ms == nil {
			continue
		}

		if pipeState.IsCompleted(msID) {
			p.logger.Info("skipping completed milestone", "name", msID)
			if ms.Verify != nil {
				vResults, err := verify.Run(ms.Verify, 0, root)
				if err == nil {
					allVerifyResults = append(allVerifyResults, vResults...)
					if !verify.AllSuccessful(vResults) {
						p.logger.Warn("some verify commands failed", "name", msID)
					}
				}
			}
			newNotes, _ := handoff.Collect(root)
			for _, n := range newNotes {
				if !seenHandoffPaths[n.Source] {
					seenHandoffPaths[n.Source] = true
					allHandoffNotes = append(allHandoffNotes, n)
				}
			}
			continue
		}

		pipeState.Set(msID, state.StatusInProgress)
		pipeState.Save()

		// Phase 1: Plan session
		p.logger.Info("starting plan session", "name", msID)
		planPrompt := buildPlanPrompt(ms, cwd)
		p.logger.Info("running plan session", "name", msID, "prompt_bytes", len(planPrompt), "model", p.Config.PlanModel)
		planResult, planErr := session.Run(msID+"-plan", planPrompt, root, 0, p.Config.PlanModel)
		if planErr != nil || planResult.ReturnCode != 0 {
			rc := -1
			if planResult != nil {
				rc = planResult.ReturnCode
			}
			p.logger.Error("plan session failed", "name", msID, "code", rc, "error", planErr,
				"plan_model", p.Config.PlanModel, "plan_path", filepath.Join(root, ".orc_history", "PLAN.md"))
			pipeState.Set(msID, state.StatusFailed)
			pipeState.Save()
			fmt.Fprintf(os.Stderr, "\nERROR: Plan session for milestone %q failed (exit code %d).\n", msID, rc)
			fmt.Fprintf(os.Stderr, "Plan model: %s\n", p.Config.PlanModel)
			fmt.Fprintf(os.Stderr, "PLAN.md not written to %s\n", filepath.Join(root, ".orc_history", "PLAN.md"))
			fmt.Fprintf(os.Stderr, "Fix your configuration and re-run.\n")
			return 1
		}
		p.logger.Info("plan session completed", "name", msID)

		// Phase 2-5: Exec session
		p.logger.Info("starting exec session", "name", msID)
		execPrompt := buildExecPrompt(ms, projectSpec.Milestones, pipeState, allHandoffNotes, cwd)
		p.logger.Info("running exec session", "name", msID, "prompt_bytes", len(execPrompt), "model", p.Config.ExecModel)
		execResult, execErr := session.Run(msID+"-exec", execPrompt, root, 0, p.Config.ExecModel)
		if execErr != nil || execResult.ReturnCode != 0 {
			pipeState.Set(msID, state.StatusFailed)
			hasFailures = true
			rc := -1
			if execResult != nil {
				rc = execResult.ReturnCode
			}
			p.logger.Warn("exec session failed", "name", msID, "code", rc, "error", execErr)
			pipeState.Save()
			continue
		}
		pipeState.Set(msID, state.StatusCompleted)
		pipeState.Save()

		if ms.Verify != nil {
			vResults, err := verify.Run(ms.Verify, 0, root)
			if err == nil {
				allVerifyResults = append(allVerifyResults, vResults...)
				if !verify.AllSuccessful(vResults) {
					p.logger.Warn("some verify commands failed", "name", msID)
				}
			}
		}

		newNotes, _ := handoff.Collect(root)
		for _, n := range newNotes {
			if !seenHandoffPaths[n.Source] {
				seenHandoffPaths[n.Source] = true
				allHandoffNotes = append(allHandoffNotes, n)
			}
		}

		if reviewResult, reviewErr := review.Phase(ms.ID, ms.Name, buildSpecContent(ms, cwd), root, allHandoffNotes, allVerifyResults, p.Config.ExecModel); reviewErr != nil {
			p.logger.Error("phase review failed", "name", msID, "error", reviewErr)
			fmt.Fprintf(os.Stderr, "\nERROR: Phase review for milestone %q failed.\n", msID)
			fmt.Fprintf(os.Stderr, "Review the output above and fix the issues, then re-run.\n")
			return 1
		} else {
			_ = reviewResult
			p.logger.Info("phase review completed", "name", msID)
		}
	}

	if pipeState.AllCompleted() {
		p.logger.Info("pipeline completed successfully")
		return 0
	}
	if hasFailures {
		p.logger.Warn("pipeline completed with failures")
		return 1
	}
	return 0
}

// ensureRoot validates root dir based on local state, inits git if needed.
func (p *Pipeline) ensureRoot(pipeState *state.State) error {
	root := p.Config.Root
	var err error
	if !filepath.IsAbs(root) {
		root, err = filepath.Abs(root)
		if err != nil {
			return fmt.Errorf("resolving root: %w", err)
		}
		p.Config.Root = root
	}

	hasProgress := false
	for _, info := range pipeState.GetAll() {
		if info != state.StatusPending {
			hasProgress = true
			break
		}
	}

	if hasProgress {
		fi, statErr := os.Stat(root)
		if statErr != nil || !fi.IsDir() {
			return fmt.Errorf("root %q does not exist, but state shows project in progress", root)
		}
		gitDir := filepath.Join(root, ".git")
		if _, gitErr := os.Stat(gitDir); gitErr != nil {
			return fmt.Errorf("root %q is not a git repository, but state shows project in progress", root)
		}
		return nil
	}

	if fi, statErr := os.Stat(root); statErr == nil {
		if !fi.IsDir() {
			return fmt.Errorf("root %q exists but is not a directory", root)
		}
		entries, readErr := os.ReadDir(root)
		if readErr != nil {
			return fmt.Errorf("reading root dir: %w", readErr)
		}
		if len(entries) > 0 {
			return fmt.Errorf("root %q must be empty for a new project", root)
		}
	} else if os.IsNotExist(statErr) {
		if mkErr := os.MkdirAll(root, 0755); mkErr != nil {
			return fmt.Errorf("creating root dir: %w", mkErr)
		}
	} else {
		return fmt.Errorf("checking root: %w", statErr)
	}

	if err := git.RepoInit(root); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	if err := git.InitCommit(root); err != nil {
		return fmt.Errorf("git init commit: %w", err)
	}
	p.logger.Info("initialized git repo in root", "root", root)

	return nil
}

func buildPlanPrompt(ms *spec.Milestone, cwd string) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("# Milestone: %s\n", ms.Name))
	parts = append(parts, "## Git Rules\n")
	parts = append(parts, "CRITICAL: Do NOT create, switch, or merge git branches. All work on the CURRENT branch only.\n")
	parts = append(parts, "After code review and quality checks pass, create a git commit on the current branch.\n")
	parts = append(parts, "You may use `git status`, `git diff`, `git log`, `git add`, `git commit` for tracking progress.\n\n")

	if len(ms.Specs) > 0 {
		parts = append(parts, "## Specs\n")
		for _, s := range ms.Specs {
			parts = append(parts, fmt.Sprintf("### %s\n", s.ID))
			if s.SpecFile != "" {
				absPath := filepath.Join(cwd, s.SpecFile)
				parts = append(parts, fmt.Sprintf("- Read `%s` for all requirements.\n", absPath))
			}
		}
	}

	parts = append(parts, "## Plan Phase\n")
	parts = append(parts, "1. Load the `writing-plans` skill and create a detailed implementation plan\n")
	parts = append(parts, "2. Write the plan to `.orc_history/PLAN.md`\n")
	parts = append(parts, "3. When writing-plans asks \"Execution Handoff\", choose \"Write Only\" — do NOT execute\n")
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func buildExecPrompt(ms *spec.Milestone, all []spec.Milestone, pipeState *state.State, handoffNotes []handoff.Note, cwd string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("# Milestone: %s\n", ms.Name))

	parts = append(parts, "## Git Rules\n")
	parts = append(parts, "CRITICAL: Do NOT create, switch, or merge git branches. All work on the CURRENT branch only.\n")
	parts = append(parts, "After code review and quality checks pass, create a git commit on the current branch.\n")
	parts = append(parts, "You may use `git status`, `git diff`, `git log`, `git add`, `git commit` for tracking progress.\n\n")

	if len(ms.Specs) > 0 {
		parts = append(parts, "## Specs\n")
		for _, s := range ms.Specs {
			parts = append(parts, fmt.Sprintf("### %s\n", s.ID))
			if s.SpecFile != "" {
				absPath := filepath.Join(cwd, s.SpecFile)
				parts = append(parts, fmt.Sprintf("- Read `%s` for all requirements.\n", absPath))
			}
		}
	}

	parts = append(parts, "## Plan\n")
	parts = append(parts, "Read `.orc_history/PLAN.md` for the implementation plan.\n\n")

	parts = append(parts, "## Development Process\n")
	parts = append(parts, "### Phase 2: Execute\n")
	parts = append(parts, "1. Load the `executing-plans` skill and execute `PLAN.md` using **inline** mode\n")
	parts = append(parts, "2. Follow the plan step by step to implement the spec\n")
	parts = append(parts, "3. Write tests first (TDD), implement, verify\n")
	parts = append(parts, "4. Do NOT commit yet — review comes next\n\n")
	parts = append(parts, "### Phase 3: Quality\n")
	parts = append(parts, "1. Load the `verification-before-completion` skill\n")
	parts = append(parts, "2. Run all quality checks — fix ALL issues immediately before proceeding\n")
	parts = append(parts, "3. Do NOT proceed until all checks pass\n\n")
	parts = append(parts, "### Phase 4: Code Review\n")
	parts = append(parts, "1. Load the `requesting-code-review` skill\n")
	parts = append(parts, "2. Review all changes:\n")
	parts = append(parts, "   - Get git SHAs: `BASE_SHA=$(git rev-parse HEAD~1)` and `HEAD_SHA=$(git rev-parse HEAD)`\n")
	parts = append(parts, "   - Check spec compliance — every requirement implemented, no scope creep\n")
	parts = append(parts, "   - Check code quality — error handling, edge cases, test coverage\n")
	parts = append(parts, "   - Categorize issues: Critical (must fix), Important (should fix), Minor (nice to have)\n")
	parts = append(parts, "3. Fix all Critical and Important issues\n")
	parts = append(parts, "4. Re-review if fixes were needed\n\n")
	parts = append(parts, "### Phase 5: Commit\n")
	parts = append(parts, "1. Stage all reviewed changes with `git add`\n")
	parts = append(parts, "2. Commit with a descriptive message\n")
	parts = append(parts, fmt.Sprintf("3. Write review notes to `.orc_history/HANDOFF-%s.md` documenting the review outcome\n", ms.ID))

	if pipeState != nil {
		parts = append(parts, "## Pipeline State\n")
		for name, st := range pipeState.GetAll() {
			parts = append(parts, fmt.Sprintf("- %s: %s\n", name, st))
		}
	}

	if len(handoffNotes) > 0 {
		parts = append(parts, "## Previous Handoff Notes\n")
		for _, n := range handoffNotes {
			parts = append(parts, fmt.Sprintf("### %s\n\n%s\n", n.Source, n.Content))
		}
	}

	var remaining []spec.Milestone
	for _, m := range all {
		if m.ID != ms.ID {
			remaining = append(remaining, m)
		}
	}
	if len(remaining) > 0 {
		parts = append(parts, "## Upcoming Milestones\n")
		for _, m := range remaining {
			parts = append(parts, fmt.Sprintf("- %s\n", m.ID))
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func buildSpecContent(ms *spec.Milestone, cwd string) string {
	var parts []string
	for _, s := range ms.Specs {
		parts = append(parts, fmt.Sprintf("### %s\n", s.ID))
		if s.SpecFile != "" {
			absPath := filepath.Join(cwd, s.SpecFile)
			parts = append(parts, fmt.Sprintf("- Read `%s` for all requirements.\n", absPath))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}


