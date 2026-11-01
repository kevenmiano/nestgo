package runtime

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type factorySource struct{ Value string }
type factoryResult struct{ Value string }
type factoryConsumer struct{ Result *factoryResult }

func TestGeneratedFactoryProviderInjected(t *testing.T) {
	source := &factorySource{Value: "ready"}
	pkg := metadata.Package{Providers: []metadata.Provider{
		{Type: reflect.TypeOf((*factoryResult)(nil)), Factory: func(resolve func(reflect.Type) (any, bool)) (any, error) {
			value, ok := resolve(reflect.TypeOf((*factorySource)(nil)))
			if !ok { return nil, fmt.Errorf("source missing") }
			return &factoryResult{Value: value.(*factorySource).Value}, nil
		}},
		{Type: reflect.TypeOf((*factoryConsumer)(nil))},
	}}
	if _, err := newHandler(handlerOptions{Instances: []any{source}}, pkg); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedFactoryFailsClosed(t *testing.T) {
	pkg := metadata.Package{Providers: []metadata.Provider{{
		Type: reflect.TypeOf((*factoryResult)(nil)),
		Factory: func(resolve func(reflect.Type) (any, bool)) (any, error) {
			return nil, fmt.Errorf("database connections missing")
		},
	}}}
	if _, err := newHandler(handlerOptions{}, pkg); err == nil || !strings.Contains(err.Error(), "database connections missing") {
		t.Fatalf("factory error should stop bootstrap: %v", err)
	}
}


func TestSharedBootstrapRunsFactoryAcrossRuntimePaths(t *testing.T) {
	typ := reflect.TypeOf((*factoryResult)(nil))
	calls := 0
	pkg := metadata.Package{Providers: []metadata.Provider{{
		Type: typ,
		Factory: func(resolve func(reflect.Type) (any, bool)) (any, error) {
			calls++
			return &factoryResult{Value: "generated"}, nil
		},
	}}}
	graph, err := prepareBootstrap([]metadata.Package{pkg}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("factory invoked %d times, expected once", calls)
	}
	got := graph.providers[typ].Interface().(*factoryResult)
	if got.Value != "generated" {
		t.Fatalf("factory result not registered: %#v", got)
	}
}

func TestSharedBootstrapExplicitProviderOverridesFactory(t *testing.T) {
	typ := reflect.TypeOf((*factoryResult)(nil))
	external := &factoryResult{Value: "external"}
	pkg := metadata.Package{Providers: []metadata.Provider{{
		Type: typ,
		Factory: func(resolve func(reflect.Type) (any, bool)) (any, error) {
			return nil, fmt.Errorf("factory must not run when an explicit instance exists")
		},
	}}}
	graph, err := prepareBootstrapWithInstances([]metadata.Package{pkg}, []any{external}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := graph.providers[typ].Interface().(*factoryResult); got != external {
		t.Fatalf("factory replaced external override: %#v", got)
	}
}
