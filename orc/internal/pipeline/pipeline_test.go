package pipeline

import (
	"strings"
	"testing"

	"orc/internal/spec"
)

func TestComputeMilestoneSpec(t *testing.T) {
	ms := &spec.Milestone{
		ID:   "m1",
		Name: "Milestone One",
		Specs: []spec.SpecItem{
			{ID: "s1", Description: "test spec", TaskCount: 2, TestCount: 2, EstMinutes: 15, SpecFile: "specs/m1.md"},
		},
	}
	result := computeMilestoneSpec(ms, nil, nil, nil, ".")
	if !strings.Contains(result, "Milestone One") {
		t.Fatal("missing milestone name")
	}
	if !strings.Contains(result, "s1") {
		t.Fatal("missing spec id")
	}
	if !strings.Contains(result, "specs/m1.md") {
		t.Fatal("missing spec file reference")
	}
	if !strings.Contains(result, "Write tests first") {
		t.Fatal("missing TDD instruction")
	}
}
