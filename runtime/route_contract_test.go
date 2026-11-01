package runtime

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

func TestSharedRouteContract(t *testing.T) {
	t.Run("valid decorated parameter", func(t *testing.T) {
		route := metadata.Route{Method: "GET", Path: "/{id}", Handler: "Lookup",
			Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}}
		method := reflect.TypeOf(func(string) string { return "" })
		contract, err := compileRouteContract("/api", route, method)
		if err != nil { t.Fatal(err) }
		if contract.Path != "/api/{id}" || contract.Key != "GET /api/{id}" || contract.Bindings[0].Name != "id" {
			t.Fatalf("unexpected route contract: %+v", contract)
		}
	})
	for _, tc := range []struct {
		name string
		method any
		params []metadata.Parameter
		legacy string
		adapter string
	}{
		{"invalid signature", func() {}, nil, "unsupported signature Lookup", "invalid signature Lookup"},
		{"invalid parameter index", func() string { return "" }, []metadata.Parameter{{Index: 1, Kind: "Query", Name: "q"}}, "invalid parameter index 1", "invalid parameter index 1"},
		{"missing decorator", func(int) string { return "" }, nil, "missing decorator for argument 0 of Lookup", "missing parameter decorator for Lookup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compileRouteContract("/api", metadata.Route{Method: "GET", Path: "/", Handler: "Lookup", Parameters: tc.params}, reflect.TypeOf(tc.method))
			if err == nil { t.Fatal("expected route validation failure") }
			if got := formatRouteContractError(err, true).Error(); !strings.Contains(got, tc.legacy) { t.Fatalf("legacy error %q; want %q", got, tc.legacy) }
			if got := formatRouteContractError(err, false).Error(); !strings.Contains(got, tc.adapter) { t.Fatalf("adapter error %q; want %q", got, tc.adapter) }
		})
	}
}

