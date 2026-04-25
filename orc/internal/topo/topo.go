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
