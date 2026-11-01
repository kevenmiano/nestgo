package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

type pipelineTrace struct{ steps []string }

func (t *pipelineTrace) add(s string) { t.steps = append(t.steps, s) }

type globalTestGuard struct{ trace *pipelineTrace }

func (g *globalTestGuard) CanActivate(httpadapter.ExecutionContext) (bool, error) {
	g.trace.add("global guard")
	return true, nil
}

type controllerTestGuard struct{ Trace *pipelineTrace `nestgo:"inject"` }

func (g *controllerTestGuard) CanActivate(httpadapter.ExecutionContext) (bool, error) {
	g.Trace.add("controller guard")
	return true, nil
}

type routeTestGuard struct{ Trace *pipelineTrace `nestgo:"inject"` }

func (g *routeTestGuard) CanActivate(c httpadapter.ExecutionContext) (bool, error) {
	g.Trace.add("route guard")
	return c.Request.Header("Authorization") != "deny", nil
}

type globalTestPipe struct{ trace *pipelineTrace }

func (p *globalTestPipe) Transform(v any, _ httpadapter.ArgumentMetadata) (any, error) {
	p.trace.add("global pipe")
	return v, nil
}

type controllerTestPipe struct{ Trace *pipelineTrace `nestgo:"inject"` }

func (p *controllerTestPipe) Transform(v any, _ httpadapter.ArgumentMetadata) (any, error) {
	p.Trace.add("controller pipe")
	return v, nil
}

type routeTestPipe struct{ Trace *pipelineTrace `nestgo:"inject"` }

func (p *routeTestPipe) Transform(v any, m httpadapter.ArgumentMetadata) (any, error) {
	p.Trace.add("route pipe")
	if m.Kind == "Param" {
		return strings.ToUpper(v.(string)), nil
	}
	return v, nil
}

type routeTestFilter struct{ Trace *pipelineTrace `nestgo:"inject"` }

func (f *routeTestFilter) Catch(err error, c httpadapter.ExecutionContext) (bool, error) {
	f.Trace.add("route filter")
	c.Request.Status(418)
	return true, c.Request.Write([]byte("route caught: " + err.Error()))
}

type globalTestFilter struct{ trace *pipelineTrace }

func (f *globalTestFilter) Catch(err error, c httpadapter.ExecutionContext) (bool, error) {
	f.trace.add("global filter")
	c.Request.Status(422)
	return true, c.Request.Write([]byte("global caught: " + err.Error()))
}

type pipelineTestController struct{ Trace *pipelineTrace `nestgo:"inject"` }

func (c *pipelineTestController) Echo(id string) string { c.Trace.add("handler"); return id }
func (*pipelineTestController) Fail() (string, error) {
	return "", errors.New("failure")
}

type pipelineInputContext struct {
	*capturedContext
	auth string
}

func (c *pipelineInputContext) Header(s string) string {
	if s == "Authorization" {
		return c.auth
	}
	return ""
}

func pipelineTestPackage() metadata.Package {
	return metadata.Package{
		Providers: []metadata.Provider{
			{Type: reflect.TypeOf((*pipelineTrace)(nil))},
			{Type: reflect.TypeOf((*controllerTestGuard)(nil))},
			{Type: reflect.TypeOf((*routeTestGuard)(nil))},
			{Type: reflect.TypeOf((*controllerTestPipe)(nil))},
			{Type: reflect.TypeOf((*routeTestPipe)(nil))},
			{Type: reflect.TypeOf((*routeTestFilter)(nil))},
		},
		Controllers: []metadata.Controller{{
			Type:   reflect.TypeOf((*pipelineTestController)(nil)),
			Prefix: "/api",
			Guards: []reflect.Type{reflect.TypeOf((*controllerTestGuard)(nil))},
			Pipes:  []reflect.Type{reflect.TypeOf((*controllerTestPipe)(nil))},
			Routes: []metadata.Route{
				{
					Method: "GET", Path: "/{id}", Handler: "Echo",
					Guards:     []reflect.Type{reflect.TypeOf((*routeTestGuard)(nil))},
					Pipes:      []reflect.Type{reflect.TypeOf((*routeTestPipe)(nil))},
					Filters:    []reflect.Type{reflect.TypeOf((*routeTestFilter)(nil))},
					Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "id"}},
					StaticHTTP: func(receiver any, ctx httpadapter.Context) (any, int, error) {
						id, code, err := httpadapter.BindParam(ctx, "id", 0)
						if err != nil {
							return nil, code, err
						}
						return receiver.(*pipelineTestController).Echo(id), 0, nil
					},
				},
				{Method: "GET", Path: "/failure", Handler: "Fail", Filters: []reflect.Type{reflect.TypeOf((*routeTestFilter)(nil))}},
			},
		}},
	}
}

func TestRequestPipelineScopesStaticDispatchAndDI(t *testing.T) {
	trace := &pipelineTrace{}
	a := &capturedAdapter{}
	app, err := New(
		WithAdapter(a), WithPackages(pipelineTestPackage()),
		WithMiddleware(func(_ httpadapter.Context, next httpadapter.Next) error {
			trace.add("global middleware before")
			err := next()
			trace.add("global middleware after")
			return err
		}),
		WithRouteMiddleware("GET", "/api/{id}", func(_ httpadapter.Context, next httpadapter.Next) error {
			trace.add("route middleware")
			return next()
		}),
		WithGuards(&globalTestGuard{trace: trace}),
		WithPipes(&globalTestPipe{trace: trace}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	// The trace used by the injected classes is the provider singleton. Verify
	// it independently from the directly supplied global pipeline instances.
	ctx := &pipelineInputContext{capturedContext: &capturedContext{params: map[string]string{"id": "ada"}}}
	if err := a.routes["GET /api/{id}"](ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.status != 200 || ctx.body != "\"ADA\"\n" {
		t.Fatalf("status=%d body=%q", ctx.status, ctx.body)
	}
	if strings.Join(trace.steps, ",") != "global middleware before,route middleware,global guard,global pipe,global middleware after" {
		t.Fatalf("global pipeline ordering: %v", trace.steps)
	}
}

func TestRequestPipelineGuardDenialAndExceptionFilter(t *testing.T) {
	a := &capturedAdapter{}
	app, err := New(WithAdapter(a), WithPackages(pipelineTestPackage()))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	deny := &pipelineInputContext{capturedContext: &capturedContext{params: map[string]string{"id": "secret"}}, auth: "deny"}
	if err := a.routes["GET /api/{id}"](deny); err != nil {
		t.Fatal(err)
	}
	if deny.status != 418 || deny.body != "route caught: Forbidden" {
		t.Fatalf("route filter: status=%d body=%q", deny.status, deny.body)
	}
	failure := &capturedContext{}
	if err := a.routes["GET /api/failure"](failure); err != nil {
		t.Fatal(err)
	}
	if failure.status != 418 || failure.body != "route caught: failure" {
		t.Fatalf("handler filter: status=%d body=%q", failure.status, failure.body)
	}
}

func TestMiddlewareErrorsOnlyGoToGlobalFilters(t *testing.T) {
	a := &capturedAdapter{}
	trace := &pipelineTrace{}
	app, err := New(
		WithAdapter(a), WithPackages(pipelineTestPackage()),
		WithMiddleware(func(httpadapter.Context, httpadapter.Next) error { return errors.New("middleware failure") }),
		WithFilters(&globalTestFilter{trace: trace}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	ctx := &capturedContext{}
	if err := a.routes["GET /api/failure"](ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.status != 422 || ctx.body != "global caught: middleware failure" || strings.Join(trace.steps, ",") != "global filter" {
		t.Fatalf("middleware filter: status=%d body=%q trace=%v", ctx.status, ctx.body, trace.steps)
	}
}

func TestRequestPipelineRejectsUnregisteredOrWrongProvider(t *testing.T) {
	p := pipelineTestPackage()
	p.Providers = p.Providers[:1]
	_, err := New(WithAdapter(&capturedAdapter{}), WithPackages(p))
	if err == nil || !strings.Contains(err.Error(), "registered provider") {
		t.Fatalf("wanted missing guard: %v", err)
	}

	_, err = New(WithMiddleware(nil))
	if err == nil || !strings.Contains(err.Error(), "nil global middleware") {
		t.Fatalf("wanted middleware validation: %v", err)
	}
}

func TestPipeFailureMapsToBadRequest(t *testing.T) {
	a := &capturedAdapter{}
	app, err := New(WithAdapter(a), WithPackages(pipelineTestPackage()),
		WithPipes(&failingPipe{}))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	ctx := &capturedContext{params: map[string]string{"id": "ada"}}
	if err := a.routes["GET /api/{id}"](ctx); err != nil {
		t.Fatal(err)
	}
	// The class-based route filter handles the pipe's error.
	if ctx.status != 418 || !strings.Contains(ctx.body, "invalid argument") {
		t.Fatalf("pipe error: %d %q", ctx.status, ctx.body)
	}
}

type failingPipe struct{}

func (*failingPipe) Transform(any, httpadapter.ArgumentMetadata) (any, error) {
	return nil, httpadapter.NewHTTPException(400, "invalid argument")
}
