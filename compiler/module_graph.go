package compiler

import (
	"sort"
	"strings"
)

// validateModuleCycles checks local module imports before emitting metadata.
// It is intentionally independent of type checking or dynamic module registrations.
func validateModuleCycles(edges map[string][]string, locations map[string]Diagnostic) error {
	state := make(map[string]uint8, len(edges))
	stack := make([]string, 0, len(edges))
	var visit func(string) error
	visit = func(node string) error {
		if state[node] == 1 {
			start := 0
			for i, name := range stack {
				if name == node {
					start = i
					break
				}
			}
			cycle := append(append([]string{}, stack[start:]...), node)
			d := locations[node]
			d.Message = "module import cycle: " + joinModuleCycle(cycle)
			return d
		}
		if state[node] == 2 {
			return nil
		}
		state[node] = 1
		stack = append(stack, node)
		for _, dependency := range edges[node] {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[node] = 2
		return nil
	}
	// Callers provide an ordered list of module names to guarantee stable diagnostics.
	for _, name := range sortedModuleNames(edges) {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

func sortedModuleNames(edges map[string][]string) []string {
	result := make([]string, 0, len(edges))
	for name := range edges {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func joinModuleCycle(cycle []string) string { return strings.Join(cycle, " -> ") }
