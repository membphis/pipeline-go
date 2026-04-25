package topo

import (
	"testing"
)

func TestEmptyMilestones(t *testing.T) {
	result, err := Sort(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestSingleMilestone(t *testing.T) {
	result, err := Sort([]Milestone{{Name: "m1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0] != "m1" {
		t.Fatalf("unexpected order: %v", result)
	}
}

func TestSimpleChain(t *testing.T) {
	ms := []Milestone{
		{Name: "m1"},
		{Name: "m2", DependsOn: []string{"m1"}},
		{Name: "m3", DependsOn: []string{"m2"}},
	}
	result, err := Sort(ms)
	if err != nil {
		t.Fatal(err)
	}
	idx := make(map[string]int)
	for i, name := range result {
		idx[name] = i
	}
	if !(idx["m1"] < idx["m2"] && idx["m2"] < idx["m3"]) {
		t.Fatalf("bad order: %v", result)
	}
}

func TestCycleDetection(t *testing.T) {
	ms := []Milestone{
		{Name: "m1", DependsOn: []string{"m2"}},
		{Name: "m2", DependsOn: []string{"m1"}},
	}
	_, err := Sort(ms)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	_, ok := err.(*CycleError)
	if !ok {
		t.Fatalf("expected *CycleError, got %T", err)
	}
}

func TestSelfCycle(t *testing.T) {
	ms := []Milestone{
		{Name: "m1", DependsOn: []string{"m1"}},
	}
	_, err := Sort(ms)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestDiamond(t *testing.T) {
	ms := []Milestone{
		{Name: "m1"},
		{Name: "m2", DependsOn: []string{"m1"}},
		{Name: "m3", DependsOn: []string{"m1"}},
		{Name: "m4", DependsOn: []string{"m2", "m3"}},
	}
	result, err := Sort(ms)
	if err != nil {
		t.Fatal(err)
	}
	idx := make(map[string]int)
	for i, name := range result {
		idx[name] = i
	}
	if !(idx["m1"] < idx["m2"] && idx["m1"] < idx["m3"] && idx["m2"] < idx["m4"] && idx["m3"] < idx["m4"]) {
		t.Fatalf("bad order: %v", result)
	}
}
