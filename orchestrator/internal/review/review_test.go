package review

import (
	"strings"
	"testing"

	"orchestrator/internal/context"
)

func TestBuildReviewPrompt(t *testing.T) {
	bundle, prompt := buildReviewPrompt("phase", "m1", []context.MilestoneInfo{{Name: "m1", Spec: "spec1"}}, nil, nil)
	if !strings.Contains(prompt, "m1") {
		t.Fatal("missing milestone name")
	}
	if !strings.Contains(prompt, "phase") {
		t.Fatal("missing review type")
	}
	if bundle == "" {
		t.Fatal("expected non-empty bundle")
	}
}
