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

	owners := make(map[reflect.Type]string)
	imported := make(map[reflect.Type]bool)
	var moduleTypes []reflect.Type
	for _, name := range names {
		for _, module := range compiled[name].Modules {
			if module.Type == nil {
				return nil, fmt.Errorf("compiled module in %q has nil type", name)
			}
			if other, exists := owners[module.Type]; exists {
				return nil, fmt.Errorf("duplicate compiled module %s in %s and %s", module.Type, other, name)
			}
			owners[module.Type] = name
			moduleTypes = append(moduleTypes, module.Type)
			for _, dependency := range module.Imports {
				if dependency == nil {
					return nil, fmt.Errorf("compiled module %s imports nil module", module.Type)
				}
				imported[dependency] = true
			}
		}
	}

	if len(moduleTypes) == 0 {
		if len(names) != 1 {
			return nil, fmt.Errorf("ambiguous compiled packages without @Module: %v", names)
		}
		return []metadata.Package{compiled[names[0]]}, nil
	}

	var roots []reflect.Type
	for _, typ := range moduleTypes {
		if !imported[typ] {
			roots = append(roots, typ)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].String() < roots[j].String()
	})
	if len(roots) != 1 {
		return nil, fmt.Errorf("expected exactly one application root @Module, found %d (%v)", len(roots), roots)
	}

	// Reuse the existing explicit import-closure validation, but without
	// requiring a registry name in application startup code.
	root := roots[0]
	packages, err := registryImportClosureFrom([]string{owners[root]}, registered)
	if err != nil {
		return nil, err
	}

	// A binary with two disconnected compiled applications must fail closed
	// rather than accidentally combining providers from unrelated services.
	allOwners := make(map[reflect.Type]string)
	for name, pkg := range registered {
		for _, module := range pkg.Modules {
			if module.Type != nil {
				allOwners[module.Type] = name
			}
		}
	}
	reachable := make(map[string]bool, len(packages))
	var visit func(string)
	visit = func(name string) {
		if reachable[name] {
			return
		}
		reachable[name] = true
		for _, module := range registered[name].Modules {
			for _, dependency := range module.Imports {
				if owner := allOwners[dependency]; owner != "" {
					visit(owner)
				}
			}
		}
	}
	visit(owners[root])
	for _, name := range names {
		if !reachable[name] {
			return nil, fmt.Errorf("compiled package %q is not reachable from application root %s", name, root)
		}
	}
	return packages, nil
}
