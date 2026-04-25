package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Project struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
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
