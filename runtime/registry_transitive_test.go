package runtime

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/runtime/testdata/externalmodule"
)

type transitiveExternalModule struct{}
type transitiveRootModule struct{}
type transitiveMiddleModule struct{}
type transitiveConsumer struct {
	Repo *externalmodule.Repository
}

var transitiveFixtureOnce sync.Once

func registerTransitiveFixture() {
	transitiveFixtureOnce.Do(func() {
		repoType := reflect.TypeOf((*externalmodule.Repository)(nil))
		externalType := reflect.TypeOf((*transitiveExternalModule)(nil))
		middleType := reflect.TypeOf((*transitiveMiddleModule)(nil))
		consumerType := reflect.TypeOf((*transitiveConsumer)(nil))
		metadata.Register("nestgo-transitive-provider-test", metadata.Package{
			Providers: []metadata.Provider{{Type: repoType}},
			Modules:   []metadata.Module{{Type: externalType, Providers: []reflect.Type{repoType}, Exports: []reflect.Type{repoType}}},
		})
		metadata.Register("nestgo-transitive-middle-test", metadata.Package{
			Modules: []metadata.Module{{Type: middleType, Imports: []reflect.Type{externalType}}},
		})
		metadata.Register("nestgo-transitive-root-test", metadata.Package{
			Providers: []metadata.Provider{{Type: consumerType}},
			Modules:   []metadata.Module{{Type: reflect.TypeOf((*transitiveRootModule)(nil)), Imports: []reflect.Type{middleType}, Providers: []reflect.Type{consumerType}}},
		})
	})
}

func TestRegistryImportClosureTraversesThreePackages(t *testing.T) {
	registerTransitiveFixture()
	packages, err := registryImportClosure([]string{"nestgo-transitive-root-test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 3 {
		t.Fatalf("transitive closure packages = %d, want 3", len(packages))
	}
}

func TestRegistryTransitiveImportDoesNotLeakProviderVisibility(t *testing.T) {
	registerTransitiveFixture()
	for name, bootstrap := range map[string]func() error{
		"legacy": func() error { _, err := FromRegistry("nestgo-transitive-root-test"); return err },
		"adapter": func() error {
			_, err := ApplicationFromRegistry(ApplicationOptions{}, "nestgo-transitive-root-test")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := bootstrap(); err == nil || !strings.Contains(err.Error(), "missing required dependency") {
				t.Fatalf("indirect import must not grant provider visibility: %v", err)
			}
		})
	}
}
