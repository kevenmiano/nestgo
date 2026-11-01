package runtime

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/runtime/testdata/externalmodule"
)

type crossGraphModule struct{}

func TestCrossPackageModuleCycleRejected(t *testing.T) {
	external := reflect.TypeOf((*externalmodule.Module)(nil))
	local := reflect.TypeOf((*crossGraphModule)(nil))
	first := metadata.Package{Modules: []metadata.Module{{Type: external, Imports: []reflect.Type{local}}}}
	second := metadata.Package{Modules: []metadata.Module{{Type: local, Imports: []reflect.Type{external}}}}
	_, err := New(WithPackages(first, second))
	if err == nil || !strings.Contains(err.Error(), "module import cycle:") {
		t.Fatalf("expected cross-package cycle error, got %v", err)
	}
}

func TestUnusedModuleInvalidExportsRejected(t *testing.T) {
	external := reflect.TypeOf((*externalmodule.Module)(nil))
	repo := reflect.TypeOf((*externalmodule.Repository)(nil))
	pkg := metadata.Package{Modules: []metadata.Module{{Type: external, Exports: []reflect.Type{repo}}}}
	_, err := New(WithPackages(pkg))
	if err == nil || !strings.Contains(err.Error(), "exports undeclared") {
		t.Fatalf("expected validation of unused module exports, got %v", err)
	}
}

func TestUnusedModuleUnknownImportRejected(t *testing.T) {
	external := reflect.TypeOf((*externalmodule.Module)(nil))
	local := reflect.TypeOf((*crossGraphModule)(nil))
	pkg := metadata.Package{Modules: []metadata.Module{{Type: local, Imports: []reflect.Type{external}}}}
	_, err := New(WithPackages(pkg))
	if err == nil || !strings.Contains(err.Error(), "imports unknown module") {
		t.Fatalf("expected validation of unused module imports, got %v", err)
	}
}
