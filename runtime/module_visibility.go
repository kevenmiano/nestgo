package runtime

import (
	"fmt"
	"reflect"

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
	visible := make(map[reflect.Type]map[reflect.Type]bool)
	for typ, moduleType := range owners {
		mod := modules[moduleType]
		allowed := make(map[reflect.Type]bool)
		for _, local := range mod.Providers {
			if !registered[local] {
				return nil, fmt.Errorf("module %s declares unregistered provider %s", moduleType, local)
			}
			allowed[local] = true
		}
		for _, imported := range mod.Imports {
			source, exists := modules[imported]
			if !exists {
				return nil, fmt.Errorf("module %s imports unknown module %v", moduleType, imported)
			}
			for _, exported := range source.Exports {
				if !containsType(source.Providers, exported) {
					return nil, fmt.Errorf("module %s exports undeclared provider %v", imported, exported)
				}
				if !registered[exported] {
					return nil, fmt.Errorf("module %s exports unregistered provider %v", imported, exported)
				}
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
