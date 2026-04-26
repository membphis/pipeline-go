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
	Status    Status     `yaml:"status"`
	Timestamp *time.Time `yaml:"timestamp,omitempty"`
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
	now := time.Now()
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
