package runtime

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type autoAppModule struct{}
type autoUsersModule struct{}
type autoBillingModule struct{}
type autoOtherRoot struct{}

func moduleType[T any]() reflect.Type {
	var pointer *T
	return reflect.TypeOf(pointer)
}

func TestCompiledRootDiscoveryLoadsTransitiveModules(t *testing.T) {
	users := moduleType[autoUsersModule]()
	billing := moduleType[autoBillingModule]()
	app := moduleType[autoAppModule]()
	registered := map[string]metadata.Package{
		"users": {Modules: []metadata.Module{{Type: users}}},
		"billing": {Modules: []metadata.Module{{Type: billing, Imports: []reflect.Type{users}}}},
		"app": {Modules: []metadata.Module{{Type: app, Imports: []reflect.Type{billing}}}},
		"unrelated-test-fixture": {Providers: []metadata.Provider{{Type: reflect.TypeOf(42)}}},
	}
	compiled := map[string]metadata.Package{
		"users": registered["users"],
		"billing": registered["billing"],
		"app": registered["app"],
	}
	packages, err := compiledRootClosure(compiled, registered)
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 3 {
		t.Fatalf("expected root + two transitive packages, got %d", len(packages))
	}
	if packages[0].Modules[0].Type != app {
		t.Fatalf("first package should be application root, got %v", packages[0].Modules)
	}
}

func TestCompiledRootDiscoveryRejectsMultipleIndependentApps(t *testing.T) {
	registered := map[string]metadata.Package{
		"app": {Modules: []metadata.Module{{Type: moduleType[autoAppModule]()}}},
		"other": {Modules: []metadata.Module{{Type: moduleType[autoOtherRoot]()}}},
	}
	_, err := compiledRootClosure(registered, registered)
	if err == nil || !strings.Contains(err.Error(), "exactly one application root") {
		t.Fatalf("expected explicit multi-root error, got %v", err)
	}
}

func TestCompiledRootDiscoveryRejectsUnreachablePackage(t *testing.T) {
	app := moduleType[autoAppModule]()
	registered := map[string]metadata.Package{
		"app": {Modules: []metadata.Module{{Type: app}}},
		"orphan": {Providers: []metadata.Provider{{Type: moduleType[autoOtherRoot]()}}},
	}
	if _, err := compiledRootClosure(registered, registered); err == nil || !strings.Contains(err.Error(), "not reachable") {
		t.Fatalf("expected orphan compiled metadata error, got %v", err)
	}
}

func TestCompiledRootDiscoverySupportsModulelessAppAndRejectsAmbiguity(t *testing.T) {
	single := map[string]metadata.Package{"app": {Controllers: []metadata.Controller{{Type: moduleType[autoAppModule]()}}}}
	loaded, err := compiledRootClosure(single, single)
	if err != nil || len(loaded) != 1 {
		t.Fatalf("single moduleless application should load: %v, %d", err, len(loaded))
	}
	ambiguous := map[string]metadata.Package{"app": {}, "other": {}}
	if _, err := compiledRootClosure(ambiguous, ambiguous); err == nil || !strings.Contains(err.Error(), "ambiguous compiled packages") {
		t.Fatalf("expected ambiguity error: %v", err)
	}
}

func TestCompiledRootDiscoveryRejectsMissingImportedModule(t *testing.T) {
	registered := map[string]metadata.Package{
		"app": {Modules: []metadata.Module{{Type: moduleType[autoAppModule](), Imports: []reflect.Type{moduleType[autoUsersModule]()}}}},
	}
	// Closure should be returned only with a valid graph; scope validation
	// additionally rejects an imported module that is not registered.
	pkgs, err := compiledRootClosure(registered, registered)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := moduleVisibility(pkgs); err == nil || !strings.Contains(err.Error(), "imports unknown module") {
		t.Fatalf("expected missing module diagnostic, got %v", err)
	}
}
