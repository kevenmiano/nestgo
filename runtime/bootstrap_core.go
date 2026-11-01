package runtime

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/kevenmiano/nestgo/metadata"
)

// bootstrapGraph owns validation, allocation and field injection for all runtime entry points.
// HTTP transport and lifecycle ownership remain responsibilities of the callers.
type bootstrapGraph struct {
	providers        map[reflect.Type]reflect.Value
	orderedProviders []reflect.Type
	controllers      []metadata.Controller
	visibility       map[reflect.Type]map[reflect.Type]bool
}

func prepareBootstrap(packages []metadata.Package, onProvider func(reflect.Type), onPackage func()) (*bootstrapGraph, error) {
	graph := &bootstrapGraph{providers: make(map[reflect.Type]reflect.Value)}
	for _, pkg := range packages {
		for _, p := range pkg.Providers {
			if p.Type == nil || p.Type.Kind() != reflect.Ptr || p.Type.Elem().Kind() != reflect.Struct {
				return nil, fmt.Errorf("invalid provider %v: must be pointer to struct", p.Type)
			}
			if _, exists := graph.providers[p.Type]; exists {
				return nil, fmt.Errorf("duplicate provider: %v", p.Type)
			}
			graph.providers[p.Type] = reflect.New(p.Type.Elem())
			graph.orderedProviders = append(graph.orderedProviders, p.Type)
			if onProvider != nil {
				onProvider(p.Type)
			}
		}
		graph.controllers = append(graph.controllers, pkg.Controllers...)
		if onPackage != nil {
			onPackage()
		}
	}
	visibility, err := moduleVisibility(packages)
	if err != nil {
		return nil, err
	}
	graph.visibility = visibility
	sort.Slice(graph.orderedProviders, func(i, j int) bool {
		return graph.orderedProviders[i].String() < graph.orderedProviders[j].String()
	})
	for _, typ := range graph.orderedProviders {
		if err := graph.inject(graph.providers[typ]); err != nil {
			return nil, err
		}
	}
	return graph, nil
}

func (g *bootstrapGraph) inject(instance reflect.Value) error {
	return injectFields(instance, g.providers, g.orderedProviders, g.visibility)
}

func (g *bootstrapGraph) newController(typ reflect.Type) (reflect.Value, error) {
	if typ == nil || typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("invalid controller %v", typ)
	}
	instance := reflect.New(typ.Elem())
	if err := g.inject(instance); err != nil {
		return reflect.Value{}, err
	}
	return instance, nil
}
