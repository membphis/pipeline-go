package spec

import (
	"testing"
)

func TestLoadSuccess(t *testing.T) {
	s, err := Load("../../../sample-project/project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if s.Project.Name != "api-gateway" {
		t.Fatalf("unexpected project name: %s", s.Project.Name)
	}
	if len(s.Milestones) != 4 {
		t.Fatalf("expected 4 milestones, got %d", len(s.Milestones))
	}
}

func TestPreflightSuccess(t *testing.T) {
	s, err := Load("../../../sample-project/project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	errs := Preflight(s)
	if len(errs) > 0 {
		t.Fatalf("expected 0 errors, got %v", errs)
	}
}

func TestGetMilestone(t *testing.T) {
	s, err := Load("../../../sample-project/project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ms := GetMilestone(s, "路由层")
	if ms == nil {
		t.Fatal("expected milestone")
	}
	if ms.Name != "路由层" {
		t.Fatalf("unexpected name: %s", ms.Name)
	}
}

func TestGetMilestoneNotFound(t *testing.T) {
	s, err := Load("../../../sample-project/project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if GetMilestone(s, "nonexistent") != nil {
		t.Fatal("expected nil")
	}
}

func TestPreflightDuplicate(t *testing.T) {
	s := &Spec{
		Project: Project{Name: "test"},
		Milestones: []Milestone{
			{Name: "m1", Tasks: []TaskSpec{{Name: "t1", Prompt: "p"}}},
			{Name: "m1", Tasks: []TaskSpec{{Name: "t2", Prompt: "p"}}},
		},
	}
	errs := Preflight(s)
	if len(errs) == 0 {
		t.Fatal("expected duplicate error")
	}
}

func TestPreflightMissingDep(t *testing.T) {
	s := &Spec{
		Project: Project{Name: "test"},
		Milestones: []Milestone{
			{Name: "m1", DependsOn: []string{"nonexistent"}, Tasks: []TaskSpec{{Name: "t1", Prompt: "p"}}},
		},
	}
	errs := Preflight(s)
	if len(errs) == 0 || errs[0] != "milestone \"m1\" has unknown dependency: \"nonexistent\"" {
		t.Fatalf("unexpected: %v", errs)
	}
}

func TestPreflightEmptyTasks(t *testing.T) {
	s := &Spec{
		Project:    Project{Name: "test"},
		Milestones: []Milestone{{Name: "m1"}},
	}
	errs := Preflight(s)
	if len(errs) == 0 {
		t.Fatal("expected empty tasks error")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
}
