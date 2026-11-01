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
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*externalConfigConsumer)(nil))}}}
	app, err := New(WithPackages(pkg))
	if err != nil {
		t.Fatalf("runtime providers unavailable to DI: %v", err)
	}
	defer app.Close()
	if app.Config == nil || app.Connections == nil {
		t.Fatal("missing database/config services")
	}
}

func TestInvalidExternalProvidersRejected(t *testing.T) {
	for _, instance := range []any{nil, 42, (*config.Service)(nil)} {
		if _, err := New(WithProvider(instance)); err == nil {
			t.Fatalf("accepted invalid external provider %T", instance)
		}
	}
}
