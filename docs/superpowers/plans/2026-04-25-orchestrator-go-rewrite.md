# Orchestrator Go Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the Python orchestrator to Go, producing a single standalone binary with identical functionality.

**Architecture:** `main.go` + `internal/{log,topo,spec,state,git,session,verify,handoff,context,review,pipeline}/` packages. Each package maps 1:1 to a Python module. Dependencies: `gopkg.in/yaml.v3` for YAML, standard library for everything else.

**Tech Stack:** Go 1.22+, `gopkg.in/yaml.v3`, standard library `os/exec`, `log/slog`, `flag`, `testing`.

---

### Task 1: Initialize Go module and root files

**Files:**
- Create: `orchestrator/go.mod`
- Create: `orchestrator/go.sum`
- Create: `orchestrator/.gitignore`

- [ ] **Set up Go module**

```bash
cd /home/rain/git/pipeline-002/orchestrator
go mod init orchestrator
go get gopkg.in/yaml.v3
```

- [ ] **Update .gitignore**

Replace Python-specific entries with Go ones in `/home/rain/git/pipeline-002/.gitignore`:

```
__pycache__/
*.pyc
.pytest_cache/
.opencode/logs/
.opencode/.tmp/
state.yaml
/dist/
.venv/
*.egg-info/

# Go
/bin/
/tmp/
```

- [ ] **Commit**

```bash
git add orchestrator/go.mod orchestrator/go.sum .gitignore
git commit -m "feat: init Go module with yaml.v3 dependency"
```

---

### Task 2: log package

**Files:**
- Create: `orchestrator/internal/log/log.go`

The logging package wraps `log/slog` to match Python's `orchestrator.log` interface:
- `log.Setup()` — configures structured JSON logger to stderr
- `log.Get(name)` — returns a `*slog.Logger` with `orchestrator.{name}` as the source

- [ ] **Write log package**

```go
package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu     sync.Mutex
	logger *slog.Logger
)

func Setup() {
	mu.Lock()
	defer mu.Unlock()
	if logger != nil {
		return
	}
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

func Get(name string) *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		Setup()
	}
	return logger.With("pkg", strings.TrimPrefix(name, "orchestrator."))
}

type contextKey struct{}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

func FromContext(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(contextKey{}).(*slog.Logger)
	if !ok {
		return Get("")
	}
	return l
}
```

- [ ] **Verify it compiles**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go build ./internal/log/`
Expected: no output

- [ ] **Commit**

```bash
git add orchestrator/internal/log/log.go
git commit -m "feat: add log package wrapping slog"
```

---

### Task 3: topo package

**Files:**
- Create: `orchestrator/internal/topo/topo.go`
- Create: `orchestrator/internal/topo/topo_test.go`

DFS topological sort with cycle detection. 1:1 port of Python `topo.py`.

- [ ] **Write topo.go**

```go
package topo

import "fmt"

type CycleError struct {
	Cycle []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("circular dependency detected: %s", e.Cycle)
}

type Milestone struct {
	Name      string
	DependsOn []string
}

func Sort(milestones []Milestone) ([]string, error) {
	graph := make(map[string][]string)
	allNames := make(map[string]bool)

	for _, ms := range milestones {
		allNames[ms.Name] = true
		graph[ms.Name] = ms.DependsOn
	}

	visited := make(map[string]bool)
	inProgress := make(map[string]bool)
	order := make([]string, 0, len(milestones))

	var visit func(node string) error
	visit = func(node string) error {
		if inProgress[node] {
			cycle := []string{node}
			for _, n := range order {
				cycle = append(cycle, n)
				if n == node {
					break
				}
			}
			return &CycleError{Cycle: cycle}
		}
		if visited[node] {
			return nil
		}
		inProgress[node] = true
		for _, dep := range graph[node] {
			if allNames[dep] {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		delete(inProgress, node)
		visited[node] = true
		order = append(order, node)
		return nil
	}

	for _, ms := range milestones {
		if !visited[ms.Name] {
			if err := visit(ms.Name); err != nil {
				return nil, err
			}
		}
	}
	return order, nil
}
```

- [ ] **Write topo_test.go**

```go
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
```

- [ ] **Run tests**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go test ./internal/topo/ -v`
Expected: all pass

- [ ] **Commit**

```bash
git add orchestrator/internal/topo/ && git commit -m "feat: add topo sort with cycle detection"
```

---

### Task 4: spec package

**Files:**
- Create: `orchestrator/internal/spec/spec.go`
- Create: `orchestrator/internal/spec/spec_test.go`

YAML spec loading and preflight validation. Maps to Python `spec.py`.

- [ ] **Write spec.go**

```go
package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Project struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
}

type TaskSpec struct {
	Name   string `yaml:"name"`
	Prompt string `yaml:"prompt"`
}

type Milestone struct {
	Name      string     `yaml:"name"`
	Spec      string     `yaml:"spec,omitempty"`
	DependsOn []string   `yaml:"depends_on,omitempty"`
	Tasks     []TaskSpec `yaml:"tasks"`
	Verify    []string   `yaml:"verify,omitempty"`
}

type Spec struct {
	Project    Project     `yaml:"project"`
	Milestones []Milestone `yaml:"milestones"`
}

func Load(path string, extraSpecs ...string) (*Spec, error) {
	var spec Spec
	files := append([]string{path}, extraSpecs...)
	for i, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var partial Spec
		if err := yaml.Unmarshal(data, &partial); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if i == 0 {
			spec = partial
		} else {
			spec.Milestones = append(spec.Milestones, partial.Milestones...)
		}
	}
	if spec.Project.Name == "" {
		return nil, fmt.Errorf("spec missing required key: project")
	}
	if len(spec.Milestones) == 0 {
		return nil, fmt.Errorf("spec missing required key: milestones")
	}
	return &spec, nil
}

func GetMilestone(s *Spec, name string) *Milestone {
	for i := range s.Milestones {
		if s.Milestones[i].Name == name {
			return &s.Milestones[i]
		}
	}
	return nil
}

func Preflight(s *Spec) []string {
	var errors []string
	names := make(map[string]bool)
	for _, ms := range s.Milestones {
		if ms.Name == "" {
			errors = append(errors, "milestone without name")
			continue
		}
		if names[ms.Name] {
			errors = append(errors, fmt.Sprintf("duplicate milestone name: %s", ms.Name))
		}
		names[ms.Name] = true
		if len(ms.Tasks) == 0 {
			errors = append(errors, fmt.Sprintf("milestone %q has no tasks", ms.Name))
		}
	}
	for _, ms := range s.Milestones {
		for _, dep := range ms.DependsOn {
			if !names[dep] {
				errors = append(errors, fmt.Sprintf("milestone %q has unknown dependency: %q", ms.Name, dep))
			}
		}
	}
	return errors
}
```

- [ ] **Write spec_test.go**

```go
package spec

import (
	"testing"
)

func TestLoadSuccess(t *testing.T) {
	s, err := Load("../../sample-project/project.yaml")
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
	s, err := Load("../../sample-project/project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	errs := Preflight(s)
	if len(errs) > 0 {
		t.Fatalf("expected 0 errors, got %v", errs)
	}
}

func TestGetMilestone(t *testing.T) {
	s, err := Load("../../sample-project/project.yaml")
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
	s, err := Load("../../sample-project/project.yaml")
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
```

- [ ] **Run tests**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go test ./internal/spec/ -v`
Expected: all pass

- [ ] **Commit**

```bash
git add orchestrator/internal/spec/ && git commit -m "feat: add spec loading and preflight"
```

---

### Task 5: state package

**Files:**
- Create: `orchestrator/internal/state/state.go`
- Create: `orchestrator/internal/state/state_test.go`

Persists milestone statuses to `state.yaml`. 1:1 port of Python `state.py`.

- [ ] **Write state.go**

```go
package state

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusFailed:
		return true
	}
	return false
}

type MilestoneInfo struct {
	Status    Status  `yaml:"status"`
	Timestamp *float64 `yaml:"timestamp,omitempty"`
}

type State struct {
	Milestones map[string]MilestoneInfo `yaml:"milestones"`
	path       string
}

func New(milestoneNames []string, path string) *State {
	s := &State{
		Milestones: make(map[string]MilestoneInfo, len(milestoneNames)),
		path:       path,
	}
	for _, name := range milestoneNames {
		s.Milestones[name] = MilestoneInfo{Status: StatusPending}
	}
	s.load()
	return s
}

func (s *State) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var loaded State
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return
	}
	for name, info := range loaded.Milestones {
		if _, ok := s.Milestones[name]; ok {
			s.Milestones[name] = info
		}
	}
}

func (s *State) Get(milestone string) (Status, error) {
	info, ok := s.Milestones[milestone]
	if !ok {
		return "", fmt.Errorf("unknown milestone: %s", milestone)
	}
	return info.Status, nil
}

func (s *State) Set(milestone string, status Status) error {
	if !status.Valid() {
		return fmt.Errorf("invalid status %q; must be one of: pending, in_progress, completed, failed", status)
	}
	if _, ok := s.Milestones[milestone]; !ok {
		return fmt.Errorf("unknown milestone: %s", milestone)
	}
	now := float64(time.Now().Unix())
	s.Milestones[milestone] = MilestoneInfo{
		Status:    status,
		Timestamp: &now,
	}
	return nil
}

func (s *State) GetAll() map[string]Status {
	result := make(map[string]Status, len(s.Milestones))
	for name, info := range s.Milestones {
		result[name] = info.Status
	}
	return result
}

func (s *State) IsCompleted(milestone string) bool {
	info, ok := s.Milestones[milestone]
	return ok && info.Status == StatusCompleted
}

func (s *State) AllCompleted() bool {
	for _, info := range s.Milestones {
		if info.Status != StatusCompleted {
			return false
		}
	}
	return true
}

func (s *State) Save() error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
```

- [ ] **Write state_test.go**

```go
package state

import (
	"os"
	"testing"
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

func TestInvalidStatus(t *testing.T) {
	s := New([]string{"m1"}, "")
	err := s.Set("m1", "invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Run tests**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go test ./internal/state/ -v`
Expected: all pass

- [ ] **Commit**

```bash
git add orchestrator/internal/state/ && git commit -m "feat: add state persistence"
```

---

### Task 6: git package

**Files:**
- Create: `orchestrator/internal/git/git.go`
- Create: `orchestrator/internal/git/git_test.go`

Git operations via `os/exec`. Uses a package variable `var execCommand = exec.Command` for testability.

- [ ] **Write git.go**

```go
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func run(args ...string) (string, error) {
	cmd := execCommand("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func CurrentBranch() (string, error) {
	out, err := run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", fmt.Errorf("detached HEAD state")
	}
	return out, nil
}

func CreateBranch(name string, base ...string) error {
	args := []string{"checkout", "-b", name}
	if len(base) > 0 && base[0] != "" {
		args = append(args, base[0])
	}
	_, err := run(args...)
	return err
}

func Commit(message string) error {
	_, err := run("commit", "-m", message)
	return err
}

func Checkout(branch string) error {
	_, err := run("checkout", branch)
	return err
}

func SquashMerge(branch string) error {
	if _, err := run("merge", "--squash", branch); err != nil {
		return err
	}
	_, err := run("commit", "-m", fmt.Sprintf("Squash merge %s", branch))
	return err
}

func Tag(name string, message ...string) error {
	if len(message) > 0 && message[0] != "" {
		_, err := run("tag", "-a", name, "-m", message[0])
		return err
	}
	_, err := run("tag", name)
	return err
}

func IsClean() bool {
	out, err := run("status", "--porcelain")
	return err == nil && out == ""
}

func HasUnpushedCommits() bool {
	out, err := run("rev-list", "--count", "@{u}..HEAD")
	if err != nil {
		return false
	}
	return out != "0"
}

func IsDetachedHead() bool {
	out, err := run("rev-parse", "--abbrev-ref", "HEAD")
	return err == nil && out == "HEAD"
}
```

- [ ] **Write git_test.go**

```go
package git

import (
	"os/exec"
	"testing"
)

func TestCurrentBranch(t *testing.T) {
	old := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("echo", "main")
	}
	defer func() { execCommand = old }()

	branch, err := CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(branch)
}
```

- [ ] **Verify compilation**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go build ./internal/git/`
Expected: no output

- [ ] **Commit**

```bash
git add orchestrator/internal/git/ && git commit -m "feat: add git operations"
```

---

### Task 7: session package

**Files:**
- Create: `orchestrator/internal/session/session.go`
- Create: `orchestrator/internal/session/session_test.go`

Runs opencode as a subprocess. Uses `var execCommand = exec.Command` for testability.

- [ ] **Write session.go**

```go
package session

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var execCommand = exec.Command

type Result struct {
	ReturnCode int
	Stdout     string
	Stderr     string
}

func Run(prompt string, timeout time.Duration) (*Result, error) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return nil, fmt.Errorf("opencode not found on $PATH: install via 'pip install opencode' or download from https://opencode.ai")
	}
	cmd := execCommand(path, prompt)
	cmd.Stdin = os.Stdin

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting opencode: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var waitErr error
	if timeout > 0 {
		select {
		case waitErr = <-done:
		case <-time.After(timeout):
			cmd.Process.Kill()
			waitErr = <-done
			return &Result{
				ReturnCode: -9,
				Stdout:     stdout.String(),
				Stderr:     stderr.String(),
			}, nil
		}
	} else {
		waitErr = <-done
	}

	return &Result{
		ReturnCode: cmd.ProcessState.ExitCode(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, waitErr
}

func ErrNotFound() bool {
	var e *exec.Error
	return errors.As(&exec.Error{}, &e) && errors.Is(e, exec.ErrNotFound)
}
```

- [ ] **Write session_test.go**

```go
package session

import (
	"os/exec"
	"testing"
	"time"
)

func TestRunOpencodeNotFound(t *testing.T) {
	old := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("nonexistent-binary-xyz")
	}
	defer func() { execCommand = old }()

	_, err := Run("test", time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Verify compilation**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go build ./internal/session/`
Expected: no output

- [ ] **Commit**

```bash
git add orchestrator/internal/session/ && git commit -m "feat: add opencode session runner"
```

---

### Task 8: verify package

**Files:**
- Create: `orchestrator/internal/verify/verify.go`
- Create: `orchestrator/internal/verify/verify_test.go`

Runs shell commands and captures output. 1:1 port of Python `verify.py`.

- [ ] **Write verify.go**

```go
package verify

import (
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	ReturnCode int
	Stdout     string
	Stderr     string
}

func (r *Result) Success() bool {
	return r.ReturnCode == 0
}

var execCommand = exec.Command

func Run(spec interface{}, timeout time.Duration) ([]Result, error) {
	switch v := spec.(type) {
	case nil:
		return nil, nil
	case string:
		r, err := runOne(v, timeout)
		if err != nil {
			return nil, err
		}
		return []Result{*r}, nil
	case []string:
		var results []Result
		for _, cmd := range v {
			r, err := runOne(cmd, timeout)
			if err != nil {
				return nil, err
			}
			results = append(results, *r)
		}
		return results, nil
	case []interface{}:
		var results []Result
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			r, err := runOne(s, timeout)
			if err != nil {
				return nil, err
			}
			results = append(results, *r)
		}
		return results, nil
	default:
		return nil, nil
	}
}

func runOne(cmdStr string, timeout time.Duration) (*Result, error) {
	cmd := execCommand("sh", "-c", cmdStr)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if timeout > 0 {
		var timer *time.Timer
		if err := cmd.Start(); err != nil {
			return &Result{Stderr: err.Error()}, err
		}
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-done:
		case <-time.After(timeout):
			cmd.Process.Kill()
			<-done
		}
	} else {
		if err := cmd.Run(); err != nil {
			// non-zero exit is not a fatal error for verify
		}
	}

	return &Result{
		ReturnCode: cmd.ProcessState.ExitCode(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, nil
}

func AllSuccessful(results []Result) bool {
	for _, r := range results {
		if !r.Success() {
			return false
		}
	}
	return true
}
```

- [ ] **Write verify_test.go**

```go
package verify

import (
	"testing"
)

func TestRunString(t *testing.T) {
	results, err := Run("echo ok", 0)
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
	results, err := Run([]string{"echo a", "echo b"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success() {
			t.Fatal("expected all to succeed")
		}
	}
}

func TestRunNil(t *testing.T) {
	results, err := Run(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil, got %v", results)
	}
}

func TestRunFailure(t *testing.T) {
	results, err := Run("false", 0)
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
```

- [ ] **Run tests**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go test ./internal/verify/ -v`
Expected: all pass

- [ ] **Commit**

```bash
git add orchestrator/internal/verify/ && git commit -m "feat: add verify command runner"
```

---

### Task 9: handoff package

**Files:**
- Create: `orchestrator/internal/handoff/handoff.go`
- Create: `orchestrator/internal/handoff/handoff_test.go`

Recursively collects `HANDOFF.md` files. 1:1 port of Python `handoff.py`.

- [ ] **Write handoff.go**

```go
package handoff

import (
	"os"
	"path/filepath"
	"strings"
)

type Note struct {
	Source  string
	Content string
}

func Collect(root string) ([]Note, error) {
	var notes []Note
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return notes, nil
	}
	err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if strings.ToUpper(filepath.Base(path)) == "HANDOFF.MD" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			notes = append(notes, Note{
				Source:  path,
				Content: string(data),
			})
		}
		return nil
	})
	return notes, err
}

func FormatNotes(notes []Note) string {
	if len(notes) == 0 {
		return ""
	}
	var parts []string
	for _, n := range notes {
		parts = append(parts, "## Handoff: "+n.Source+"\n\n"+n.Content)
	}
	return strings.Join(parts, "\n\n---\n\n")
}
```

- [ ] **Write handoff_test.go**

```go
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
```

- [ ] **Run tests**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go test ./internal/handoff/ -v`
Expected: all pass

- [ ] **Commit**

```bash
git add orchestrator/internal/handoff/ && git commit -m "feat: add handoff notes collection"
```

---

### Task 10: context package

**Files:**
- Create: `orchestrator/internal/context/context.go`
- Create: `orchestrator/internal/context/context_test.go`

Context bundle assembly with token budget control. 1:1 port of Python `context.py`.

- [ ] **Write context.go**

```go
package context

import (
	"math"
	"strings"

	"orchestrator/internal/handoff"
	"orchestrator/internal/verify"
)

type DegradeStrategy string

const (
	DegradeNoVerify  DegradeStrategy = "no_verify"
	DegradeNoHandoff DegradeStrategy = "no_handoff"
	DegradeMinimal   DegradeStrategy = "minimal"
)

const TokenEstimateRatio = 4
const MaxTokens = 180_000

func CountTokens(text string) int {
	return int(math.Ceil(float64(len(text)) / TokenEstimateRatio))
}

type MilestoneInfo struct {
	Name string
	Spec string
}

func BuildBundle(milestones []MilestoneInfo, handoffNotes []handoff.Note, verifyResults []verify.Result) string {
	var parts []string

	if len(milestones) > 0 {
		parts = append(parts, "## Pipeline State\n")
		for _, ms := range milestones {
			parts = append(parts, "### Milestone: "+ms.Name+"\n\n"+ms.Spec+"\n")
		}
	}

	if len(handoffNotes) > 0 {
		parts = append(parts, "## Handoff Notes\n")
		for _, n := range handoffNotes {
			parts = append(parts, "### From: "+n.Source+"\n\n"+n.Content+"\n")
		}
	}

	if len(verifyResults) > 0 {
		parts = append(parts, "## Verify Results\n")
		for _, r := range verifyResults {
			status := "PASS"
			if r.ReturnCode != 0 {
				status = "FAIL"
			}
			parts = append(parts, "- "+status+": "+r.Stdout)
			if r.Stderr != "" {
				parts = append(parts, "  stderr: "+r.Stderr)
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func ExceedsThreshold(text string, maxTokens int) bool {
	if maxTokens <= 0 {
		maxTokens = MaxTokens
	}
	return CountTokens(text) > maxTokens
}

func Degrade(bundle string, strategy DegradeStrategy) string {
	lines := strings.Split(bundle, "\n")
	var filtered []string
	skipSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if (strategy == DegradeNoVerify || strategy == DegradeMinimal) && trimmed == "## Verify Results" {
			skipSection = true
			continue
		}
		if (strategy == DegradeNoHandoff || strategy == DegradeMinimal) && trimmed == "## Handoff Notes" {
			skipSection = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && trimmed != "## Pipeline State" {
			skipSection = false
			continue
		}
		if strings.HasPrefix(trimmed, "## Pipeline State") {
			skipSection = false
		}
		if !skipSection {
			filtered = append(filtered, line)
		}
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}
```

- [ ] **Write context_test.go**

```go
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
```

- [ ] **Run tests**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go test ./internal/context/ -v`
Expected: all pass

- [ ] **Commit**

```bash
git add orchestrator/internal/context/ && git commit -m "feat: add context bundle with token budget"
```

---

### Task 11: review package

**Files:**
- Create: `orchestrator/internal/review/review.go`
- Create: `orchestrator/internal/review/review_test.go`

Phase and final review sessions. 1:1 port of Python `review.py`.

- [ ] **Write review.go**

```go
package review

import (
	"fmt"
	"strings"
	"time"

	"orchestrator/internal/context"
	"orchestrator/internal/handoff"
	"orchestrator/internal/session"
	"orchestrator/internal/verify"
)

type Result struct {
	ReturnCode int
	Stdout     string
	Stderr     string
}

func buildReviewPrompt(reviewType string, projectName string, milestones []context.MilestoneInfo, handoffNotes []handoff.Note, verifyResults []verify.Result) (string, string) {
	bundle := context.BuildBundle(milestones, handoffNotes, verifyResults)

	if context.ExceedsThreshold(bundle, 0) {
		degraded := false
		if len(verifyResults) > 0 {
			bundle = context.Degrade(bundle, context.DegradeNoVerify)
			degraded = true
		}
		if len(handoffNotes) > 0 && context.ExceedsThreshold(bundle, 0) {
			bundle = context.Degrade(bundle, context.DegradeNoHandoff)
			degraded = true
		}
		if !degraded {
			bundle = context.Degrade(bundle, context.DegradeMinimal)
		}
	}

	target := projectName
	prompt := fmt.Sprintf(
		"Review: %s for %s\n\n## Context Bundle\n\n%s\n\nPlease perform a %s review and provide feedback. Write your review to HANDOFF.md in the current directory.",
		reviewType, target, bundle, reviewType,
	)
	return bundle, prompt
}

func Phase(milestoneName, milestoneSpec string, handoffNotes []handoff.Note, verifyResults []verify.Result) (*Result, error) {
	ms := []context.MilestoneInfo{{Name: milestoneName, Spec: milestoneSpec}}
	_, prompt := buildReviewPrompt("phase", milestoneName, ms, handoffNotes, verifyResults)
	result, err := session.Run(prompt, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	return &Result{
		ReturnCode: result.ReturnCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}, nil
}

func Final(projectName string, milestones []context.MilestoneInfo, handoffNotes []handoff.Note, verifyResults []verify.Result) (*Result, error) {
	_, prompt := buildReviewPrompt("final", projectName, milestones, handoffNotes, verifyResults)
	result, err := session.Run(prompt, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	return &Result{
		ReturnCode: result.ReturnCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}, nil
}
```

- [ ] **Write review_test.go**

```go
package review

import (
	"strings"
	"testing"
)

func TestBuildReviewPrompt(t *testing.T) {
	bundle, prompt := buildReviewPrompt("phase", "m1", nil, nil, nil)
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
```

- [ ] **Verify compilation**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go build ./internal/review/`
Expected: no output

- [ ] **Commit**

```bash
git add orchestrator/internal/review/ && git commit -m "feat: add phase and final review"
```

---

### Task 12: pipeline package and main.go

**Files:**
- Create: `orchestrator/internal/pipeline/pipeline.go`
- Create: `orchestrator/internal/pipeline/pipeline_test.go`
- Create: `orchestrator/main.go`

Core orchestration loop and CLI entrypoint.

- [ ] **Write pipeline.go**

```go
package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"orchestrator/internal/context"
	"log/slog"

	"orchestrator/internal/git"
	"orchestrator/internal/handoff"
	"orchestrator/internal/log"
	"orchestrator/internal/review"
	"orchestrator/internal/session"
	"orchestrator/internal/spec"
	"orchestrator/internal/state"
	"orchestrator/internal/topo"
	"orchestrator/internal/verify"
)

type Config struct {
	SpecPath   string
	Root       string
	ExtraSpecs []string
	Branch     string
}

type Pipeline struct {
	Config Config
	logger *slog.Logger
}

func New(cfg Config) *Pipeline {
	return &Pipeline{
		Config: cfg,
		logger: log.Get("pipeline"),
	}
}

func (p *Pipeline) Run() int {
	specPath := filepath.Join(p.Config.Root, p.Config.SpecPath)
	var extraPaths []string
	for _, e := range p.Config.ExtraSpecs {
		extraPaths = append(extraPaths, filepath.Join(p.Config.Root, e))
	}

	projectSpec, err := spec.Load(specPath, extraPaths...)
	if err != nil {
		p.logger.Error("loading spec", "error", err)
		return 1
	}

	if errs := spec.Preflight(projectSpec); len(errs) > 0 {
		p.logger.Error("preflight validation failed")
		for _, e := range errs {
			p.logger.Error("  - " + e)
		}
		return 1
	}

	topoMilestones := make([]topo.Milestone, len(projectSpec.Milestones))
	names := make([]string, len(projectSpec.Milestones))
	for i, ms := range projectSpec.Milestones {
		topoMilestones[i] = topo.Milestone{Name: ms.Name, DependsOn: ms.DependsOn}
		names[i] = ms.Name
	}

	ordered, err := topo.Sort(topoMilestones)
	if err != nil {
		p.logger.Error("topological sort failed", "error", err)
		return 1
	}
	p.logger.Info("milestone order", "order", ordered)

	pipeState := state.New(names, filepath.Join(p.Config.Root, "state.yaml"))
	defaultBranch := detectDefaultBranch()

	var seenHandoffPaths = make(map[string]bool)
	var allHandoffNotes []handoff.Note
	var allVerifyResults []verify.Result
	hasFailures := false

	for _, msName := range ordered {
		ms := spec.GetMilestone(projectSpec, msName)
		if ms == nil {
			continue
		}

		pipeState.Set(msName, state.StatusInProgress)
		pipeState.Save()

		var branchName string
		if p.Config.Branch != "" {
			branchName = p.Config.Branch
			if !git.IsClean() {
				p.logger.Warn("working tree not clean")
			}
		} else {
			branchName = msName + "-pipeline"
			if err := git.Checkout(defaultBranch); err != nil {
				p.logger.Error("checkout failed", "branch", defaultBranch, "error", err)
				return 1
			}
			if err := git.CreateBranch(branchName); err != nil {
				p.logger.Error("create branch failed", "branch", branchName, "error", err)
				return 1
			}
		}

		p.logger.Info("starting milestone", "name", msName, "branch", branchName)
		prompt := computeMilestoneSpec(ms, projectSpec.Milestones, pipeState, allHandoffNotes)
		p.logger.Info("running milestone", "name", msName, "prompt_bytes", len(prompt))

		result, err := session.Run(prompt, 0)
		if err != nil || result.ReturnCode != 0 {
			pipeState.Set(msName, state.StatusFailed)
			hasFailures = true
			p.logger.Warn("session failed", "name", msName, "code", result.ReturnCode, "error", err)
		} else {
			pipeState.Set(msName, state.StatusCompleted)
		}
		pipeState.Save()

		// verify
		if ms.Verify != nil {
			vResults, err := verify.Run(ms.Verify, 0)
			if err == nil {
				allVerifyResults = append(allVerifyResults, vResults...)
			}
		}

		// collect handoff
		newNotes, _ := handoff.Collect(p.Config.Root)
		var deduped []handoff.Note
		for _, n := range newNotes {
			if !seenHandoffPaths[n.Source] {
				seenHandoffPaths[n.Source] = true
				deduped = append(deduped, n)
			}
		}
		allHandoffNotes = append(allHandoffNotes, deduped...)

		// phase review
		review.Phase(msName, ms.Spec, deduped, allVerifyResults)

		// git operations
		if p.Config.Branch == "" {
			if err := git.Commit("Milestone " + msName); err != nil {
				p.logger.Error("commit failed", "error", err)
				return 1
			}
			if err := git.Tag(msName + "-done"); err != nil {
				p.logger.Warn("tag failed", "error", err)
			}
			if err := git.Checkout(defaultBranch); err != nil {
				p.logger.Error("checkout failed", "error", err)
				return 1
			}
			if err := git.SquashMerge(branchName); err != nil {
				p.logger.Error("squash merge failed", "error", err)
				return 1
			}
		}
	}

	// final review
	var msInfos []context.MilestoneInfo
	for _, ms := range projectSpec.Milestones {
		msInfos = append(msInfos, context.MilestoneInfo{Name: ms.Name, Spec: ms.Spec})
	}
	review.Final(projectSpec.Project.Name, msInfos, allHandoffNotes, allVerifyResults)

	if pipeState.AllCompleted() {
		git.Tag(projectSpec.Project.Name + "-v1.0")
		p.logger.Info("pipeline completed successfully")
		return 0
	}
	if hasFailures {
		p.logger.Warn("pipeline completed with failures")
		return 1
	}
	return 0
}

func detectDefaultBranch() string {
	if b, err := git.CurrentBranch(); err == nil {
		return b
	}
	return "main"
}

func computeMilestoneSpec(ms *spec.Milestone, all []spec.Milestone, pipeState *state.State, handoffNotes []handoff.Note) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("# Milestone: %s\n", ms.Name))
	if ms.Spec != "" {
		parts = append(parts, "## Spec\n\n"+ms.Spec+"\n")
	}

	if len(ms.Tasks) > 0 {
		parts = append(parts, "## Tasks\n")
		for _, t := range ms.Tasks {
			parts = append(parts, fmt.Sprintf("### %s\n\n%s\n", t.Name, t.Prompt))
		}
	}

	if pipeState != nil {
		parts = append(parts, "## Pipeline State\n")
		for name, st := range pipeState.GetAll() {
			parts = append(parts, fmt.Sprintf("- %s: %s\n", name, st))
		}
	}

	if len(handoffNotes) > 0 {
		parts = append(parts, "## Previous Handoff Notes\n")
		for _, n := range handoffNotes {
			parts = append(parts, fmt.Sprintf("### %s\n\n%s\n", n.Source, n.Content))
		}
	}

	var remaining []spec.Milestone
	for _, m := range all {
		if m.Name != ms.Name {
			remaining = append(remaining, m)
		}
	}
	if len(remaining) > 0 {
		parts = append(parts, "## Upcoming Milestones\n")
		for _, m := range remaining {
			specPreview := m.Spec
			if len(specPreview) > 80 {
				specPreview = specPreview[:80]
			}
			parts = append(parts, fmt.Sprintf("- %s: %s...\n", m.Name, specPreview))
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}
```

- [ ] **Write main.go**

```go
package main

import (
	"flag"
	"os"

	"orchestrator/internal/log"
	"orchestrator/internal/pipeline"
)

func main() {
	specPath := flag.String("spec", "project.yaml", "Path to project.yaml")
	specShort := flag.String("s", "project.yaml", "Path to project.yaml (shorthand)")
	root := flag.String("root", ".", "Project root directory")
	branch := flag.String("branch", "", "Use existing branch instead of creating milestone branches")
	flag.Parse()

	log.Setup()
	logger := log.Get("main")

	var extraSpecs []string
	// parse remaining args for --extra-spec
	args := flag.Args()
	for i := 0; i < len(args); i++ {
		if args[i] == "--extra-spec" && i+1 < len(args) {
			extraSpecs = append(extraSpecs, args[i+1])
			i++
		}
	}

	// prefer -s over --spec if provided
	finalSpec := *specPath
	if *specShort != "project.yaml" {
		finalSpec = *specShort
	}

	cfg := pipeline.Config{
		SpecPath:   finalSpec,
		Root:       *root,
		ExtraSpecs: extraSpecs,
		Branch:     *branch,
	}

	p := pipeline.New(cfg)
	os.Exit(p.Run())
}
```

- [ ] **Write pipeline_test.go**

```go
package pipeline

import (
	"strings"
	"testing"

	"orchestrator/internal/spec"
)

func TestDetectDefaultBranch(t *testing.T) {
	branch := detectDefaultBranch()
	if branch == "" {
		t.Fatal("expected non-empty branch")
	}
}

func TestComputeMilestoneSpec(t *testing.T) {
	ms := &spec.Milestone{
		Name: "m1",
		Spec: "test spec",
		Tasks: []spec.TaskSpec{
			{Name: "t1", Prompt: "do something"},
		},
	}
	result := computeMilestoneSpec(ms, nil, nil, nil)
	if !strings.Contains(result, "m1") {
		t.Fatal("missing milestone name")
	}
	if !strings.Contains(result, "do something") {
		t.Fatal("missing task prompt")
	}
}
```

- [ ] **Build the binary**

Run: `cd /home/rain/git/pipeline-002/orchestrator && go build -o orchestrator .`
Expected: produces `orchestrator/orchestrator` binary

- [ ] **Test with sample project**

Run: `cd /home/rain/git/pipeline-002 && ./orchestrator/orchestrator --spec sample-project/project.yaml --branch test --dry-run` (Note: we need a --dry-run flag or just verify parsing)

Actually for a quick smoke test:
```bash
cd /home/rain/git/pipeline-002 && ./orchestrator/orchestrator --help
```
Expected: shows usage

- [ ] **Commit**

```bash
git add orchestrator/internal/pipeline/ orchestrator/main.go
git commit -m "feat: add pipeline orchestration and CLI entrypoint"
```

---

### Task 13: Build verification and cross-check

- [ ] **Run all tests**

```bash
cd /home/rain/git/pipeline-002/orchestrator && go test ./... -v 2>&1 | head -100
```

Expected: all tests pass

- [ ] **Verify binary runs**

```bash
cd /home/rain/git/pipeline-002/orchestrator && go build -o orchestrator . && ./orchestrator --help
```

Expected: shows CLI usage

- [ ] **Clean up test binary**

```bash
rm -f /home/rain/git/pipeline-002/orchestrator/orchestrator
```

- [ ] **Add Go build to .gitignore**

Append to `/home/rain/git/pipeline-002/.gitignore`:
```
orchestrator/orchestrator
```

- [ ] **Commit**

```bash
git add .gitignore && git commit -m "chore: add binary to gitignore"
```

---

### Task 14: Update README.md

**Files:**
- Modify: `README.md`

Replace Python content with Go version.

- [ ] **Write README.md**

```markdown
# AI Project Pipeline Orchestrator

AI 项目流水线编排器，将复杂项目分解为多个里程碑（Milestone），
每个里程碑创建 git 分支、调用 AI 编码会话、执行验证命令、
进行阶段审查，最后合并回主分支。

## Quick Start

```bash
# Build
cd orchestrator && go build -o orchestrator .

# Run
./orchestrator/orchestrator [--spec project.yaml] [--root .] [--branch X]
```

## Usage

Write a `project.yaml`:

```yaml
project:
  name: my-project

milestones:
  - name: setup
    spec: Initialize project
    tasks:
      - name: init
        prompt: Create project structure
    verify:
      - ls -la
      - python3 -c "import yaml; print('ok')"
```

Run:

```bash
# Basic
orchestrator

# Custom spec
orchestrator --spec path/to/project.yaml

# Combine specs
orchestrator --spec base.yaml --extra-spec features.yaml

# Existing branch (skip git branch creation)
orchestrator --branch existing-branch
```

## Requirements

- Go 1.22+
- Git
- [opencode](https://opencode.ai) (on `$PATH`)

## Development

```bash
cd orchestrator
go test ./... -v
go build -o orchestrator .
```

## Project Structure

```
orchestrator/
├── main.go                    # CLI 入口
├── internal/
│   ├── pipeline/              # Pipeline 核心编排
│   ├── spec/                  # YAML 加载和校验
│   ├── state/                 # 状态持久化
│   ├── topo/                  # 拓扑排序
│   ├── git/                   # Git 操作
│   ├── session/               # AI 会话管理
│   ├── handoff/               # HANDOFF.md 收集
│   ├── verify/                # 验证命令执行
│   ├── review/                # 阶段/最终审查
│   ├── context/               # 上下文包管理
│   └── log/                   # 日志配置
├── go.mod
└── sample-project/
```

## How It Works

1. **Load YAML** — 解析 project.yaml
2. **Preflight** — 校验格式和依赖完整性
3. **Topo Sort** — 根据 depends_on 计算执行顺序
4. **Per Milestone** — 创建分支 → 运行 opencode → 验证 → 收集 HANDOFF.md → 阶段审查 → 合并
5. **Final Review** — 全部完成后最终审查
6. **Tag** — 成功完成后打标签
```

- [ ] **Commit**

```bash
git add README.md && git commit -m "docs: update README for Go version"
```

---

### Task 15: Update AGENTS.md

**Files:**
- Modify: `AGENTS.md`

Replace Python commands with Go equivalents.

- [ ] **Write AGENTS.md**

```markdown
# orchestrator — AI Project Pipeline Orchestrator

## Quick start
```bash
go build -o orchestrator .              # build binary
./orchestrator --help                   # verify
go test ./... -v                        # run all tests
go test ./internal/topo/ -v             # single package
```

## Architecture
- **Entrypoint**: `orchestrator/main.go` → CLI: `orchestrator [--spec project.yaml] [--root .] [--branch X]`
- **Config**: `go.mod` module `orchestrator`, single dep `gopkg.in/yaml.v3`, Go 1.22+
- **Core loop** (`pipeline.Run()`): load YAML → preflight → topo sort → for each milestone: create git branch → run opencode session → run verify → collect HANDOFF.md → phase review → squash-merge → tag
- **Modules**: `main.go` (CLI), `internal/pipeline/` (orchestration), `spec/` (YAML load+validate), `topo/` (sort+cycle detection), `state/` (persistence), `session/` (opencode subprocess), `git/` (branch/commit/tag/merge), `handoff/` (HANDOFF.md collector), `verify/` (shell commands), `review/` (phase/final review), `context/` (bundle+token budget), `log/` (slog wrapper)
- **Design doc**: `docs/superpowers/specs/2026-04-25-orchestrator-go-rewrite-design.md`

## Key conventions
- `internal/*` packages, each in its own directory
- Functions > interfaces (simple wrappers); interfaces only for testability (execCommand variables)
- Logging: `log.Get("name")` returns `*slog.Logger`
- Error handling: return errors to caller; Pipeline.Run() returns int exit code
- Tests: Go standard `testing` package, table-driven where appropriate

## Pipeline details
- YAML must have `project.name` and `milestones[]` with `name` and `tasks[]`
- `depends_on` is milestone-level string list
- `verify` can be string or `[]string`; exit code 0 = pass
- `--branch X` skips branch creation
- Each milestone = one opencode session
- Handoff notes: `**/HANDOFF.md` collected after each milestone
- State: `state.yaml` (gitignored)
- Tags: `{ms_name}-done` per milestone, `{project_name}-v1.0` on completion
- Default branch: detected from `git rev-parse --abbrev-ref HEAD`, fallback `main`

## Context bundle
- `context` package: ~4 chars/token, threshold 180k
- Degrade: `no_verify` → skip verify results; `no_handoff` → skip handoff; `minimal` → skip both
- Manual degrade in `review` package only

## Testing quirks
- Mock external commands via `var execCommand = exec.Command` pattern (git, session, verify packages)
- `session` tests patch `execCommand` to avoid real opencode calls
- `topo`, `spec`, `state`, `context` have no external deps, easily testable
- Run `go test ./... -v` from `orchestrator/` after any change
```

- [ ] **Commit**

```bash
git add AGENTS.md && git commit -m "docs: update AGENTS.md for Go version"
```

---

### Task 16: Remove Python source files

- [ ] **Delete Python source and test files**

```bash
rm -rf /home/rain/git/pipeline-002/orchestrator/__init__.py
rm -rf /home/rain/git/pipeline-002/orchestrator/main.py
rm -rf /home/rain/git/pipeline-002/orchestrator/spec.py
rm -rf /home/rain/git/pipeline-002/orchestrator/state.py
rm -rf /home/rain/git/pipeline-002/orchestrator/topo.py
rm -rf /home/rain/git/pipeline-002/orchestrator/git.py
rm -rf /home/rain/git/pipeline-002/orchestrator/session.py
rm -rf /home/rain/git/pipeline-002/orchestrator/handoff.py
rm -rf /home/rain/git/pipeline-002/orchestrator/verify.py
rm -rf /home/rain/git/pipeline-002/orchestrator/review.py
rm -rf /home/rain/git/pipeline-002/orchestrator/context.py
rm -rf /home/rain/git/pipeline-002/orchestrator/log.py
rm -rf /home/rain/git/pipeline-002/tests/
rm -rf /home/rain/git/pipeline-002/pyproject.toml
rm -rf /home/rain/git/pipeline-002/uv.lock
rm -rf /home/rain/git/pipeline-002/orchestrator.egg-info/
rm -rf /home/rain/git/pipeline-002/.mypy_cache/
rm -rf /home/rain/git/pipeline-002/.pytest_cache/
rm -rf /home/rain/git/pipeline-002/.ruff_cache/
rm -rf /home/rain/git/pipeline-002/.venv/
rm -rf /home/rain/git/pipeline-002/orchestrator/__pycache__/
```

- [ ] **Remove CLAUDE.md** (replaced by AGENTS.md)

```bash
rm /home/rain/git/pipeline-002/CLAUDE.md
```

- [ ] **Verify build still works after cleanup**

```bash
cd /home/rain/git/pipeline-002/orchestrator && go build -o orchestrator . && ./orchestrator --help
```

- [ ] **Commit**

```bash
git add -A && git commit -m "chore: remove Python source, replace with Go implementation"
```

---

### Task 17: Final sanity check

- [ ] **Full test suite**

```bash
cd /home/rain/git/pipeline-002/orchestrator && go test ./... -v
```

Expected: all tests pass

- [ ] **Verify project structure**

```bash
ls -la /home/rain/git/pipeline-002/
ls -la /home/rain/git/pipeline-002/orchestrator/internal/
```
Expected: clean Go project structure, no Python artifacts

- [ ] **Final commit if needed**

```bash
git status  # confirm clean
```
