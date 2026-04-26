package pipeline

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"orc/internal/context"
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
	specPath := filepath.Join(p.Config.Root, p.Config.SpecPath)
	var extraPaths []string
	for _, e := range p.Config.ExtraSpecs {
		extraPaths = append(extraPaths, filepath.Join(p.Config.Root, e))
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

	pipeState := state.New(ids, filepath.Join(p.Config.Root, "state.yaml"))

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
				vResults, err := verify.Run(ms.Verify, 0)
				if err == nil {
					allVerifyResults = append(allVerifyResults, vResults...)
					if !verify.AllSuccessful(vResults) {
						p.logger.Warn("some verify commands failed", "name", msID)
					}
				}
			}
			newNotes, _ := handoff.Collect(p.Config.Root)
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

		p.logger.Info("starting milestone", "name", msID)
		prompt := computeMilestoneSpec(ms, projectSpec.Milestones, pipeState, allHandoffNotes, p.Config.Root)
		p.logger.Info("running milestone", "name", msID, "prompt_bytes", len(prompt))

		result, err := session.Run(msID, prompt, p.Config.Root, 0)
		if err != nil || result.ReturnCode != 0 {
			pipeState.Set(msID, state.StatusFailed)
			hasFailures = true
			rc := -1
			if result != nil {
				rc = result.ReturnCode
			}
			p.logger.Warn("session failed", "name", msID, "code", rc, "error", err)
			pipeState.Save()
			continue
		}
		pipeState.Set(msID, state.StatusCompleted)
		pipeState.Save()

		if ms.Verify != nil {
			vResults, err := verify.Run(ms.Verify, 0)
			if err == nil {
				allVerifyResults = append(allVerifyResults, vResults...)
				if !verify.AllSuccessful(vResults) {
					p.logger.Warn("some verify commands failed", "name", msID)
				}
			}
		}

		newNotes, _ := handoff.Collect(p.Config.Root)
		var deduped []handoff.Note
		for _, n := range newNotes {
			if !seenHandoffPaths[n.Source] {
				seenHandoffPaths[n.Source] = true
				deduped = append(deduped, n)
			}
		}
		allHandoffNotes = append(allHandoffNotes, deduped...)

		review.Phase(msID, buildSpecContent(ms, p.Config.Root), p.Config.Root, deduped, allVerifyResults)
	}

	var msInfos []context.MilestoneInfo
	for _, ms := range projectSpec.Milestones {
		msInfos = append(msInfos, context.MilestoneInfo{Name: ms.Name, Spec: buildSpecContent(&ms, p.Config.Root)})
	}
	review.Final(projectSpec.Project.Name, p.Config.Root, msInfos, allHandoffNotes, allVerifyResults)

	if pipeState.AllCompleted() {
		git.Tag(projectSpec.Project.Name + "-v1.0")
		p.logger.Info("pipeline completed successfully")
		return 0
	}
	if hasFailures {
		p.logger.Warn("pipeline completed with failures")
		return 1
	}
	return 0
}

func computeMilestoneSpec(ms *spec.Milestone, all []spec.Milestone, pipeState *state.State, handoffNotes []handoff.Note, rootDir string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("# Milestone: %s\n", ms.Name))

	parts = append(parts, "## Git Rules\n")
	parts = append(parts, "CRITICAL: Do NOT create, switch, or merge git branches. All work on the CURRENT branch only.\n")
	parts = append(parts, "After completing each spec (tests passing), create a git commit on the current branch.\n")
	parts = append(parts, "You may use `git status`, `git diff`, `git log`, `git add`, `git commit` for tracking progress.\n\n")

	if len(ms.Specs) > 0 {
		parts = append(parts, "## Specs\n")
		for _, s := range ms.Specs {
			parts = append(parts, fmt.Sprintf("### %s\n", s.ID))
			if s.SpecFile != "" {
				parts = append(parts, fmt.Sprintf("- Read `%s` for all requirements.\n", s.SpecFile))
			}
		}
	}

	parts = append(parts, "## Development Process\n")
	parts = append(parts, "### Phase 1: Plan\n")
	parts = append(parts, "1. Load the `writing-plans` skill and create a detailed implementation plan\n")
	parts = append(parts, "2. Write the plan to `PLAN.md` in the project root\n\n")
	parts = append(parts, "### Phase 2: Execute\n")
	parts = append(parts, "1. Load the `executing-plans` skill and execute `PLAN.md` using **inline** mode\n")
	parts = append(parts, "2. Follow the plan step by step to implement the spec\n")
	parts = append(parts, "3. Write tests first (TDD), implement, verify, and commit\n")

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

func buildSpecContent(ms *spec.Milestone, rootDir string) string {
	var parts []string
	for _, s := range ms.Specs {
		parts = append(parts, fmt.Sprintf("### %s\n", s.ID))
		if s.SpecFile != "" {
			parts = append(parts, fmt.Sprintf("- Read `%s` for all requirements.\n", s.SpecFile))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}


