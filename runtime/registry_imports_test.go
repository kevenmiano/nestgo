package runtime

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/runtime/testdata/externalmodule"
)

type registryRootModule struct{}
type registryMissingImportModule struct{}
type registryConsumer struct {
	Repo *externalmodule.Repository
}

var registerClosureFixtures sync.Once

func TestFromRegistryLoadsImportedPackage(t *testing.T) {
	registerClosureFixtures.Do(func() {
		repo := reflect.TypeOf((*externalmodule.Repository)(nil))
		external := reflect.TypeOf((*externalmodule.Module)(nil))
		consumer := reflect.TypeOf((*registryConsumer)(nil))
		metadata.Register("nestgo-closure-external-test", metadata.Package{
			Providers: []metadata.Provider{{Type: repo}},
			Modules:   []metadata.Module{{Type: external, Providers: []reflect.Type{repo}, Exports: []reflect.Type{repo}}},
		})
		metadata.Register("nestgo-closure-root-test", metadata.Package{
			Providers: []metadata.Provider{{Type: consumer}},
			Modules:   []metadata.Module{{Type: reflect.TypeOf((*registryRootModule)(nil)), Providers: []reflect.Type{consumer}, Imports: []reflect.Type{external}}},
		})
		metadata.Register("nestgo-closure-no-import-test", metadata.Package{
			Providers: []metadata.Provider{{Type: consumer}},
			Modules:   []metadata.Module{{Type: reflect.TypeOf((*registryMissingImportModule)(nil)), Providers: []reflect.Type{consumer}}},
		})
	})
	if _, err := FromRegistry("nestgo-closure-root-test"); err != nil {
		t.Fatalf("root module must load imported provider metadata automatically: %v", err)
	}
	if _, err := ApplicationFromRegistry(ApplicationOptions{}, "nestgo-closure-root-test"); err != nil {
		t.Fatalf("application must resolve imported provider packages: %v", err)
	}
	if _, err := FromRegistry("nestgo-closure-no-import-test"); err == nil || !strings.Contains(err.Error(), "missing required dependency") {
		t.Fatalf("unimported provider must remain inaccessible: %v", err)
	}
}

func TestApplicationFromRegistryRejectsInvalidImportedPackage(t *testing.T) {
	type externalModule struct{}
	type rootModule struct{}
	external := reflect.TypeOf((*externalModule)(nil))
	metadata.Register("nestgo-app-invalid-import-test", metadata.Package{
		Providers: []metadata.Provider{{Type: reflect.TypeOf(123)}},
		Modules: []metadata.Module{{
			Type: external,
			Providers: []reflect.Type{reflect.TypeOf(123)},
		}},
	})
	metadata.Register("nestgo-app-import-root-test", metadata.Package{
		Modules: []metadata.Module{{
			Type: reflect.TypeOf((*rootModule)(nil)),
			Imports: []reflect.Type{external},
		}},
	})
	if _, err := ApplicationFromRegistry(ApplicationOptions{}, "nestgo-app-import-root-test"); err == nil || !strings.Contains(err.Error(), "invalid provider") {
		t.Fatalf("invalid imported package must fail at application bootstrap: %v", err)
	}
}
