package runtime

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/kevenmiano/nestgo/metadata"
)

// registryImportClosure resolves imported module metadata across registered Go
// packages. Test fixtures may supply root names through the internal helper.
func registryImportClosure(names []string) ([]metadata.Package, error) {
	return registryImportClosureFrom(names, metadata.All())
}

func registryImportClosureFrom(names []string, registered map[string]metadata.Package) ([]metadata.Package, error) {
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

// discoverCompiledPackages resolves the linked application's module tree once
// at bootstrap. Code-first applications contain no compiled package metadata.
func discoverCompiledPackages() ([]metadata.Package, error) {
	compiled := metadata.Compiled()
	if len(compiled) == 0 {
		return nil, nil
	}
	return compiledRootClosure(compiled, metadata.All())
}

// compiledRootClosure operates on snapshots so tests do not mutate the global
// registry. Unrelated manually registered fixtures never become app roots.
func compiledRootClosure(compiled, registered map[string]metadata.Package) ([]metadata.Package, error) {
	names := make([]string, 0, len(compiled))
	for name := range compiled {
		names = append(names, name)
	}
	sort.Strings(names)

	// Index module ownership once. The previous implementation constructed
	// separate ownership and reachability maps during multiple graph walks.
	owners := make(map[reflect.Type]string, len(registered))
	for name, pkg := range registered {
		for _, mod := range pkg.Modules {
			if mod.Type == nil {
				continue
			}
			if other, exists := owners[mod.Type]; exists && other != name {
				return nil, fmt.Errorf("duplicate module %s in %s and %s", mod.Type, other, name)
			}
			owners[mod.Type] = name
		}
	}

	imported := make(map[reflect.Type]bool)
	var moduleTypes []reflect.Type
	for _, name := range names {
		for _, mod := range compiled[name].Modules {
			if mod.Type == nil {
				return nil, fmt.Errorf("compiled module in %q has nil type", name)
			}
			moduleTypes = append(moduleTypes, mod.Type)
			for _, dep := range mod.Imports {
				if dep == nil {
					return nil, fmt.Errorf("compiled module %s imports nil module", mod.Type)
				}
				imported[dep] = true
			}
		}
	}
	if len(moduleTypes) == 0 {
		if len(names) != 1 {
			return nil, fmt.Errorf("ambiguous compiled packages without @Module: %v", names)
		}
		return []metadata.Package{compiled[names[0]]}, nil
	}

	rootCount := 0
	var root reflect.Type
	for _, typ := range moduleTypes {
		if !imported[typ] {
			rootCount++
			root = typ
		}
	}
	if rootCount != 1 {
		return nil, fmt.Errorf("expected exactly one application root @Module, found %d", rootCount)
	}

	visited := make(map[string]bool, len(compiled))
	queue := []string{owners[root]}
	packages := make([]metadata.Package, 0, len(compiled))
	for i := 0; i < len(queue); i++ {
		name := queue[i]
		if visited[name] {
			continue
		}
		pkg, exists := registered[name]
		if !exists {
			return nil, fmt.Errorf("metadata package not found: %s", name)
		}
		visited[name] = true
		packages = append(packages, pkg)

		imports := make([]reflect.Type, 0)
		for _, mod := range pkg.Modules {
			imports = append(imports, mod.Imports...)
		}
		sort.Slice(imports, func(i, j int) bool { return imports[i].String() < imports[j].String() })
		for _, dependency := range imports {
			if dependency == nil {
				return nil, fmt.Errorf("metadata package %s imports nil module", name)
			}
			if owner := owners[dependency]; owner != "" && !visited[owner] {
				queue = append(queue, owner)
			}
			// Unknown imports remain errors in moduleVisibility.
		}
	}

	for _, name := range names {
		if !visited[name] {
			return nil, fmt.Errorf("compiled package %q is not reachable from application root %s", name, root)
		}
	}
	return packages, nil
}
