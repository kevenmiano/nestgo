package runtime

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

type fastStringController struct{}

func (*fastStringController) ByID(id string) string            { return id }
func (*fastStringController) ByQuery(q string) (string, error) { return q, nil }
func (*fastStringController) ByHeader(header string) string    { return header }
func (*fastStringController) Implicit(id string) string        { return id }
func (*fastStringController) Fail(q string) (string, error) {
	return "", errors.New("private application failure")
}

type fastStringContext struct {
	*capturedContext
	query, header string
}

func (c *fastStringContext) Query(string) string  { return c.query }
func (c *fastStringContext) Header(string) string { return c.header }

func fastStringTestRoutes() (*capturedAdapter, error) {
	adapter := &capturedAdapter{}
	pkg := metadata.Package{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*fastStringController)(nil)),
		Prefix: "/fast",
		Routes: []metadata.Route{
			{Method: "GET", Path: "/{id}", Handler: "ByID", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}}},
			{Method: "GET", Path: "/search", Handler: "ByQuery", Parameters: []metadata.Parameter{{Index: 0, Kind: "Query", Name: "q"}}},
			{Method: "GET", Path: "/header", Handler: "ByHeader", Parameters: []metadata.Parameter{{Index: 0, Kind: "Headers", Name: "X-Key"}}},
			{Method: "GET", Path: "/implicit/{id}", Handler: "Implicit"},
			{Method: "GET", Path: "/fail", Handler: "Fail", Parameters: []metadata.Parameter{{Index: 0, Kind: "Query", Name: "q"}}},
		},
	}}}
	if err := registerRoutes(adapter, []metadata.Package{pkg}); err != nil {
		return nil, err
	}
	return adapter, nil
}

func TestStringInvokerContractAndErrors(t *testing.T) {
	adapter, err := fastStringTestRoutes()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, route, id, query, header, want string
		status                               int
	}{
		{"param", "GET /fast/{id}", "Ada", "", "", "\"Ada\"\n", 200},
		{"query", "GET /fast/search", "", "Grace", "", "\"Grace\"\n", 200},
		{"header", "GET /fast/header", "", "", "Token", "\"Token\"\n", 200},
		{"implicit", "GET /fast/implicit/{id}", "Path", "", "", "\"Path\"\n", 200},
		{"error", "GET /fast/fail", "", "bad", "", "Internal server error", 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fastStringContext{capturedContext: &capturedContext{params: map[string]string{"id": tc.id}}, query: tc.query, header: tc.header}
			if err := adapter.routes[tc.route](ctx); err != nil {
				t.Fatal(err)
			}
			if ctx.status != tc.status || !strings.Contains(ctx.body, tc.want) {
				t.Fatalf("status=%d body=%q, want %d %q", ctx.status, ctx.body, tc.status, tc.want)
			}
			if tc.name == "error" && strings.Contains(ctx.body, "private application failure") {
				t.Fatal("internal handler error leaked")
			}
		})
	}
}

func TestStringInvokerFallsBackForBodyAndOtherTypes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fn     any
		marker metadata.Parameter
	}{
		{"body", func(string) string { return "" }, metadata.Parameter{Index: 0, Kind: "Body"}},
		{"non-string", func(int) string { return "" }, metadata.Parameter{Index: 0, Kind: "Query", Name: "q"}},
		{"different-result", func(string) int { return 1 }, metadata.Parameter{Index: 0, Kind: "Query", Name: "q"}},
		{"multiple-args", func(string, string) string { return "" }, metadata.Parameter{Index: 0, Kind: "Query", Name: "q"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if invoke := compileStringRouteInvoker(reflect.ValueOf(tc.fn), metadata.Route{}, map[int]metadata.Parameter{0: tc.marker}); invoke != nil {
				t.Fatal("unsupported binding unexpectedly enabled typed fast path")
			}
		})
	}
}

func TestStringInvokerConcurrentRequests(t *testing.T) {
	adapter, err := fastStringTestRoutes()
	if err != nil {
		t.Fatal(err)
	}
	handler := adapter.routes["GET /fast/search"]
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := &fastStringContext{capturedContext: &capturedContext{}, query: "safe"}
			if err := handler(ctx); err != nil {
				t.Error(err)
			} else if ctx.body != "\"safe\"\n" {
				t.Errorf("incorrect concurrent response: %q", ctx.body)
			}
		}()
	}
	wg.Wait()
}

var _ httpadapter.Context = (*fastStringContext)(nil)
