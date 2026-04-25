package pipeline

import (
	"strings"
	"testing"

	"orchestrator/internal/spec"
)

func TestDetectDefaultBranch(t *testing.T) {
	branch := detectDefaultBranch()
	if branch == "" {
		t.Fatal("expected non-empty branch")
	}
}

func TestComputeMilestoneSpec(t *testing.T) {
	ms := &spec.Milestone{
		Name: "m1",
		Spec: "test spec",
		Tasks: []spec.TaskSpec{
			{Name: "t1", Prompt: "do something"},
		},
	}
	result := computeMilestoneSpec(ms, nil, nil, nil)
	if !strings.Contains(result, "m1") {
		t.Fatal("missing milestone name")
	}
	if !strings.Contains(result, "do something") {
		t.Fatal("missing task prompt")
	}
}
