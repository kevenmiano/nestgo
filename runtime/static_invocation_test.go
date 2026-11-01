package runtime

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type staticTestController struct{}

func (*staticTestController) Ready() string { return "ready" }
func (*staticTestController) Echo(v string) string { return v }

func TestStaticInvocationAndDynamicFallback(t *testing.T) {
	calls := 0
	pkg := metadata.Package{Controllers: []metadata.Controller{{
		Type: reflect.TypeOf((*staticTestController)(nil)),
		Prefix: "/test",
		Routes: []metadata.Route{
			{Method: "GET", Path: "/ready", Handler: "Ready", StaticInvoke: func(v any) (any, error) {
				calls++
				return v.(*staticTestController).Ready(), nil
			}},
			{Method: "GET", Path: "/echo/{id}", Handler: "Echo", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}},
		},
	}}}
	adapter := &capturedAdapter{}
	app, err := New(WithAdapter(adapter), WithPackages(pkg))
	if err != nil { t.Fatal(err) }
	defer app.Shutdown(context.Background())
	for _, tc := range []struct { path, want string; params map[string]string }{
		{"/test/ready", "\"ready\"\n", nil},
		{"/test/echo/{id}", "\"hello\"\n", map[string]string{"id": "hello"}},
	} {
		ctx := &capturedContext{params: tc.params}
		if err := adapter.routes["GET "+tc.path](ctx); err != nil { t.Fatal(err) }
		if ctx.status != 200 || ctx.body != tc.want {
			t.Fatalf("%s: status=%d body=%q", tc.path, ctx.status, ctx.body)
		}
	}
	if calls != 1 { t.Fatalf("static invoker calls=%d, want 1", calls) }
}
