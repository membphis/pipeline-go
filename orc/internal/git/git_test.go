package git

import (
	"os"
	"strings"
	"testing"
)

func TestRepoInit(t *testing.T) {
	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := RepoInit(dir); err != nil {
		t.Fatal(err)
	}

	out, err := RunInDir(dir, "rev-parse", "--is-bare-repository")
	if err != nil {
		t.Fatal(err)
	}
	if out != "false" {
		t.Fatalf("expected non-bare repo, got %q", out)
	}
}

func TestInitCommit(t *testing.T) {
	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := RepoInit(dir); err != nil {
		t.Fatal(err)
	}
	if err := InitCommit(dir); err != nil {
		t.Fatal(err)
	}

	out, err := RunInDir(dir, "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "init") {
		t.Fatalf("expected commit message 'init', got %q", out)
	}

	// Verify scaffold files
	if _, err := os.Stat(dir + "/README.md"); err != nil {
		t.Fatal("expected README.md to exist")
	}
	if _, err := os.Stat(dir + "/.gitignore"); err != nil {
		t.Fatal("expected .gitignore to exist")
	}
	if _, err := os.Stat(dir + "/.orc_history/.gitignore"); err != nil {
		t.Fatal("expected .orc_history/.gitignore to exist")
	}
}

func TestTag(t *testing.T) {
	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := RepoInit(dir); err != nil {
		t.Fatal(err)
	}
	if err := InitCommit(dir); err != nil {
		t.Fatal(err)
	}
	if err := Tag(dir, "test-v1.0"); err != nil {
		t.Fatal(err)
	}

	out, err := RunInDir(dir, "tag")
	if err != nil {
		t.Fatal(err)
	}
	if out != "test-v1.0" {
		t.Fatalf("expected tag 'test-v1.0', got %q", out)
	}
}
