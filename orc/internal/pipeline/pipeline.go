package pipeline

import (
	"fmt"
	"log/slog"
	"os"
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

		pipeState.Set(msID, state.StatusInProgress)
		pipeState.Save()

		p.logger.Info("starting milestone", "name", msID)
		prompt := computeMilestoneSpec(ms, projectSpec.Milestones, pipeState, allHandoffNotes, p.Config.Root)
		p.logger.Info("running milestone", "name", msID, "prompt_bytes", len(prompt))

		result, err := session.Run(prompt, 0)
		if err != nil || result.ReturnCode != 0 {
			pipeState.Set(msID, state.StatusFailed)
			hasFailures = true
			rc := -1
			if result != nil {
				rc = result.ReturnCode
			}
			p.logger.Warn("session failed", "name", msID, "code", rc, "error", err)
		} else {
			pipeState.Set(msID, state.StatusCompleted)
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

		review.Phase(msID, buildSpecContent(ms, p.Config.Root), deduped, allVerifyResults)
	}

	var msInfos []context.MilestoneInfo
	for _, ms := range projectSpec.Milestones {
		msInfos = append(msInfos, context.MilestoneInfo{Name: ms.Name, Spec: buildSpecContent(&ms, p.Config.Root)})
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

func computeMilestoneSpec(ms *spec.Milestone, all []spec.Milestone, pipeState *state.State, handoffNotes []handoff.Note, rootDir string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("# Milestone: %s\n", ms.Name))

	if len(ms.Specs) > 0 {
		parts = append(parts, "## Specs\n")
		for _, s := range ms.Specs {
			parts = append(parts, fmt.Sprintf("### %s: %s\n", s.ID, s.Description))
			parts = append(parts, fmt.Sprintf("- Tasks: %d | Est: %d min\n", s.TaskCount, s.EstMinutes))
			if s.TestCount > 0 {
				parts = append(parts, fmt.Sprintf("- Tests: %d\n", s.TestCount))
			}
			if s.SpecFile != "" {
				specPath := filepath.Join(rootDir, s.SpecFile)
				data, err := os.ReadFile(specPath)
				if err == nil {
					parts = append(parts, "\n"+string(data)+"\n")
				}
			}
		}
	}

	if hasTests(ms.Specs) {
		parts = append(parts, "## Development Process\n")
		parts = append(parts, "Follow Test-Driven Development (TDD):\n")
		parts = append(parts, "1. Write failing tests first\n")
		parts = append(parts, "2. Verify they fail correctly\n")
		parts = append(parts, "3. Write minimal implementation to pass\n")
		parts = append(parts, "4. Verify all tests pass\n")
		parts = append(parts, "5. Refactor while keeping tests green\n\n")
		for _, s := range ms.Specs {
			if s.TestCount > 0 {
				parts = append(parts, fmt.Sprintf("For %s: generate %d test cases before implementation.\n", s.ID, s.TestCount))
			}
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
		if m.ID != ms.ID {
			remaining = append(remaining, m)
		}
	}
	if len(remaining) > 0 {
		parts = append(parts, "## Upcoming Milestones\n")
		for _, m := range remaining {
			var descParts []string
			for _, s := range m.Specs {
				descParts = append(descParts, s.Description)
			}
			preview := strings.Join(descParts, "; ")
			if len([]rune(preview)) > 80 {
				preview = string([]rune(preview)[:80])
			}
			parts = append(parts, fmt.Sprintf("- %s: %s...\n", m.Name, preview))
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func buildSpecContent(ms *spec.Milestone, rootDir string) string {
	var parts []string
	for _, s := range ms.Specs {
		parts = append(parts, fmt.Sprintf("### %s: %s\n", s.ID, s.Description))
		parts = append(parts, fmt.Sprintf("- Tasks: %d | Est: %d min\n", s.TaskCount, s.EstMinutes))
		if s.TestCount > 0 {
			parts = append(parts, fmt.Sprintf("- Tests: %d\n", s.TestCount))
		}
		if s.SpecFile != "" {
			specPath := filepath.Join(rootDir, s.SpecFile)
			data, err := os.ReadFile(specPath)
			if err == nil {
				parts = append(parts, "\n"+string(data)+"\n")
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func hasTests(specs []spec.SpecItem) bool {
	for _, s := range specs {
		if s.TestCount > 0 {
			return true
		}
	}
	return false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
