package runtime

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kevenmiano/nestgo/metadata"
)

// moduleVisibility builds injectable provider scopes from static module metadata.
// Legacy modules with no explicit provider/controller lists retain global DI.
func moduleVisibility(packages []metadata.Package) (map[reflect.Type]map[reflect.Type]bool, error) {
	modules := make(map[reflect.Type]metadata.Module)
	owners := make(map[reflect.Type]reflect.Type)
	registered := make(map[reflect.Type]bool)
	for _, pkg := range packages {
		for _, provider := range pkg.Providers {
			registered[provider.Type] = true
		}
		for _, mod := range pkg.Modules {
			if mod.Type == nil {
				return nil, fmt.Errorf("module has nil type")
			}
			if _, exists := modules[mod.Type]; exists {
				return nil, fmt.Errorf("duplicate module %s", mod.Type)
			}
			modules[mod.Type] = mod
			for _, typ := range append(append([]reflect.Type{}, mod.Providers...), mod.Controllers...) {
				if typ == nil {
					return nil, fmt.Errorf("module %s has nil member", mod.Type)
				}
				if previous, exists := owners[typ]; exists && previous != mod.Type {
					return nil, fmt.Errorf("type %s belongs to both %s and %s", typ, previous, mod.Type)
				}
				owners[typ] = mod.Type
			}
		}
	}

	// Validate every module, including modules without owned providers.
	// Sorting prevents metadata registration order from changing diagnostics.
	moduleTypes := make([]reflect.Type, 0, len(modules))
	for moduleType := range modules {
		moduleTypes = append(moduleTypes, moduleType)
	}
	sort.Slice(moduleTypes, func(i, j int) bool {
		return moduleTypes[i].String() < moduleTypes[j].String()
	})
	for _, moduleType := range moduleTypes {
		mod := modules[moduleType]
		for _, local := range mod.Providers {
			if !registered[local] {
				return nil, fmt.Errorf("module %s declares unregistered provider %s", moduleType, local)
			}
		}
		for _, exported := range mod.Exports {
			if !containsType(mod.Providers, exported) || !registered[exported] {
				return nil, fmt.Errorf("module %s exports undeclared or unregistered provider %v", moduleType, exported)
			}
		}
		for _, imported := range mod.Imports {
			if imported == nil {
				return nil, fmt.Errorf("module %s imports nil module", moduleType)
			}
			if _, exists := modules[imported]; !exists {
				return nil, fmt.Errorf("module %s imports unknown module %v", moduleType, imported)
			}
		}
	}
	// Reject import cycles across package boundaries before constructing scopes.
	state := make(map[reflect.Type]uint8, len(modules))
	stack := make([]reflect.Type, 0, len(modules))
	var visit func(reflect.Type) error
	visit = func(moduleType reflect.Type) error {
		if state[moduleType] == 2 {
			return nil
		}
		if state[moduleType] == 1 {
			start := 0
			for i, item := range stack {
				if item == moduleType {
					start = i
					break
				}
			}
			parts := make([]string, 0, len(stack)-start+1)
			for _, item := range stack[start:] {
				parts = append(parts, item.String())
			}
			parts = append(parts, moduleType.String())
			return fmt.Errorf("module import cycle: %s", strings.Join(parts, " -> "))
		}
		state[moduleType] = 1
		stack = append(stack, moduleType)
		imports := append([]reflect.Type(nil), modules[moduleType].Imports...)
		sort.Slice(imports, func(i, j int) bool { return imports[i].String() < imports[j].String() })
		for _, imported := range imports {
			if err := visit(imported); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[moduleType] = 2
		return nil
	}
	for _, moduleType := range moduleTypes {
		if err := visit(moduleType); err != nil {
			return nil, err
		}
	}
	visible := make(map[reflect.Type]map[reflect.Type]bool)
	for typ, moduleType := range owners {
		mod := modules[moduleType]
		allowed := make(map[reflect.Type]bool)
		for _, local := range mod.Providers {
			allowed[local] = true
		}
		for _, imported := range mod.Imports {
			for _, exported := range modules[imported].Exports {
				allowed[exported] = true
			}
		}
		visible[typ] = allowed
	}
	return visible, nil
}

func containsType(types []reflect.Type, target reflect.Type) bool {
	for _, typ := range types {
		if typ == target {
			return true
		}
	}
	return false
}
