package verify

import (
	"testing"
)

func TestRunString(t *testing.T) {
	results, err := Run("echo ok", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ReturnCode != 0 {
		t.Fatalf("expected 0, got %d", results[0].ReturnCode)
	}
	if results[0].Stdout != "ok\n" {
		t.Fatalf("expected 'ok\\n', got %q", results[0].Stdout)
	}
}

func TestRunList(t *testing.T) {
	results, err := Run([]string{"echo a", "echo b"}, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success() {
			t.Fatal("expected all to succeed")
		}
	}
}

func TestRunNil(t *testing.T) {
	results, err := Run(nil, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil, got %v", results)
	}
}

func TestRunFailure(t *testing.T) {
	results, err := Run("false", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success() {
		t.Fatal("expected failure")
	}
}

func TestAllSuccessful(t *testing.T) {
	if !AllSuccessful([]Result{{ReturnCode: 0}, {ReturnCode: 0}}) {
		t.Fatal("expected true")
	}
	if AllSuccessful([]Result{{ReturnCode: 0}, {ReturnCode: 1}}) {
		t.Fatal("expected false")
	}
}

func TestResultDataclass(t *testing.T) {
	r := Result{ReturnCode: 0, Stdout: "out", Stderr: "err"}
	if !r.Success() {
		t.Fatal("expected success")
	}
}
