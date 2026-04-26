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
			{ID: "s1", Description: "test spec", TaskCount: 2, TestCount: 2, EstMinutes: 15},
		},
	}
	result := computeMilestoneSpec(ms, nil, nil, nil, ".")
	if !strings.Contains(result, "Milestone One") {
		t.Fatal("missing milestone name")
	}
	if !strings.Contains(result, "test spec") {
		t.Fatal("missing spec description")
	}
	if !strings.Contains(result, "generate 2 test cases") {
		t.Fatal("missing TDD instruction")
	}
}
