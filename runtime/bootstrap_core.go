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
	// Retain only caller-owned or factory-constructed resources on errors.
	allocated []any
}

// prepareBootstrap is retained for the adapter runtime while both paths share
// validation, allocation, deterministic injection and generated factories.
func prepareBootstrap(packages []metadata.Package, onProvider func(reflect.Type), onPackage func()) (bootstrapGraph, error) {
	return prepareBootstrapWithInstances(packages, nil, onProvider, onPackage)
}

func prepareBootstrapWithInstances(packages []metadata.Package, instances []any, onProvider func(reflect.Type), onPackage func()) (bootstrapGraph, error) {
	if len(packages) == 0 && len(instances) == 0 {
		// Pure code-first apps skip metadata and DI allocation entirely.
		return bootstrapGraph{}, nil
	}
	providerCount, controllerCount := len(instances), 0
	for _, pkg := range packages {
		providerCount += len(pkg.Providers)
		controllerCount += len(pkg.Controllers)
	}
	graph := bootstrapGraph{
		providers:        make(map[reflect.Type]reflect.Value, providerCount),
		orderedProviders: make([]reflect.Type, 0, providerCount),
		controllers:      make([]metadata.Controller, 0, controllerCount),
	}
	for _, external := range instances {
		if external == nil {
			return graph, fmt.Errorf("nil external provider instance")
		}
		instance := reflect.ValueOf(external)
		if instance.Kind() != reflect.Ptr || instance.IsNil() || instance.Elem().Kind() != reflect.Struct {
			return graph, fmt.Errorf("external provider must be a non-nil pointer to struct: %T", external)
		}
		if _, exists := graph.providers[instance.Type()]; exists {
			return graph, fmt.Errorf("duplicate external provider: %s", instance.Type())
		}
		graph.providers[instance.Type()] = instance
		graph.orderedProviders = append(graph.orderedProviders, instance.Type())
		graph.allocated = append(graph.allocated, external)
	}
	declaredProviders := make(map[reflect.Type]bool)
	factoryProviders := make(map[reflect.Type]metadata.Provider)
	for _, pkg := range packages {
		for _, p := range pkg.Providers {
			if p.Type == nil || p.Type.Kind() != reflect.Ptr || p.Type.Elem().Kind() != reflect.Struct {
				return graph, fmt.Errorf("invalid provider %v: must be pointer to struct", p.Type)
			}
			if declaredProviders[p.Type] {
				return graph, fmt.Errorf("duplicate provider: %v", p.Type)
			}
			declaredProviders[p.Type] = true
			if p.Factory != nil {
				factoryProviders[p.Type] = p
			}
			if _, exists := graph.providers[p.Type]; !exists {
				graph.providers[p.Type] = reflect.New(p.Type.Elem())
				graph.orderedProviders = append(graph.orderedProviders, p.Type)
				if onProvider != nil {
					onProvider(p.Type)
				}
			}
		}
		graph.controllers = append(graph.controllers, pkg.Controllers...)
		if onPackage != nil {
			onPackage()
		}
	}
	// Resolve generated providers only after all dependencies are registered.
	// Caller-owned explicit instances override generated factories.
	for _, typ := range graph.orderedProviders {
		factory, exists := factoryProviders[typ]
		if !exists {
			continue
		}
		supplied := false
		for _, external := range instances {
			if reflect.TypeOf(external) == typ {
				supplied = true
				break
			}
		}
		if supplied {
			continue
		}
		result, err := factory.Factory(func(dependency reflect.Type) (any, bool) {
			value, found := graph.providers[dependency]
			if !found || !value.IsValid() {
				return nil, false
			}
			return value.Interface(), true
		})
		if err != nil {
			return graph, fmt.Errorf("provider factory %s: %w", typ, err)
		}
		value := reflect.ValueOf(result)
		if !value.IsValid() || value.Type() != typ || value.IsNil() {
			return graph, fmt.Errorf("provider factory %s returned wrong or nil type", typ)
		}
		graph.providers[typ] = value
		graph.allocated = append(graph.allocated, result)
	}
	visibility, err := moduleVisibility(packages)
	if err != nil {
		return graph, err
	}
	graph.visibility = visibility
	slices.SortFunc(graph.orderedProviders, func(a, b reflect.Type) int {
		return strings.Compare(a.String(), b.String())
	})
	for _, typ := range graph.orderedProviders {
		if err := graph.inject(graph.providers[typ]); err != nil {
			return graph, err
		}
	}
	if err := orderBootstrap(packages, &graph); err != nil {
		return graph, err
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
