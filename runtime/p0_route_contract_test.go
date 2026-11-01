package runtime

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type unsafeConcreteError struct{}

func (unsafeConcreteError) Error() string { return "concrete error" }

type p0SignatureController struct{}

func (*p0SignatureController) OnlyError() error { return errors.New("failure") }
func (*p0SignatureController) ConcreteError() (string, unsafeConcreteError) {
	return "", unsafeConcreteError{}
}
func (*p0SignatureController) Parameter(n int) string         { return "ok" }
func (*p0SignatureController) Valid(s string) (string, error) { return s, nil }

func TestP0RejectUnsafeHandlerContractBeforeServing(t *testing.T) {
	tests := []struct {
		name       string
		handler    string
		parameters []metadata.Parameter
		want       string
	}{
		{name: "error-only return", handler: "OnlyError", want: "invalid signature"},
		{name: "concrete error return", handler: "ConcreteError", want: "invalid signature"},
		{name: "invalid path binding type", handler: "Parameter", parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "n"}}, want: "requires a string"},
		{name: "duplicate argument binding", handler: "Valid", parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}, {Index: 0, Kind: "Query", Name: "q"}}, want: "duplicate parameter index"},
		{name: "unsupported decorator", handler: "Valid", parameters: []metadata.Parameter{{Index: 0, Kind: "Unknown", Name: "x"}}, want: "unsupported parameter decorator"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &capturedAdapter{}
			err := registerRoutes(adapter, []metadata.Package{{Controllers: []metadata.Controller{{
				Type:   reflect.TypeOf((*p0SignatureController)(nil)),
				Prefix: "/test",
				Routes: []metadata.Route{{Method: "GET", Path: "/invalid", Handler: tc.handler, Parameters: tc.parameters}},
			}}}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v does not contain %q", err, tc.want)
			}
			if len(adapter.routes) != 0 {
				t.Fatalf("invalid route registered: %v", adapter.routes)
			}
		})
	}
}

func TestP0ValidErrorResultUsesStandardResponse(t *testing.T) {
	a := &capturedAdapter{}
	err := registerRoutes(a, []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*p0SignatureController)(nil)),
		Prefix: "/test",
		Routes: []metadata.Route{{Method: "GET", Path: "/{id}", Handler: "Valid", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}}},
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &capturedContext{params: map[string]string{"id": "ok"}}
	if err := a.routes["GET /test/{id}"](ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.status != 200 || ctx.body != "\"ok\"\n" {
		t.Fatalf("unexpected response: status=%d body=%q", ctx.status, ctx.body)
	}
}
