package diagnostics

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type moduleAlpha struct{}
type moduleBeta struct{}
type moduleGamma struct{}
type moduleMissing struct{}
type exportedService struct{}
type notRegisteredService struct{}

func module(t any, name string, imports []reflect.Type, exports []reflect.Type) metadata.Module {
	return metadata.Module{Type: reflect.TypeOf(t), Name: name, Imports: imports, Exports: exports}
}

func TestModuleGraphDeterministicAcrossRegistrationOrder(t *testing.T) {
	alpha := module((*moduleAlpha)(nil), "alpha", []reflect.Type{reflect.TypeOf((*moduleBeta)(nil))}, []reflect.Type{reflect.TypeOf((*exportedService)(nil))})
	beta := module((*moduleBeta)(nil), "beta", []reflect.Type{reflect.TypeOf((*moduleAlpha)(nil))}, nil)
	gamma := module((*moduleGamma)(nil), "gamma", nil, nil)
	provider := metadata.Provider{Type: reflect.TypeOf((*exportedService)(nil)), Scope: "singleton"}
	first := AnalyzeModules(metadata.Package{Modules: []metadata.Module{alpha, beta, gamma}, Providers: []metadata.Provider{provider}})
	second := AnalyzeModules(metadata.Package{Modules: []metadata.Module{gamma, beta}, Providers: []metadata.Provider{provider}}, metadata.Package{Modules: []metadata.Module{alpha}})
	j1, _ := first.JSON()
	j2, _ := second.JSON()
	if !bytes.Equal(j1, j2) || first.DOT() != second.DOT() {
		t.Fatalf("module graph varies with registration order\n%s\n%s", j1, j2)
	}
	if len(first.Modules) != 3 || len(first.Imports) != 2 || len(first.Cycles) != 1 || len(first.Findings) != 0 {
		t.Fatalf("invalid module graph: %+v", first)
	}
	if len(first.Modules[0].Exports) != 1 {
		t.Fatalf("expected exported provider: %+v", first.Modules)
	}
}

func TestModuleGraphMissingImportsUnknownExportsAndSelfCycle(t *testing.T) {
	g := AnalyzeModules(metadata.Package{Modules: []metadata.Module{
		module((*moduleAlpha)(nil), "alpha", []reflect.Type{reflect.TypeOf((*moduleMissing)(nil)), reflect.TypeOf((*moduleAlpha)(nil))}, []reflect.Type{reflect.TypeOf((*notRegisteredService)(nil))}),
	}})
	kinds := map[string]bool{}
	for _, finding := range g.Findings {
		kinds[finding.Kind] = true
	}
	if !kinds["missing-module"] || !kinds["unknown-export"] || len(g.Cycles) != 1 {
		t.Fatalf("missing or cyclic module not reported: %+v", g)
	}
}

func TestLegacyModuleMetadataKeepsEmptyGraphEdges(t *testing.T) {
	g := AnalyzeModules(metadata.Package{Modules: []metadata.Module{{Type: reflect.TypeOf((*moduleAlpha)(nil)), Name: "legacy"}}})
	if len(g.Modules) != 1 || len(g.Imports) != 0 || len(g.Findings) != 0 || len(g.Cycles) != 0 {
		t.Fatalf("old generated metadata should still work: %+v", g)
	}
}

func TestModuleGraphInvalidDuplicateAndInvalidImport(t *testing.T) {
	alpha := module((*moduleAlpha)(nil), "alpha", []reflect.Type{reflect.TypeOf(int(0))}, nil)
	g := AnalyzeModules(metadata.Package{Modules: []metadata.Module{alpha, alpha, {Type: reflect.TypeOf(int(0)), Name: "invalid"}}})
	kinds := map[string]bool{}
	for _, f := range g.Findings {
		kinds[f.Kind] = true
	}
	for _, kind := range []string{"invalid-module", "invalid-import", "duplicate-module"} {
		if !kinds[kind] {
			t.Errorf("missing %s: %+v", kind, g.Findings)
		}
	}
}
