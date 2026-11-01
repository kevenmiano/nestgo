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
	if _, err := newHandler(Options{Instances: []any{source}}, pkg); err != nil {
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
	if _, err := newHandler(Options{}, pkg); err == nil || !strings.Contains(err.Error(), "database connections missing") {
		t.Fatalf("factory error should stop bootstrap: %v", err)
	}
}
