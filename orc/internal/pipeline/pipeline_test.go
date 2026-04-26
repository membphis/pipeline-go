package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orc/internal/git"
	"orc/internal/spec"
	"orc/internal/state"
)

func TestBuildExecPrompt(t *testing.T) {
	ms := &spec.Milestone{
		ID:   "m1",
		Name: "Milestone One",
		Specs: []spec.SpecItem{
			{ID: "s1", Description: "test spec", TaskCount: 2, TestCount: 2, EstMinutes: 15, SpecFile: "specs/m1.md"},
		},
	}
	result := buildExecPrompt(ms, nil, nil, nil, ".", ".")
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
	if !strings.Contains(result, "PLAN.md") {
		t.Fatal("missing plan reference")
	}
}

func TestBuildPlanPrompt(t *testing.T) {
	ms := &spec.Milestone{
		ID:   "m1",
		Name: "Milestone One",
		Specs: []spec.SpecItem{
			{ID: "s1", Description: "test spec", TaskCount: 2, TestCount: 2, EstMinutes: 15, SpecFile: "specs/m1.md"},
		},
	}
	result := buildPlanPrompt(ms, ".", ".")
	if !strings.Contains(result, "Milestone One") {
		t.Fatal("missing milestone name")
	}
	if !strings.Contains(result, "Write Only") {
		t.Fatal("missing Write Only instruction")
	}
	if !strings.Contains(result, "PLAN.md") {
		t.Fatal("missing plan path")
	}
	if strings.Contains(result, "Phase 2") {
		t.Fatal("plan prompt should not contain Phase 2+")
	}
}

func TestEnsureRootNewProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	rootDir := filepath.Join(tmpDir, "new-project")
	statePath := filepath.Join(tmpDir, "state.yaml")

	p := New(Config{
		SpecPath: "proj.yaml",
		Root:     rootDir,
	})

	pipeState := state.New([]string{"m1"}, statePath)
	if err := p.ensureRoot(pipeState); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, ".git")); err != nil {
		t.Fatal("expected .git to exist")
	}
}

func TestEnsureRootExistingEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	rootDir := filepath.Join(tmpDir, "empty-dir")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(tmpDir, "state.yaml")
	p := New(Config{SpecPath: "proj.yaml", Root: rootDir})
	pipeState := state.New([]string{"m1"}, statePath)

	if err := p.ensureRoot(pipeState); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, ".git")); err != nil {
		t.Fatal("expected .git to exist")
	}
}

func TestEnsureRootExistingNonEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	rootDir := filepath.Join(tmpDir, "non-empty")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(rootDir, "some-file"), []byte{}, 0644)

	statePath := filepath.Join(tmpDir, "state.yaml")
	p := New(Config{SpecPath: "proj.yaml", Root: rootDir})
	pipeState := state.New([]string{"m1"}, statePath)

	if err := p.ensureRoot(pipeState); err == nil {
		t.Fatal("expected error for non-empty root")
	}
}

func TestEnsureRootInProgress(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	rootDir := filepath.Join(tmpDir, "existing-project")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(tmpDir, "state.yaml")

	if err := git.RepoInit(rootDir); err != nil {
		t.Fatal(err)
	}
	if err := git.InitCommit(rootDir); err != nil {
		t.Fatal(err)
	}

	pipeState := state.New([]string{"m1"}, statePath)
	pipeState.Set("m1", state.StatusCompleted)
	pipeState.Save()

	p := New(Config{SpecPath: "proj.yaml", Root: rootDir})
	if err := p.ensureRoot(pipeState); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureRootInProgressNoGit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	rootDir := filepath.Join(tmpDir, "no-git-dir")
	os.MkdirAll(rootDir, 0755)

	statePath := filepath.Join(tmpDir, "state.yaml")
	pipeState := state.New([]string{"m1"}, statePath)
	pipeState.Set("m1", state.StatusCompleted)
	pipeState.Save()

	p := New(Config{SpecPath: "proj.yaml", Root: rootDir})
	if err := p.ensureRoot(pipeState); err == nil {
		t.Fatal("expected error for non-git root with in-progress state")
	}
}

func TestEnsureRootPathResolution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "state.yaml")

	p := New(Config{
		SpecPath: "proj.yaml",
		Root:     tmpDir + "/relative-subdir",
	})

	pipeState := state.New([]string{"m1"}, statePath)
	if err := p.ensureRoot(pipeState); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "relative-subdir", ".git")); err != nil {
		t.Fatal("expected .git to exist in relative path")
	}
}
