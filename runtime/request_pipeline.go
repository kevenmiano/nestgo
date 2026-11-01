package runtime

import (
	"errors"
	"fmt"
	"github.com/kevenmiano/nestgo/http"
	"github.com/kevenmiano/nestgo/validation"
	"reflect"

	"github.com/kevenmiano/nestgo/httpadapter"
)

// requestPipelineConfig is immutable after bootstrap. Shared instances must be
// concurrency-safe, just like singleton providers and controllers.
type requestPipelineConfig struct {
	middleware      []httpadapter.Middleware
	routeMiddleware map[string][]httpadapter.Middleware
	guards          []httpadapter.Guard
	pipes           []httpadapter.Pipe
	filters         []httpadapter.ExceptionFilter
}

type pipeContext struct {
	httpadapter.Context
	pipes []httpadapter.Pipe
}

func (c *pipeContext) ApplyPipes(value any, meta httpadapter.ArgumentMetadata) (any, error) {
	var err error
	for _, p := range c.pipes {
		value, err = p.Transform(value, meta)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func validatePipeline(c requestPipelineConfig) error {
	for i, mw := range c.middleware {
		if mw == nil {
			return fmt.Errorf("nil global middleware at index %d", i)
		}
	}
	for key, mws := range c.routeMiddleware {
		for i, mw := range mws {
			if mw == nil {
				return fmt.Errorf("nil middleware for %s at index %d", key, i)
			}
		}
	}
	for i, g := range c.guards {
		if g == nil || reflect.ValueOf(g).Kind() == reflect.Ptr && reflect.ValueOf(g).IsNil() {
			return fmt.Errorf("nil global guard at index %d", i)
		}
	}
	for i, p := range c.pipes {
		if p == nil || reflect.ValueOf(p).Kind() == reflect.Ptr && reflect.ValueOf(p).IsNil() {
			return fmt.Errorf("nil global pipe at index %d", i)
		}
	}
	for i, f := range c.filters {
		if f == nil || reflect.ValueOf(f).Kind() == reflect.Ptr && reflect.ValueOf(f).IsNil() {
			return fmt.Errorf("nil global filter at index %d", i)
		}
	}
	return nil
}

// compileRequestComponents resolves class references once at bootstrap. Hot
// requests never use reflection for guard/pipe/filter construction.
func compileRequestComponents(graph *bootstrapGraph, owner reflect.Type, refs []reflect.Type, kind string) ([]any, error) {
	result := make([]any, 0, len(refs))
	for _, typ := range refs {
		if typ == nil {
			return nil, fmt.Errorf("nil %s for %s", kind, owner)
		}
		value, ok := graph.providers[typ]
		if !ok {
			return nil, fmt.Errorf("%s %s must be a registered provider", kind, typ)
		}
		if visible, scoped := graph.visibility[owner]; scoped && !visible[typ] {
			return nil, fmt.Errorf("%s %s not visible to controller %s", kind, typ, owner)
		}
		result = append(result, value.Interface())
	}
	return result, nil
}

func resolveGuards(graph *bootstrapGraph, owner reflect.Type, refs []reflect.Type) ([]httpadapter.Guard, error) {
	instances, err := compileRequestComponents(graph, owner, refs, "guard")
	if err != nil {
		return nil, err
	}
	guards := make([]httpadapter.Guard, 0, len(instances))
	for _, instance := range instances {
		g, ok := instance.(httpadapter.Guard)
		if !ok {
			return nil, fmt.Errorf("%T does not implement httpadapter.Guard", instance)
		}
		guards = append(guards, g)
	}
	return guards, nil
}
func resolvePipes(graph *bootstrapGraph, owner reflect.Type, refs []reflect.Type) ([]httpadapter.Pipe, error) {
	instances, err := compileRequestComponents(graph, owner, refs, "pipe")
	if err != nil {
		return nil, err
	}
	pipes := make([]httpadapter.Pipe, 0, len(instances))
	for _, instance := range instances {
		p, ok := instance.(httpadapter.Pipe)
		if !ok {
			return nil, fmt.Errorf("%T does not implement httpadapter.Pipe", instance)
		}
		pipes = append(pipes, p)
	}
	return pipes, nil
}
func resolveFilters(graph *bootstrapGraph, owner reflect.Type, refs []reflect.Type) ([]httpadapter.ExceptionFilter, error) {
	instances, err := compileRequestComponents(graph, owner, refs, "filter")
	if err != nil {
		return nil, err
	}
	filters := make([]httpadapter.ExceptionFilter, 0, len(instances))
	for _, instance := range instances {
		f, ok := instance.(httpadapter.ExceptionFilter)
		if !ok {
			return nil, fmt.Errorf("%T does not implement httpadapter.ExceptionFilter", instance)
		}
		filters = append(filters, f)
	}
	return filters, nil
}

func handleHTTPException(err error, exec httpadapter.ExecutionContext, filters []httpadapter.ExceptionFilter) error {
	if err == nil {
		return nil
	}
	for _, filter := range filters {
		handled, writeErr := filter.Catch(err, exec)
		if writeErr != nil {
			return writeErr
		}
		if handled {
			return nil
		}
	}
	var validationErrors validation.Errors
	if errors.As(err, &validationErrors) { return respondValidationError(exec.Request, err) }
	var pipelineError *httpadapter.HTTPException
	if errors.As(err, &pipelineError) && pipelineError != nil {
		message := pipelineError.Message
		if pipelineError.Status >= 500 {
			message = "Internal server error"
		}
		err = http.NewHTTPError(pipelineError.Status, message)
	}
	return http.WriteError(exec.Request, err)
}

// middlewareChain is compiled once per route at bootstrap. Keeping the
// downstream function outside the request path avoids rebuilding the recursive
// dispatcher and its closure on every request.
//
// Each middleware still gets its own request-local Next closure: sharing or
// pooling Next would break the exactly-once rule and could leak state between
// concurrent requests.
type middlewareChain func(httpadapter.Context, httpadapter.Next) error

func compileMiddlewareChain(middlewares []httpadapter.Middleware) middlewareChain {
	if len(middlewares) == 0 {
		return func(_ httpadapter.Context, next httpadapter.Next) error { return next() }
	}

	var chain middlewareChain = func(_ httpadapter.Context, next httpadapter.Next) error {
		return next()
	}
	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]
		downstream := chain
		chain = func(ctx httpadapter.Context, terminal httpadapter.Next) error {
			advanced := false
			return middleware(ctx, func() error {
				if advanced {
					return fmt.Errorf("middleware called next more than once")
				}
				advanced = true
				return downstream(ctx, terminal)
			})
		}
	}
	return chain
}
