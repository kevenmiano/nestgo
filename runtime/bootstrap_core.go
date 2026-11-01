package runtime

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

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

func prepareBootstrap(packages []metadata.Package, onProvider func(reflect.Type), onPackage func()) (bootstrapGraph, error) {
	if len(packages) == 0 {
		// Code-first applications do not need provider allocation or module graph validation.
		return bootstrapGraph{}, nil
	}
	providerCount, controllerCount := 0, 0
	for _, pkg := range packages {
		providerCount += len(pkg.Providers)
		controllerCount += len(pkg.Controllers)
	}
	graph := bootstrapGraph{
		providers:        make(map[reflect.Type]reflect.Value, providerCount),
		orderedProviders: make([]reflect.Type, 0, providerCount),
		controllers:      make([]metadata.Controller, 0, controllerCount),
	}
	for _, pkg := range packages {
		for _, p := range pkg.Providers {
			if p.Type == nil || p.Type.Kind() != reflect.Ptr || p.Type.Elem().Kind() != reflect.Struct {
				return bootstrapGraph{}, fmt.Errorf("invalid provider %v: must be pointer to struct", p.Type)
			}
			if _, exists := graph.providers[p.Type]; exists {
				return bootstrapGraph{}, fmt.Errorf("duplicate provider: %v", p.Type)
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
		return bootstrapGraph{}, err
	}
	graph.visibility = visibility
	slices.SortFunc(graph.orderedProviders, func(a, b reflect.Type) int {
		return strings.Compare(a.String(), b.String())
	})
	for _, typ := range graph.orderedProviders {
		if err := graph.inject(graph.providers[typ]); err != nil {
			return bootstrapGraph{}, err
		}
	}
	if err := orderBootstrap(packages, &graph); err != nil {
		return bootstrapGraph{}, err
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
