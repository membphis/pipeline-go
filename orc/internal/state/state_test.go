package state

import (
	"bytes"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestNewStatePending(t *testing.T) {
	s := New([]string{"m1", "m2"}, "")
	if s.Milestones["m1"].Status != StatusPending {
		t.Fatalf("expected pending, got %s", s.Milestones["m1"].Status)
	}
}

func TestSetAndGet(t *testing.T) {
	s := New([]string{"m1"}, "")
	if err := s.Set("m1", StatusCompleted); err != nil {
		t.Fatal(err)
	}
	st, _ := s.Get("m1")
	if st != StatusCompleted {
		t.Fatalf("expected completed, got %s", st)
	}
}

func TestInvalidStatus(t *testing.T) {
	s := New([]string{"m1"}, "")
	err := s.Set("m1", "invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAllCompleted(t *testing.T) {
	s := New([]string{"m1", "m2"}, "")
	if s.AllCompleted() {
		t.Fatal("expected false")
	}
	s.Set("m1", StatusCompleted)
	s.Set("m2", StatusCompleted)
	if !s.AllCompleted() {
		t.Fatal("expected true")
	}
}

func TestSaveAndLoad(t *testing.T) {
	f, _ := os.CreateTemp("", "state-*.yaml")
	defer os.Remove(f.Name())
	s := New([]string{"m1"}, f.Name())
	s.Set("m1", StatusCompleted)
	s.Save()

	s2 := New([]string{"m1"}, f.Name())
	st, _ := s2.Get("m1")
	if st != StatusCompleted {
		t.Fatalf("expected completed after load, got %s", st)
	}
}

func TestGetAll(t *testing.T) {
	s := New([]string{"m1", "m2"}, "")
	s.Set("m1", StatusCompleted)
	all := s.GetAll()
	if all["m1"] != StatusCompleted || all["m2"] != StatusPending {
		t.Fatal("unexpected statuses")
	}
}

func TestTimestampIsHumanReadable(t *testing.T) {
	s := New([]string{"m1"}, "")
	s.Set("m1", StatusCompleted)
	data, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("T")) {
		t.Fatalf("expected RFC3339 human-readable timestamp, got:\n%s", data)
	}
	var loaded struct {
		Milestones map[string]struct {
			Status    string     `yaml:"status"`
			Timestamp *time.Time `yaml:"timestamp,omitempty"`
		} `yaml:"milestones"`
	}
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	ts := loaded.Milestones["m1"].Timestamp
	if ts == nil {
		t.Fatal("timestamp should not be nil after round-trip")
	}
	if time.Since(*ts) > 10*time.Second {
		t.Fatalf("timestamp should be recent, got %v", ts)
	}
}
