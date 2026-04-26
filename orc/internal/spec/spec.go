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

type SpecItem struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	SpecFile    string `yaml:"spec_file"`
	TaskCount   int    `yaml:"task_count"`
	TestCount   int    `yaml:"test_count,omitempty"`
	EstMinutes  int    `yaml:"est_minutes"`
}

type Milestone struct {
	ID        string     `yaml:"id"`
	Name      string     `yaml:"name"`
	Order     int        `yaml:"order,omitempty"`
	DependsOn []string   `yaml:"depends_on,omitempty"`
	Specs     []SpecItem `yaml:"specs"`
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

func GetMilestone(s *Spec, id string) *Milestone {
	for i := range s.Milestones {
		if s.Milestones[i].ID == id {
			return &s.Milestones[i]
		}
	}
	return nil
}

func Preflight(s *Spec) []string {
	var errors []string
	ids := make(map[string]bool)
	for _, ms := range s.Milestones {
		if ms.ID == "" {
			errors = append(errors, "milestone without id")
			continue
		}
		if ids[ms.ID] {
			errors = append(errors, fmt.Sprintf("duplicate milestone id: %s", ms.ID))
		}
		ids[ms.ID] = true
		if len(ms.Specs) == 0 {
			errors = append(errors, fmt.Sprintf("milestone %q has no specs", ms.ID))
		}
	}
	for _, ms := range s.Milestones {
		for _, dep := range ms.DependsOn {
			if !ids[dep] {
				errors = append(errors, fmt.Sprintf("milestone %q has unknown dependency: %q", ms.ID, dep))
			}
		}
	}
	return errors
}
