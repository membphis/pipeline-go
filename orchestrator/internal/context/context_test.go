package context

import (
	"strings"
	"testing"

	"orchestrator/internal/handoff"
	"orchestrator/internal/verify"
)

func TestCountTokensEmpty(t *testing.T) {
	if CountTokens("") != 0 {
		t.Fatal("expected 0")
	}
}

func TestCountTokensSimple(t *testing.T) {
	if CountTokens("hello world") != 3 {
		t.Fatalf("expected 3, got %d", CountTokens("hello world"))
	}
}

func TestBuildBundleMilestones(t *testing.T) {
	bundle := BuildBundle([]MilestoneInfo{{Name: "m1", Spec: "spec"}}, nil, nil)
	if !strings.Contains(bundle, "m1") {
		t.Fatal("missing milestone name")
	}
}

func TestBuildBundleHandoff(t *testing.T) {
	notes := []handoff.Note{{Source: "path/HANDOFF.md", Content: "# Note"}}
	bundle := BuildBundle(nil, notes, nil)
	if !strings.Contains(bundle, "# Note") {
		t.Fatal("missing handoff content")
	}
}

func TestBuildBundleVerify(t *testing.T) {
	results := []verify.Result{{ReturnCode: 0, Stdout: "All good"}}
	bundle := BuildBundle(nil, nil, results)
	if !strings.Contains(bundle, "All good") {
		t.Fatal("missing verify result")
	}
}

func TestDegradeNoVerify(t *testing.T) {
	notes := []handoff.Note{{Source: "h.md", Content: "note"}}
	results := []verify.Result{{ReturnCode: 0, Stdout: "v"}}
	bundle := BuildBundle([]MilestoneInfo{{Name: "m1", Spec: "spec"}}, notes, results)
	degraded := Degrade(bundle, DegradeNoVerify)
	if strings.Contains(degraded, "v") {
		t.Fatal("verify results should be removed")
	}
	if !strings.Contains(degraded, "note") {
		t.Fatal("handoff should remain")
	}
}

func TestDegradeMinimal(t *testing.T) {
	notes := []handoff.Note{{Source: "h.md", Content: "note"}}
	results := []verify.Result{{ReturnCode: 0, Stdout: "v"}}
	bundle := BuildBundle([]MilestoneInfo{{Name: "m1", Spec: "spec"}}, notes, results)
	degraded := Degrade(bundle, DegradeMinimal)
	if strings.Contains(degraded, "note") || strings.Contains(degraded, "v") {
		t.Fatal("both should be removed")
	}
}
