package runtime

import (
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

// Untrusted or malformed generated metadata must fail at bootstrap rather than
// crash the request path. Go test exercises these seeds; go test -fuzz extends
// them with randomized metadata.
func FuzzRouteContractRejectsInvalidBindings(f *testing.F) {
	f.Add("Param", 0, "id")
	f.Add("Query", 0, "q")
	f.Add("Body", 0, "")
	f.Add("Headers", 0, "x")
	f.Add("Param", -1, "id")
	f.Add("Query", 7, "q")
	f.Add("", 0, "")

	method := reflect.TypeOf(func(string) (string, error) { return "", nil })
	f.Fuzz(func(t *testing.T, kind string, index int, name string) {
		if len(kind) > 128 || len(name) > 128 {
			t.Skip()
		}
		route := metadata.Route{
			Method:  "GET",
			Path:    "/{id}",
			Handler: "Echo",
			Parameters: []metadata.Parameter{{Index: index, Kind: kind, Name: name}},
		}
		_, err := compileRouteContract("/api", route, method)
		validKind := kind == "Param" || kind == "Query" || kind == "Body"
		if err == nil && (index != 0 || !validKind) {
			t.Fatalf("accepted invalid binding: kind=%q index=%d", kind, index)
		}
		if err != nil && index == 0 && validKind {
			t.Fatalf("rejected valid binding: kind=%q: %v", kind, err)
		}
	})
}
