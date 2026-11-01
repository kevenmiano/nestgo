package runtime

import (
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type sharedBootstrapDependency struct{}
type sharedBootstrapConsumer struct{ Dependency *sharedBootstrapDependency }
type sharedBootstrapController struct{ Dependency *sharedBootstrapDependency }

func TestSharedBootstrapGraphInjectsProvidersAndControllers(t *testing.T) {
	pkg := metadata.Package{
		Providers: []metadata.Provider{
			{Type: reflect.TypeOf((*sharedBootstrapConsumer)(nil))},
			{Type: reflect.TypeOf((*sharedBootstrapDependency)(nil))},
		},
	}
	graph, err := prepareBootstrap([]metadata.Package{pkg}, nil, nil)
	if err != nil { t.Fatal(err) }
	consumer := graph.providers[reflect.TypeOf((*sharedBootstrapConsumer)(nil))].Interface().(*sharedBootstrapConsumer)
	if consumer.Dependency == nil { t.Fatal("provider dependency not injected") }
	controller, err := graph.newController(reflect.TypeOf((*sharedBootstrapController)(nil)))
	if err != nil { t.Fatal(err) }
	resolved := controller.Interface().(*sharedBootstrapController)
	if resolved.Dependency != consumer.Dependency { t.Fatal("controller and provider must share singleton instance") }
}

func TestSharedBootstrapGraphRejectsUnresolvedDependency(t *testing.T) {
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*sharedBootstrapConsumer)(nil))}}}
	if _, err := prepareBootstrap([]metadata.Package{pkg}, nil, nil); err == nil { t.Fatal("expected missing dependency") }
}

func TestSharedBootstrapGraphRejectsInvalidController(t *testing.T) {
	graph, err := prepareBootstrap(nil, nil, nil)
	if err != nil { t.Fatal(err) }
	if _, err := graph.newController(reflect.TypeOf(123)); err == nil { t.Fatal("expected invalid controller error") }
}
