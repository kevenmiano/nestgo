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

 // discoverCompiledPackages selects the unique application root from the module
 // import graph linked into this executable. No runtime reflection or name
 // strings are required from the application.
 func discoverCompiledPackages() ([]metadata.Package, error) {
 	compiled := metadata.Compiled()
 	if len(compiled) == 0 {
 		return nil, nil // code-first applications have no compiled metadata
 	}
 	return compiledRootClosure(compiled, metadata.All())
 }

 // compiledRootClosure accepts explicit maps to keep graph tests isolated from
 // process-global package registration.
 func compiledRootClosure(compiled, registered map[string]metadata.Package) ([]metadata.Package, error) {
 	names := make([]string, 0, len(compiled))
 	for name := range compiled { names = append(names, name) }
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
 		if !imported[typ] { roots = append(roots, typ) }
 	}
 	sort.Slice(roots, func(i,j int) bool { return roots[i].String() < roots[j].String() })
 	if len(roots) != 1 {
 		return nil, fmt.Errorf("expected exactly one application root @Module, found %d (%v)", len(roots), roots)
 	}
 	packages, err := registryImportClosureFrom([]string{owners[roots[0]]}, registered)
 	if err != nil { return nil, err }
 	// Multiple unconnected compiled trees in the same executable are not
 	// silently merged into one application's provider scope.
 	reachable := make(map[string]bool, len(packages))
 	// Resolve the visited package names through the corresponding package
 	// module ownership, including packages that contain no module.
 	allOwners := make(map[reflect.Type]string)
 	for name, pkg := range registered {
 		for _, module := range pkg.Modules {
 			if module.Type != nil { allOwners[module.Type] = name }
 		}
 	}
 	var visit func(string)
 	visit = func(name string) {
 		if reachable[name] { return }
 		reachable[name] = true
 		pkg := registered[name]
 		for _, module := range pkg.Modules {
 			for _, dependency := range module.Imports {
 				if owner := allOwners[dependency]; owner != "" { visit(owner) }
 			}
 		}
 	}
 	visit(owners[roots[0]])
 	for _, name := range names {
 		if !reachable[name] {
 			return nil, fmt.Errorf("compiled package %q is not reachable from application root %s", name, roots[0])
 		}
 	}
 	return packages, nil
 }
