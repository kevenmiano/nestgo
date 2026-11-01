package runtime

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type scopedRepo struct{}
type scopedConsumer struct {
	Repo *scopedRepo `nestgo:"inject"`
}
type scopedModuleA struct{}
type scopedModuleB struct{}

func TestModuleProviderVisibility(t *testing.T) {
	repo := reflect.TypeOf((*scopedRepo)(nil))
	consumer := reflect.TypeOf((*scopedConsumer)(nil))
	a := reflect.TypeOf((*scopedModuleA)(nil))
	b := reflect.TypeOf((*scopedModuleB)(nil))
	for _, tc := range []struct {
		name               string
		imported, exported bool
		want               string
	}{
		{"valid", true, true, ""},
		{"no-import", false, true, "missing required dependency"},
		{"no-export", true, false, "missing required dependency"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := metadata.Module{Type: a, Providers: []reflect.Type{repo}}
			if tc.exported {
				source.Exports = []reflect.Type{repo}
			}
			target := metadata.Module{Type: b, Providers: []reflect.Type{consumer}}
			if tc.imported {
				target.Imports = []reflect.Type{a}
			}
			pkg := metadata.Package{Providers: []metadata.Provider{{Type: repo}, {Type: consumer}}, Modules: []metadata.Module{source, target}}
			_, err := New(WithPackages(pkg))
			if tc.want == "" && err != nil {
				t.Fatal(err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
}
