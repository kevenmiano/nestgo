package runtime

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	netadapter "github.com/kevenmiano/nestgo/platform/nethttp"
	"github.com/kevenmiano/nestgo/runtime/testdata/externalmodule"
)

type exportBoundaryModule struct{}
type exportBoundaryConsumerModule struct{}
type exportBoundaryConsumer struct {
	Repository *externalmodule.Repository
}
type exportBoundaryOptionalConsumer struct {
	Repository *externalmodule.Repository `nestgo:"optional"`
}

func TestCrossPackageConcreteProviderExportBoundary(t *testing.T) {
	repository := reflect.TypeOf((*externalmodule.Repository)(nil))
	source := reflect.TypeOf((*exportBoundaryModule)(nil))
	target := reflect.TypeOf((*exportBoundaryConsumerModule)(nil))
	for _, tc := range []struct {
		name     string
		exported bool
		optional bool
		wantFail bool
	}{
		{name: "exported", exported: true},
		{name: "private", wantFail: true},
		{name: "private-optional", optional: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			consumer := reflect.TypeOf((*exportBoundaryConsumer)(nil))
			if tc.optional {
				consumer = reflect.TypeOf((*exportBoundaryOptionalConsumer)(nil))
			}
			providerModule := metadata.Module{Type: source, Providers: []reflect.Type{repository}}
			if tc.exported {
				providerModule.Exports = []reflect.Type{repository}
			}
			packages := []metadata.Package{
				{Providers: []metadata.Provider{{Type: repository}}, Modules: []metadata.Module{providerModule}},
				{Providers: []metadata.Provider{{Type: consumer}}, Modules: []metadata.Module{{
					Type: target, Imports: []reflect.Type{source}, Providers: []reflect.Type{consumer},
				}}},
			}
			cases := []struct {
				name  string
				start func() error
			}{
				{name: "legacy", start: func() error {
					_, err := newHandler(handlerOptions{}, packages...)
					return err
				}},
				{name: "fasthttp", start: func() error {
					_, err := NewApplication(ApplicationOptions{Adapter: fastadapter.New()}, packages...)
					return err
				}},
				{name: "nethttp", start: func() error {
					_, err := NewApplication(ApplicationOptions{Adapter: netadapter.New()}, packages...)
					return err
				}},
			}
			for _, boot := range cases {
				t.Run(boot.name, func(t *testing.T) {
					err := boot.start()
					if tc.wantFail {
						if err == nil || !strings.Contains(err.Error(), "missing required dependency") {
							t.Fatalf("private cross-package concrete provider must be inaccessible: %v", err)
						}
					} else if err != nil {
						t.Fatalf("valid scoped dependency should bootstrap: %v", err)
					}
				})
			}
		})
	}
}
