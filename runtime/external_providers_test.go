package runtime

import (
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/config"
	"github.com/kevenmiano/nestgo/database"
	"github.com/kevenmiano/nestgo/metadata"
)

type externalConfigConsumer struct {
	Config      *config.Service
	Connections *database.Connections
}

func TestInitializedConfigAndConnectionsInjected(t *testing.T) {
	settings := config.FromValues(map[string]string{"APP_NAME": "nestgo"})
	registry := database.NewConnections()
	defer registry.Close()
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*externalConfigConsumer)(nil))}}}
	if _, err := newHandler(handlerOptions{Instances: []any{settings, registry}}, pkg); err != nil {
		t.Fatalf("external providers unavailable to DI: %v", err)
	}
}

func TestInvalidExternalProvidersRejected(t *testing.T) {
	for _, instance := range []any{nil, 42, (*config.Service)(nil)} {
		if _, err := newHandler(handlerOptions{Instances: []any{instance}}); err == nil {
			t.Fatalf("accepted invalid external provider %T", instance)
		}
	}
}
