package runtime

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/runtime/testdata/externalmodule"
)

type importedModuleConsumer struct {
	Repository *externalmodule.Repository
}
type importedModuleRoot struct{}

func TestImportedModuleAcrossPackages(t *testing.T) {
	repo := reflect.TypeOf((*externalmodule.Repository)(nil))
	external := reflect.TypeOf((*externalmodule.Module)(nil))
	service := reflect.TypeOf((*importedModuleConsumer)(nil))
	root := reflect.TypeOf((*importedModuleRoot)(nil))
	remote := metadata.Package{
		Providers: []metadata.Provider{{Type: repo}},
		Modules:   []metadata.Module{{Type: external, Providers: []reflect.Type{repo}, Exports: []reflect.Type{repo}}},
	}
	local := metadata.Package{
		Providers: []metadata.Provider{{Type: service}},
		Modules:   []metadata.Module{{Type: root, Providers: []reflect.Type{service}, Imports: []reflect.Type{external}}},
	}
	if _, err := newHandler(Options{}, remote, local); err != nil {
		t.Fatal(err)
	}
	local.Modules[0].Imports = nil
	if _, err := newHandler(Options{}, remote, local); err == nil || !strings.Contains(err.Error(), "missing required dependency") {
		t.Fatalf("expected cross-package visibility denial, got %v", err)
	}
}
