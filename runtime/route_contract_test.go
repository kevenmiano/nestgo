package runtime

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

func TestSharedRouteContract(t *testing.T) {
	route := metadata.Route{Method: "GET", Path: "/{id}", Handler: "Lookup",
		Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}}
	contract, err := compileRouteContract("/api", route, reflect.TypeOf(func(string) string { return "" }))
	if err != nil { t.Fatal(err) }
	if contract.Path != "/api/{id}" || contract.Key != "GET /api/{id}" || contract.Bindings[0].Name != "id" {
		t.Fatalf("unexpected route contract: %+v", contract)
	}
	for _, tc := range []struct { name string; method any; params []metadata.Parameter; want string }{
		{"invalid signature", func(){}, nil, "invalid signature Lookup"},
		{"invalid parameter index", func() string { return "" }, []metadata.Parameter{{Index: 1, Kind: "Query", Name: "q"}}, "invalid parameter index 1"},
		{"missing decorator", func(int) string { return "" }, nil, "missing parameter decorator for Lookup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compileRouteContract("/api", metadata.Route{Method:"GET", Path:"/", Handler:"Lookup", Parameters:tc.params}, reflect.TypeOf(tc.method))
			if err == nil || !strings.Contains(err.Error(), tc.want) { t.Fatalf("got %v; want %q", err, tc.want) }
		})
	}
}
