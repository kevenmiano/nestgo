package runtime

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/kevenmiano/nestgo/metadata"
)

// registryImportClosure resolves imported module metadata across registered Go
// packages. Callers supply root package names through WithRegistry.
func registryImportClosure(names []string) ([]metadata.Package, error) {
	registered := metadata.All()
	packageNames := make([]string, 0, len(registered))
	for name := range registered {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)
	owners := make(map[reflect.Type][]string)
	for _, name := range packageNames {
		for _, module := range registered[name].Modules {
			if module.Type != nil {
				owners[module.Type] = append(owners[module.Type], name)
			}
		}
	}
	visited := make(map[string]bool)
	queue := append([]string(nil), names...)
	packages := make([]metadata.Package, 0, len(names))
	for len(queue) != 0 {
		name := queue[0]
		queue = queue[1:]
		if visited[name] {
			continue
		}
		pkg, exists := registered[name]
		if !exists {
			return nil, fmt.Errorf("metadata package not found: %s", name)
		}
		visited[name] = true
		packages = append(packages, pkg)
		imports := make(map[reflect.Type]bool)
		for _, module := range pkg.Modules {
			for _, imported := range module.Imports {
				imports[imported] = true
			}
		}
		types := make([]reflect.Type, 0, len(imports))
		for imported := range imports {
			if imported == nil {
				return nil, fmt.Errorf("metadata package %s imports nil module", name)
			}
			types = append(types, imported)
		}
		sort.Slice(types, func(i, j int) bool { return types[i].String() < types[j].String() })
		for _, imported := range types {
			candidates := owners[imported]
			if len(candidates) > 1 {
				return nil, fmt.Errorf("ambiguous imported module %s is registered in %v", imported, candidates)
			}
			if len(candidates) == 1 && !visited[candidates[0]] {
				queue = append(queue, candidates[0])
			}
			// Missing modules remain errors in moduleVisibility's validation.
		}
	}
	return packages, nil
}
