package runtime

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/kevenmiano/nestgo/metadata"
)

// providerEdge describes an actually injected dependency, not a merely
// compatible provider type. This also handles interface and optional fields.
type providerEdge struct {
	target reflect.Type
	field  string
}

func typeLess(a, b reflect.Type) int {
	if a.String() != b.String() {
		return strings.Compare(a.String(), b.String())
	}
	return strings.Compare(a.PkgPath(), b.PkgPath())
}

// orderBootstrap initializes dependencies before consumers and uses module
// import order to break ties between independent providers/controllers.
// All binding and module visibility checks have already succeeded.
func orderBootstrap(packages []metadata.Package, graph *bootstrapGraph) error {
	owners := make(map[reflect.Type]reflect.Type)
	modules := make(map[reflect.Type]metadata.Module)
	for _, pkg := range packages {
		for _, mod := range pkg.Modules {
			modules[mod.Type] = mod
			for _, typ := range mod.Providers {
				owners[typ] = mod.Type
			}
			for _, typ := range mod.Controllers {
				owners[typ] = mod.Type
			}
		}
	}

	// moduleVisibility has already rejected invalid imports/cycles.
	moduleTypes := make([]reflect.Type, 0, len(modules))
	for typ := range modules {
		moduleTypes = append(moduleTypes, typ)
	}
	slices.SortFunc(moduleTypes, typeLess)
	moduleRank := make(map[reflect.Type]int, len(modules))
	visitedModules := make(map[reflect.Type]bool, len(modules))
	var visitModule func(reflect.Type)
	visitModule = func(typ reflect.Type) {
		if visitedModules[typ] {
			return
		}
		visitedModules[typ] = true
		imports := slices.Clone(modules[typ].Imports)
		slices.SortFunc(imports, typeLess)
		for _, imported := range imports {
			visitModule(imported)
		}
		moduleRank[typ] = len(moduleRank)
	}
	for _, typ := range moduleTypes {
		visitModule(typ)
	}
	rank := func(typ reflect.Type) int {
		if mod, ok := owners[typ]; ok {
			return moduleRank[mod]
		}
		return len(moduleRank)
	}
	less := func(a, b reflect.Type) int {
		if rank(a) != rank(b) {
			return rank(a) - rank(b)
		}
		return typeLess(a, b)
	}

	// Match each field's resolved provider *identity*. Scanning the
	// declared interface type alone could create false dependencies.
	byPointer := make(map[uintptr]reflect.Type, len(graph.providers))
	for typ, instance := range graph.providers {
		byPointer[instance.Pointer()] = typ
	}
	edges := make(map[reflect.Type][]providerEdge, len(graph.providers))
	for typ, instance := range graph.providers {
		structValue := instance.Elem()
		structType := structValue.Type()
		for i := 0; i < structValue.NumField(); i++ {
			fieldInfo := structType.Field(i)
			if marker := fieldInfo.Tag.Get("nestgo"); marker != "inject" && marker != "optional" {
				continue
			}
			field := structValue.Field(i)
			if (field.Kind() != reflect.Ptr && field.Kind() != reflect.Interface) || field.IsNil() {
				continue
			}
			if field.Kind() == reflect.Interface {
				field = field.Elem()
			}
			if field.Kind() != reflect.Ptr || field.IsNil() {
				continue
			}
			if dep, ok := byPointer[field.Pointer()]; ok {
				edges[typ] = append(edges[typ], providerEdge{target: dep, field: fieldInfo.Name})
			}
		}
		slices.SortFunc(edges[typ], func(a, b providerEdge) int {
			if v := less(a.target, b.target); v != 0 {
				return v
			}
			return strings.Compare(a.field, b.field)
		})
	}

	// DFS produces a stable dependency-first topological ordering and
	// actionable edge/field diagnostics for indirect and self-cycles.
	state := make(map[reflect.Type]uint8, len(graph.providers))
	stack := make([]reflect.Type, 0, len(graph.providers))
	stackFields := make([]string, 0, len(graph.providers))
	order := make([]reflect.Type, 0, len(graph.providers))
	var visit func(reflect.Type) error
	visit = func(typ reflect.Type) error {
		state[typ] = 1
		stack = append(stack, typ)
		for _, edge := range edges[typ] {
			if state[edge.target] == 1 {
				start := 0
				for i, item := range stack {
					if item == edge.target {
						start = i
						break
					}
				}
				parts := make([]string, 0, len(stack)-start+1)
				for i := start; i < len(stack)-1; i++ {
					parts = append(parts, fmt.Sprintf("%s.%s", stack[i], stackFields[i]))
				}
				parts = append(parts, fmt.Sprintf("%s.%s", typ, edge.field), edge.target.String())
				return rejectInjection("dependency_cycle", typ, edge.field, edge.target,
					"provider dependency cycle: %s", strings.Join(parts, " -> "))
			}
			if state[edge.target] == 0 {
				stackFields = append(stackFields, edge.field)
				if err := visit(edge.target); err != nil {
					return err
				}
				stackFields = stackFields[:len(stackFields)-1]
			}
		}
		stack = stack[:len(stack)-1]
		state[typ] = 2
		order = append(order, typ)
		return nil
	}
	candidates := slices.Clone(graph.orderedProviders)
	slices.SortFunc(candidates, less)
	for _, typ := range candidates {
		if state[typ] == 0 {
			if err := visit(typ); err != nil {
				return err
			}
		}
	}
	graph.orderedProviders = order
	// Controllers run after all providers; order their independent hooks
	// by module imports, retaining declaration order within a module.
	slices.SortStableFunc(graph.controllers, func(a, b metadata.Controller) int {
		if rank(a.Type) < rank(b.Type) {
			return -1
		}
		if rank(a.Type) > rank(b.Type) {
			return 1
		}
		return 0
	})
	return nil
}
