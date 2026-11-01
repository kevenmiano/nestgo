package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kevenmiano/nestgo/http"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/validation"
	"reflect"
	"strings"
)

// registerRoutes resolves injectable fields once during bootstrap and registers
// engine-independent handlers. Generated static invokers can replace reflect.Call.
func registerRoutes(adapter httpadapter.Adapter, packages []metadata.Package) error {
	return registerRoutesWithLifecycle(adapter, packages, nil)
}

func registerRoutesWithLifecycle(adapter httpadapter.Adapter, packages []metadata.Package, lifecycle *lifecycleRegistry, direct ...httpadapter.Route) error {
	return registerRoutesWithPipeline(adapter, packages, lifecycle, requestPipelineConfig{}, nil, direct...)
}

// registerRoutesWithLifecycleAndInstances supplies external SQL and configuration providers.
func registerRoutesWithLifecycleAndInstances(adapter httpadapter.Adapter, packages []metadata.Package, lifecycle *lifecycleRegistry, instances []any, direct ...httpadapter.Route) error {
	return registerRoutesWithPipeline(adapter, packages, lifecycle, requestPipelineConfig{}, instances, direct...)
}
func registerRoutesWithPipeline(adapter httpadapter.Adapter, packages []metadata.Package, lifecycle *lifecycleRegistry, pipeline requestPipelineConfig, instances []any, direct ...httpadapter.Route) error {
	if err := validatePipeline(pipeline); err != nil {
		return err
	}
	var onProvider func(reflect.Type)
	if lifecycle != nil {
		onProvider = lifecycle.onProvider
	}
	graph, err := prepareBootstrapWithInstances(packages, instances, onProvider, nil)
	if err != nil {
		// A factory can have acquired resources before a later factory, DI
		// or module validation fails. The outer bootstrap rollback owns them.
		if lifecycle != nil {
			lifecycle.instances = append(lifecycle.instances, graph.allocated...)
		}
		return err
	}
	if lifecycle != nil {
		for _, typ := range graph.orderedProviders {
			lifecycle.instances = append(lifecycle.instances, graph.providers[typ].Interface())
		}
	}
	seen := map[string]bool{}
	for _, controller := range graph.controllers {
		instance, err := graph.newController(controller.Type)
		if err != nil {
			return err
		}
		if lifecycle != nil {
			lifecycle.instances = append(lifecycle.instances, instance.Interface())
			if lifecycle.onController != nil {
				lifecycle.onController(controller.Type, controller.Prefix)
			}
		}
		controllerGuards, err := resolveGuards(&graph, controller.Type, controller.Guards)
		if err != nil {
			return err
		}
		controllerPipes, err := resolvePipes(&graph, controller.Type, controller.Pipes)
		if err != nil {
			return err
		}
		controllerFilters, err := resolveFilters(&graph, controller.Type, controller.Filters)
		if err != nil {
			return err
		}
		for _, route := range controller.Routes {
			method := instance.MethodByName(route.Handler)
			if !method.IsValid() {
				return fmt.Errorf("handler %s not found", route.Handler)
			}
			typ := method.Type()
			contract, contractErr := compileRouteContract(controller.Prefix, route, typ)
			if contractErr != nil {
				return contractErr
			}
			path := contract.Path
			key := contract.Key
			if seen[key] {
				return fmt.Errorf("duplicate route %s", key)
			}
			seen[key] = true
			localGuards, err := resolveGuards(&graph, controller.Type, route.Guards)
			if err != nil {
				return err
			}
			localPipes, err := resolvePipes(&graph, controller.Type, route.Pipes)
			if err != nil {
				return err
			}
			localFilters, err := resolveFilters(&graph, controller.Type, route.Filters)
			if err != nil {
				return err
			}
			guards := make([]httpadapter.Guard, 0, len(pipeline.guards)+len(controllerGuards)+len(localGuards))
			guards = append(guards, pipeline.guards...)
			guards = append(guards, controllerGuards...)
			guards = append(guards, localGuards...)
			pipes := make([]httpadapter.Pipe, 0, len(pipeline.pipes)+len(controllerPipes)+len(localPipes))
			pipes = append(pipes, pipeline.pipes...)
			pipes = append(pipes, controllerPipes...)
			pipes = append(pipes, localPipes...)
			filters := make([]httpadapter.ExceptionFilter, 0, len(localFilters)+len(controllerFilters)+len(pipeline.filters))
			filters = append(filters, localFilters...)
			filters = append(filters, controllerFilters...)
			filters = append(filters, pipeline.filters...)
			middlewares := append([]httpadapter.Middleware(nil), pipeline.middleware...)
			middlewares = append(middlewares, pipeline.routeMiddleware[key]...)
			chain := compileMiddlewareChain(middlewares)
			controllerInstance := instance.Interface()
			bindings := contract.Bindings
			handler := method
			rt := route
			typedString := compileStringRouteInvoker(method, rt, bindings)
			// The optimized string invoker reads raw arguments; use pipe-aware binding if needed.
			if len(pipes) != 0 {
				typedString = nil
			}
			params := bindings
			methodType := typ
			explicit, passthrough := false, false
			for _, p := range bindings {
				if p.Kind == "Res" || p.Kind == "Response" {
					explicit, passthrough = true, p.Name == "passthrough"
				}
			}
			registrationErr := adapter.Register(httpadapter.Route{Method: rt.Method, Path: path, Handler: func(ctx httpadapter.Context) error {
				exec := httpadapter.ExecutionContext{Request: ctx, Controller: controllerInstance, Handler: rt.Handler}
				routeExecution := func() error {
					for _, guard := range guards {
						allowed, err := guard.CanActivate(exec)
						if err != nil {
							return err
						}
						if !allowed {
							return httpadapter.NewHTTPException(403, "Forbidden")
						}
					}
					bound := ctx
					if len(pipes) != 0 {
						bound = &pipeContext{Context: ctx, pipes: pipes}
					}
					writer := http.NewWriter(ctx)
					routeResponseHeaders(ctx, rt)
					var result any
					var resultStatus int
					if rt.StaticHTTP != nil {
						value, status, err := rt.StaticHTTP(controllerInstance, bound)
						if err != nil {
							var validationErrors validation.Errors
							if errors.As(err, &validationErrors) {
								return err
							}
							if status >= 400 && status <= 599 {
								return httpadapter.NewHTTPException(status, err.Error())
							}
							// Never accept success or redirect statuses for an exception.
							return err
						}
						result, resultStatus = value, status
						if handlerError, ok := value.(error); ok && handlerError != nil {
							return handlerError
						}
					} else if rt.StaticInvoke != nil && methodType.NumIn() == 0 {
						value, err := rt.StaticInvoke(controllerInstance)
						if err != nil {
							return err
						}
						result = value
						if handlerError, ok := value.(error); ok && handlerError != nil {
							return handlerError
						}
					} else if typedString != nil {
						value, err := typedString(bound)
						if err != nil {
							return err
						}
						result = value
					} else {
						args := make([]reflect.Value, methodType.NumIn())
						for i := 0; i < methodType.NumIn(); i++ {
							t := methodType.In(i)
							v := reflect.New(t)
							if p, ok := params[i]; ok {
								switch p.Kind {
								case "Body":
									body := bound.Body()
									if len(body) > 1<<20 {
										return httpadapter.NewHTTPException(413, "request body too large")
									}
									if err := json.Unmarshal(body, v.Interface()); err != nil {
										return httpadapter.NewHTTPException(400, "invalid JSON: "+err.Error())
									}
								case "Param":
									if t.Kind() != reflect.String {
										return httpadapter.NewHTTPException(500, "path parameter must be string")
									}
									v.Elem().SetString(bound.Param(p.Name))
								case "Query":
									if t.Kind() != reflect.String {
										return httpadapter.NewHTTPException(500, "query parameter must be string")
									}
									v.Elem().SetString(bound.Query(p.Name))
								case "Headers":
									if t.Kind() != reflect.String {
										return httpadapter.NewHTTPException(500, "header parameter must be string")
									}
									v.Elem().SetString(bound.Header(p.Name))
								case "Res", "Response":
									v.Elem().Set(reflect.ValueOf(writer))
								default:
									return httpadapter.NewHTTPException(500, "unsupported parameter decorator")
								}
								if p.Kind != "Res" && p.Kind != "Response" {
									transformed, err := httpadapter.ApplyPipes(bound, v.Elem().Interface(), httpadapter.ArgumentMetadata{Index: i, Kind: p.Kind, Name: p.Name, Type: t})
									if err != nil {
										return err
									}
									value := reflect.ValueOf(transformed)
									if !value.IsValid() || !value.Type().AssignableTo(t) {
										return httpadapter.NewHTTPException(400, "pipe returned an incompatible parameter type")
									}
									v.Elem().Set(value)
									if p.Kind == "Body" {
										if err := validation.Check(v.Elem().Interface()); err != nil {
											return err
										}
									}
								}
							} else {
								for _, part := range strings.Split(rt.Path, "/") {
									if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
										v.Elem().SetString(bound.Param(strings.Trim(part, "{}")))
										break
									}
								}
							}
							args[i] = v.Elem()
						}
						results := handler.Call(args)
						if len(results) == 2 && !results[1].IsNil() {
							return results[1].Interface().(error)
						}
						errType := reflect.TypeOf((*error)(nil)).Elem()
						if len(results) == 1 && results[0].Type().Implements(errType) {
							if !results[0].IsNil() {
								return results[0].Interface().(error)
							}
						} else if len(results) > 0 {
							result = results[0].Interface()
						}
					}
					replyRoute := rt
					if resultStatus != 0 {
						replyRoute.Status = resultStatus
					}
					return finishHTTPResponse(ctx, replyRoute, result, writer, explicit, passthrough)
				}
				managed := func() error { return handleHTTPException(routeExecution(), exec, filters) }
				return handleHTTPException(chain(ctx, managed), exec, pipeline.filters)
			}})
			if registrationErr != nil {
				return fmt.Errorf("register %s: %w", key, registrationErr)
			}
			if lifecycle != nil && lifecycle.onRoute != nil {
				lifecycle.onRoute(rt.Method, path)
			}
		}
	}
	for _, route := range direct {
		if route.Handler == nil || route.Method == "" || !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("invalid direct route %s %s: method, absolute path and handler are required", route.Method, route.Path)
		}
		key := route.Method + " " + route.Path
		if seen[key] {
			return fmt.Errorf("duplicate route %s", key)
		}
		seen[key] = true
		middlewares := append([]httpadapter.Middleware(nil), pipeline.middleware...)
		middlewares = append(middlewares, pipeline.routeMiddleware[key]...)
		chain := compileMiddlewareChain(middlewares)
		handler := route.Handler
		wrapped := httpadapter.Route{Method: route.Method, Path: route.Path, Handler: func(ctx httpadapter.Context) error {
			exec := httpadapter.ExecutionContext{Request: ctx, Handler: key}
			managed := func() error {
				for _, guard := range pipeline.guards {
					ok, err := guard.CanActivate(exec)
					if err != nil {
						return handleHTTPException(err, exec, pipeline.filters)
					}
					if !ok {
						return handleHTTPException(httpadapter.NewHTTPException(403, "Forbidden"), exec, pipeline.filters)
					}
				}
				bound := ctx
				if len(pipeline.pipes) != 0 {
					bound = &pipeContext{Context: ctx, pipes: pipeline.pipes}
				}
				return handleHTTPException(handler(bound), exec, pipeline.filters)
			}
			return handleHTTPException(chain(ctx, managed), exec, pipeline.filters)
		}}
		if err := adapter.Register(wrapped); err != nil {
			return fmt.Errorf("register %s: %w", key, err)
		}
		if lifecycle != nil && lifecycle.onRoute != nil {
			lifecycle.onRoute(route.Method, route.Path)
		}
	}
	return nil
}
func respondError(ctx httpadapter.Context, status int, message string) error {
	if status >= 500 {
		message = "Internal server error"
	}
	ctx.Status(status)
	ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
	return ctx.Write([]byte(message + "\n"))
}

func respondValidationError(ctx httpadapter.Context, err error) error {
	var fields validation.Errors
	if !errors.As(err, &fields) {
		return respondError(ctx, 400, err.Error())
	}
	ctx.Status(400)
	ctx.SetHeader("Content-Type", "application/json")
	payload := struct {
		StatusCode int               `json:"statusCode"`
		Error      string            `json:"error"`
		Fields     validation.Errors `json:"fields"`
	}{400, "Bad Request", fields}
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return respondError(ctx, 500, "Internal server error")
	}
	return ctx.Write(append(data, '\n'))
}
