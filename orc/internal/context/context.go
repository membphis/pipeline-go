package context

import (
	"math"
	"strings"

	"orc/internal/handoff"
	"orc/internal/verify"
)

type DegradeStrategy string

const (
	DegradeNoVerify  DegradeStrategy = "no_verify"
	DegradeNoHandoff DegradeStrategy = "no_handoff"
	DegradeMinimal   DegradeStrategy = "minimal"
)

const TokenEstimateRatio = 4
const MaxTokens = 180_000

func CountTokens(text string) int {
	return int(math.Ceil(float64(len(text)) / TokenEstimateRatio))
}

type MilestoneInfo struct {
	Name string
	Spec string
}

func BuildBundle(milestones []MilestoneInfo, handoffNotes []handoff.Note, verifyResults []verify.Result) string {
	var parts []string

	if len(milestones) > 0 {
		parts = append(parts, "## Pipeline State\n")
		for _, ms := range milestones {
			parts = append(parts, "### Milestone: "+ms.Name+"\n\n"+ms.Spec+"\n")
		}
	}

	if len(handoffNotes) > 0 {
		parts = append(parts, "## Handoff Notes\n")
		for _, n := range handoffNotes {
			parts = append(parts, "### From: "+n.Source+"\n\n"+n.Content+"\n")
		}
	}

	if len(verifyResults) > 0 {
		parts = append(parts, "## Verify Results\n")
		for _, r := range verifyResults {
			status := "PASS"
			if r.ReturnCode != 0 {
				status = "FAIL"
			}
			parts = append(parts, "- "+status+": "+r.Stdout)
			if r.Stderr != "" {
				parts = append(parts, "  stderr: "+r.Stderr)
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func ExceedsThreshold(text string, maxTokens int) bool {
	if maxTokens <= 0 {
		maxTokens = MaxTokens
	}
	return CountTokens(text) > maxTokens
}

func Degrade(bundle string, strategy DegradeStrategy) string {
	lines := strings.Split(bundle, "\n")
	var filtered []string
	skipSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if (strategy == DegradeNoVerify || strategy == DegradeMinimal) && trimmed == "## Verify Results" {
			skipSection = true
			continue
		}
		if (strategy == DegradeNoHandoff || strategy == DegradeMinimal) && trimmed == "## Handoff Notes" {
			skipSection = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && trimmed != "## Pipeline State" {
			skipSection = false
			continue
		}
		if strings.HasPrefix(trimmed, "## Pipeline State") {
			skipSection = false
		}
		if !skipSection {
			filtered = append(filtered, line)
		}
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}
