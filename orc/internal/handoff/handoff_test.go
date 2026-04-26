package handoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectNone(t *testing.T) {
	notes, err := Collect(os.TempDir() + "/nonexistent-xyz")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected 0, got %d", len(notes))
	}
}

func TestCollectSingle(t *testing.T) {
	dir, _ := os.MkdirTemp("", "handoff-*")
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "HANDOFF.md"), []byte("# Note\n\nBody"), 0644)

	notes, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1, got %d", len(notes))
	}
	if !strings.Contains(notes[0].Content, "Body") {
		t.Fatal("missing content")
	}
}

func TestCollectSkipsNonHandoff(t *testing.T) {
	dir, _ := os.MkdirTemp("", "handoff-*")
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0644)
	os.WriteFile(filepath.Join(dir, "HANDOFF.md"), []byte("handoff"), 0644)

	notes, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1, got %d", len(notes))
	}
}

func TestFormatNotes(t *testing.T) {
	result := FormatNotes(nil)
	if result != "" {
		t.Fatalf("expected empty, got %q", result)
	}

	notes := []Note{{Source: "a/HANDOFF.md", Content: "First"}}
	result = FormatNotes(notes)
	if !strings.Contains(result, "First") {
		t.Fatal("missing content in formatted output")
	}
}

func TestCollectMultiple(t *testing.T) {
	dir, _ := os.MkdirTemp("", "handoff-*")
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "HANDOFF-m1.md"), []byte("Note 1"), 0644)
	os.WriteFile(filepath.Join(dir, "HANDOFF-m2.md"), []byte("Note 2"), 0644)

	notes, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2, got %d", len(notes))
	}
}

func TestCollectSkipsNonHandoffPrefix(t *testing.T) {
	dir, _ := os.MkdirTemp("", "handoff-*")
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0644)
	os.WriteFile(filepath.Join(dir, "PLAN.md"), []byte("plan"), 0644)
	os.WriteFile(filepath.Join(dir, "HANDOFF-m1.md"), []byte("handoff"), 0644)

	notes, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1, got %d", len(notes))
	}
}
