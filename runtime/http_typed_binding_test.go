package runtime

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

type typedBenchDTO struct {
	Name string `json:"name"`
}

type typedBenchController struct{}

func (*typedBenchController) ByID(id string) string { return id }
func (*typedBenchController) Search(q string) string { return q }
func (*typedBenchController) Create(v typedBenchDTO) string { return v.Name }
func (*typedBenchController) Fail(q string) (string, error) {
	return "", errors.New("application failure")
}

type typedBenchContext struct {
	*capturedContext
	body  []byte
	query string
}

func (c *typedBenchContext) Body() []byte { return c.body }
func (c *typedBenchContext) Query(string) string { return c.query }

func typedBenchRoutes(useStatic bool) (*capturedAdapter, error) {
	routes := []metadata.Route{
		{Method: "GET", Path: "/path/{id}", Handler: "ByID", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}},
		{Method: "GET", Path: "/query", Handler: "Search", Parameters: []metadata.Parameter{{Index: 0, Kind: "Query", Name: "q"}}},
		{Method: "POST", Path: "/body", Handler: "Create", Parameters: []metadata.Parameter{{Index: 0, Kind: "Body"}}},
	}
	if useStatic {
		routes[0].StaticHTTP = func(receiver any, ctx httpadapter.Context) (any, int, error) {
			return receiver.(*typedBenchController).ByID(ctx.Param("id")), 0, nil
		}
		routes[1].StaticHTTP = func(receiver any, ctx httpadapter.Context) (any, int, error) {
			return receiver.(*typedBenchController).Search(ctx.Query("q")), 0, nil
		}
		routes[2].StaticHTTP = func(receiver any, ctx httpadapter.Context) (any, int, error) {
			dto, code, err := httpadapter.DecodeBody[typedBenchDTO](ctx)
			if err != nil {
				return nil, code, err
			}
			return receiver.(*typedBenchController).Create(dto), 0, nil
		}
	}
	adapter := &capturedAdapter{}
	packages := []metadata.Package{{Controllers: []metadata.Controller{{
		Type: reflect.TypeOf((*typedBenchController)(nil)), Prefix: "/typed", Routes: routes,
	}}}}
	if err := registerRoutes(adapter, packages); err != nil {
		return nil, err
	}
	return adapter, nil
}

func TestTypedHTTPDispatchAndErrors(t *testing.T) {
	adapter, err := typedBenchRoutes(true)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, route, query, body, id, want string
		status int
	}{
		{"path", "GET /typed/path/{id}", "", "", "Ada", "\"Ada\"\n", 200},
		{"query", "GET /typed/query", "Grace", "", "", "\"Grace\"\n", 200},
		{"body", "POST /typed/body", "", `{"name":"Ada"}`, "", "\"Ada\"\n", 201},
		{"invalid_json", "POST /typed/body", "", "{", "", "invalid JSON:", 400},
		{"large_body", "POST /typed/body", "", string(make([]byte, 1<<20+1)), "", "request body too large\n", 413},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &typedBenchContext{
				capturedContext: &capturedContext{params: map[string]string{"id": tc.id}},
				body: []byte(tc.body), query: tc.query,
			}
			if err := adapter.routes[tc.route](ctx); err != nil {
				t.Fatal(err)
			}
			if ctx.status != tc.status {
				t.Fatalf("status=%d, want %d", ctx.status, tc.status)
			}
			if tc.name == "invalid_json" {
				if len(ctx.body) == 0 || len(ctx.capturedContext.body) < len(tc.want) || ctx.capturedContext.body[:len(tc.want)] != tc.want {
					t.Fatalf("unexpected invalid JSON response: %q", ctx.capturedContext.body)
				}
			} else if ctx.capturedContext.body != tc.want {
				t.Fatalf("body=%q, want %q", ctx.capturedContext.body, tc.want)
			}
		})
	}
}

func TestTypedHTTPDispatchConcurrent(t *testing.T) {
	adapter, err := typedBenchRoutes(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := adapter.routes["GET /typed/path/{id}"]
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := &typedBenchContext{capturedContext: &capturedContext{params: map[string]string{"id": "ok"}}}
			if err := handler(ctx); err != nil {
				errs <- err
			} else if ctx.capturedContext.body != "\"ok\"\n" {
				errs <- errors.New("unexpected concurrent response")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func BenchmarkHTTPBinding(b *testing.B) {
	for _, mode := range []struct {
		name string
		static bool
	}{
		{"typed", true},
		{"reflection", false},
	} {
		adapter, err := typedBenchRoutes(mode.static)
		if err != nil {
			b.Fatal(err)
		}
		for _, scenario := range []struct {
			name, route, payload, query string
		}{
			{"path", "GET /typed/path/{id}", "", ""},
			{"query", "GET /typed/query", "", "needle"},
			{"body", "POST /typed/body", `{"name":"Ada"}`, ""},
		} {
			b.Run(scenario.name+"/"+mode.name, func(b *testing.B) {
				handler := adapter.routes[scenario.route]
				ctx := &typedBenchContext{
					capturedContext: &capturedContext{params: map[string]string{"id": "42"}},
					query: scenario.query, body: []byte(scenario.payload),
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ctx.capturedContext.body = ""
					if err := handler(ctx); err != nil {
						b.Fatal(err)
					}
				}
				benchmarkHTTPResult = ctx.capturedContext.body
			})
		}
	}
}
