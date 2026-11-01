package runtime

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type staticTestController struct{}

func (*staticTestController) Ready() string        { return "ready" }
func (*staticTestController) Echo(v string) string { return v }

func TestStaticInvocationAndDynamicFallback(t *testing.T) {
	calls := 0
	pkg := metadata.Package{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*staticTestController)(nil)),
		Prefix: "/test",
		Routes: []metadata.Route{
			{Method: "GET", Path: "/ready", Handler: "Ready", StaticInvoke: func(v any) (any, error) {
				calls++
				return v.(*staticTestController).Ready(), nil
			}},
			{Method: "GET", Path: "/echo/{id}", Handler: "Echo", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}},
		},
	}}}
	h, err := newHandler(handlerOptions{}, pkg)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, want string }{
		{"/test/ready", "\"ready\"\n"},
		{"/test/echo/hello", "\"hello\"\n"},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != 200 || w.Body.String() != tc.want {
			t.Fatalf("%s: status=%d body=%q", tc.path, w.Code, w.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("static invoker calls=%d, want 1", calls)
	}
}
