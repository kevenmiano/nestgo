package runtime

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

type pipelineBenchmarkController struct{}

func (*pipelineBenchmarkController) Lookup(id string) string { return id }
func (*pipelineBenchmarkController) Fail(id string) (string, error) {
	return "", errors.New("internal")
}

type benchmarkAllowGuard struct{}

func (*benchmarkAllowGuard) CanActivate(httpadapter.ExecutionContext) (bool, error) {
	return true, nil
}

type benchmarkIdentityPipe struct{}

func (*benchmarkIdentityPipe) Transform(value any, _ httpadapter.ArgumentMetadata) (any, error) {
	return value, nil
}

type benchmarkPassthroughFilter struct{}

func (*benchmarkPassthroughFilter) Catch(error, httpadapter.ExecutionContext) (bool, error) {
	return false, nil
}

func benchmarkNextMiddleware(_ httpadapter.Context, next httpadapter.Next) error {
	return next()
}

// BenchmarkRequestPipeline isolates request-pipeline overhead from transport
// and network overhead. All scenarios resolve the same route, marshal the same
// result and write to the same Context implementation. The setup is excluded
// from timings. Static calls represent compiler-generated dispatch; reflection
// is an explicit fallback comparison.
func BenchmarkRequestPipeline(b *testing.B) {
	guard := &benchmarkAllowGuard{}
	pipe := &benchmarkIdentityPipe{}
	filter := &benchmarkPassthroughFilter{}
	cases := []struct {
		name string
		static bool
		options []Option
	}{
		{name: "baseline/static", static: true},
		{name: "middleware_1/static", static: true, options: []Option{WithMiddleware(benchmarkNextMiddleware)}},
		{name: "middleware_3/static", static: true, options: []Option{WithMiddleware(benchmarkNextMiddleware, benchmarkNextMiddleware, benchmarkNextMiddleware)}},
		{name: "guard_1/static", static: true, options: []Option{WithGuards(guard)}},
		{name: "guard_3/static", static: true, options: []Option{WithGuards(guard, guard, guard)}},
		{name: "pipe_1/static", static: true, options: []Option{WithPipes(pipe)}},
		{name: "pipe_3/static", static: true, options: []Option{WithPipes(pipe, pipe, pipe)}},
		{name: "filter_1_success/static", static: true, options: []Option{WithFilters(filter)}},
		{name: "combined/static", static: true, options: []Option{
			WithMiddleware(benchmarkNextMiddleware), WithGuards(guard), WithPipes(pipe), WithFilters(filter),
		}},
		{name: "baseline/reflection"},
		{name: "combined/reflection", options: []Option{
			WithMiddleware(benchmarkNextMiddleware), WithGuards(guard), WithPipes(pipe), WithFilters(filter),
		}},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			route := metadata.Route{
				Method: "GET", Path: "/{id}", Handler: "Lookup",
				Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}},
			}
			if tc.static {
				route.StaticHTTP = func(receiver any, ctx httpadapter.Context) (any, int, error) {
					id, status, err := httpadapter.BindParam(ctx, "id", 0)
					if err != nil { return nil, status, err }
					return receiver.(*pipelineBenchmarkController).Lookup(id), 0, nil
				}
			}
			pkg := metadata.Package{Controllers: []metadata.Controller{{
				Type: reflect.TypeOf((*pipelineBenchmarkController)(nil)),
				Prefix: "/bench", Routes: []metadata.Route{route},
			}}}
			adapter := &capturedAdapter{}
			opts := []Option{WithAdapter(adapter), WithPackages(pkg)}
			opts = append(opts, tc.options...)
			if _, err := New(opts...); err != nil { b.Fatal(err) }
			handler := adapter.routes["GET /bench/{id}"]
			if handler == nil { b.Fatal("route was not registered") }
			ctx := &capturedContext{params: map[string]string{"id": "123"}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx.body = ""
				if err := handler(ctx); err != nil { b.Fatal(err) }
			}
			b.StopTimer()
			if ctx.status != 200 || ctx.body != "\"123\"\n" {
				b.Fatalf("unexpected response %d %q", ctx.status, ctx.body)
			}
		})
	}
}
