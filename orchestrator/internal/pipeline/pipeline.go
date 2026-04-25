package pipeline

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"orchestrator/internal/context"
	"orchestrator/internal/git"
	"orchestrator/internal/handoff"
	"orchestrator/internal/log"
	"orchestrator/internal/review"
	"orchestrator/internal/session"
	"orchestrator/internal/spec"
	"orchestrator/internal/state"
	"orchestrator/internal/topo"
	"orchestrator/internal/verify"
)

type Config struct {
	SpecPath   string
	Root       string
	ExtraSpecs []string
	Branch     string
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
	names := make([]string, len(projectSpec.Milestones))
	for i, ms := range projectSpec.Milestones {
		topoMilestones[i] = topo.Milestone{Name: ms.Name, DependsOn: ms.DependsOn}
		names[i] = ms.Name
	}

	ordered, err := topo.Sort(topoMilestones)
	if err != nil {
		p.logger.Error("topological sort failed", "error", err)
		return 1
	}
	p.logger.Info("milestone order", "order", fmt.Sprintf("%v", ordered))

	pipeState := state.New(names, filepath.Join(p.Config.Root, "state.yaml"))
	defaultBranch := detectDefaultBranch()

	seenHandoffPaths := make(map[string]bool)
	var allHandoffNotes []handoff.Note
	var allVerifyResults []verify.Result
	hasFailures := false

	for _, msName := range ordered {
		ms := spec.GetMilestone(projectSpec, msName)
		if ms == nil {
			continue
		}

		pipeState.Set(msName, state.StatusInProgress)
		pipeState.Save()

		var branchName string
		if p.Config.Branch != "" {
			branchName = p.Config.Branch
			if !git.IsClean() {
				p.logger.Warn("working tree not clean")
			}
		} else {
			branchName = msName + "-pipeline"
			if err := git.Checkout(defaultBranch); err != nil {
				p.logger.Error("checkout failed", "branch", defaultBranch, "error", err)
				return 1
			}
			if err := git.CreateBranch(branchName); err != nil {
				p.logger.Error("create branch failed", "branch", branchName, "error", err)
				return 1
			}
		}

		p.logger.Info("starting milestone", "name", msName, "branch", branchName)
		prompt := computeMilestoneSpec(ms, projectSpec.Milestones, pipeState, allHandoffNotes)
		p.logger.Info("running milestone", "name", msName, "prompt_bytes", len(prompt))

		result, err := session.Run(prompt, 0)
		if err != nil || result.ReturnCode != 0 {
			pipeState.Set(msName, state.StatusFailed)
			hasFailures = true
			rc := -1
			if result != nil {
				rc = result.ReturnCode
			}
			p.logger.Warn("session failed", "name", msName, "code", rc, "error", err)
		} else {
			pipeState.Set(msName, state.StatusCompleted)
		}
		pipeState.Save()

		if ms.Verify != nil {
			vResults, err := verify.Run(ms.Verify, 0)
			if err == nil {
				allVerifyResults = append(allVerifyResults, vResults...)
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

		review.Phase(msName, ms.Spec, deduped, allVerifyResults)

		if p.Config.Branch == "" {
			if err := git.Commit("Milestone " + msName); err != nil {
				p.logger.Error("commit failed", "error", err)
				return 1
			}
			if err := git.Tag(msName + "-done"); err != nil {
				p.logger.Warn("tag failed", "error", err)
			}
			if err := git.Checkout(defaultBranch); err != nil {
				p.logger.Error("checkout failed", "error", err)
				return 1
			}
			if err := git.SquashMerge(branchName); err != nil {
				p.logger.Error("squash merge failed", "error", err)
				return 1
			}
		}
	}

	var msInfos []context.MilestoneInfo
	for _, ms := range projectSpec.Milestones {
		msInfos = append(msInfos, context.MilestoneInfo{Name: ms.Name, Spec: ms.Spec})
	}
	review.Final(projectSpec.Project.Name, msInfos, allHandoffNotes, allVerifyResults)

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

func detectDefaultBranch() string {
	if b, err := git.CurrentBranch(); err == nil {
		return b
	}
	return "main"
}

func computeMilestoneSpec(ms *spec.Milestone, all []spec.Milestone, pipeState *state.State, handoffNotes []handoff.Note) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("# Milestone: %s\n", ms.Name))
	if ms.Spec != "" {
		parts = append(parts, "## Spec\n\n"+ms.Spec+"\n")
	}

	if len(ms.Tasks) > 0 {
		parts = append(parts, "## Tasks\n")
		for _, t := range ms.Tasks {
			parts = append(parts, fmt.Sprintf("### %s\n\n%s\n", t.Name, t.Prompt))
		}
	}

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
		if m.Name != ms.Name {
			remaining = append(remaining, m)
		}
	}
	if len(remaining) > 0 {
		parts = append(parts, "## Upcoming Milestones\n")
		for _, m := range remaining {
			specPreview := m.Spec
			if len(specPreview) > 80 {
				specPreview = specPreview[:80]
			}
			parts = append(parts, fmt.Sprintf("- %s: %s...\n", m.Name, specPreview))
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}
