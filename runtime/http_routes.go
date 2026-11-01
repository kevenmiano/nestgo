package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

// registerRoutes resolves injectable fields once during bootstrap and registers
// engine-independent handlers. Generated static invokers bypass reflect.Call.
func registerRoutes(adapter httpadapter.Adapter, packages []metadata.Package) error {
	return registerRoutesWithLifecycle(adapter, packages, nil)
}

func registerRoutesWithLifecycle(adapter httpadapter.Adapter, packages []metadata.Package, lifecycle *lifecycleRegistry, direct ...httpadapter.Route) error {
	return registerRoutesWithPipeline(adapter, packages, lifecycle, requestPipelineConfig{}, direct...)
}

func registerRoutesWithPipeline(adapter httpadapter.Adapter, packages []metadata.Package, lifecycle *lifecycleRegistry, pipeline requestPipelineConfig, direct ...httpadapter.Route) error {
	if err := validatePipeline(pipeline); err != nil {
		return err
	}
	var onProvider func(reflect.Type)
	if lifecycle != nil {
		onProvider = lifecycle.onProvider
	}
	graph, err := prepareBootstrap(packages, onProvider, nil)
	if err != nil {
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

			bindings := contract.Bindings
			handler := method
			rt := route
			methodType := typ
			controllerInstance := instance.Interface()
			registered := httpadapter.Route{Method: rt.Method, Path: contract.Path, Handler: func(ctx httpadapter.Context) error {
				exec := httpadapter.ExecutionContext{Request: ctx, Controller: controllerInstance, Handler: rt.Handler}
				// Route filters see failures from guards, pipes and the controller.
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
					bindingContext := ctx
					if len(pipes) != 0 {
						bindingContext = &pipeContext{Context: ctx, pipes: pipes}
					}
					var result any
					if rt.StaticHTTP != nil {
						value, status, err := rt.StaticHTTP(controllerInstance, bindingContext)
						if err != nil {
							if status >= 400 && status <= 599 && httpadapter.StatusOf(err) == 500 {
								if status >= 500 {
									return httpadapter.NewHTTPException(status, "Internal server error")
								}
								v := reflect.ValueOf(err)
								switch v.Kind() {
								case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
									if v.IsNil() {
										return httpadapter.NewHTTPException(status, "Bad Request")
									}
								}
								return httpadapter.NewHTTPException(status, err.Error())
							}
							return err
						}
						result = value
					} else if rt.StaticInvoke != nil && methodType.NumIn() == 0 {
						value, err := rt.StaticInvoke(controllerInstance)
						if err != nil {
							return err
						}
						result = value
					} else {
						args := make([]reflect.Value, methodType.NumIn())
						for i := 0; i < methodType.NumIn(); i++ {
							t := methodType.In(i)
							v := reflect.New(t)
							if p, ok := bindings[i]; ok {
								switch p.Kind {
								case "Body":
									body := bindingContext.Body()
									if len(body) > 1<<20 {
										return httpadapter.NewHTTPException(413, "request body too large")
									}
									if err := json.Unmarshal(body, v.Interface()); err != nil {
										return httpadapter.NewHTTPException(400, "invalid JSON: "+err.Error())
									}
								case "Param":
									v.Elem().SetString(bindingContext.Param(p.Name))
								case "Query":
									v.Elem().SetString(bindingContext.Query(p.Name))
								default:
									return fmt.Errorf("unsupported parameter decorator")
								}
								transformed, err := httpadapter.ApplyPipes(bindingContext, v.Elem().Interface(), httpadapter.ArgumentMetadata{Index: i, Kind: p.Kind, Name: p.Name, Type: t})
								if err != nil {
									return err
								}
								value := reflect.ValueOf(transformed)
								if !value.IsValid() || !value.Type().AssignableTo(t) {
									return httpadapter.NewHTTPException(400, "pipe returned an incompatible parameter type")
								}
								v.Elem().Set(value)
							} else {
								for _, part := range strings.Split(rt.Path, "/") {
									if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
										v.Elem().SetString(bindingContext.Param(strings.Trim(part, "{}")))
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
						result = results[0].Interface()
					}
					status := rt.Status
					if status == 0 {
						if rt.Method == "POST" {
							status = 201
						} else {
							status = 200
						}
					}
					payload, err := json.Marshal(result)
					if err != nil {
						return fmt.Errorf("failed to encode HTTP response: %w", err)
					}
					ctx.Status(status)
					ctx.SetHeader("Content-Type", "application/json")
					return ctx.Write(append(payload, '\n'))
				}
				managedExecution := func() error {
					return handleHTTPException(routeExecution(), exec, filters)
				}
				// A middleware failure is intentionally handled by global filters
				// only, as in NestJS.
				return handleHTTPException(runMiddleware(ctx, middlewares, managedExecution), exec, pipeline.filters)
			}}
			if err := adapter.Register(registered); err != nil {
				return fmt.Errorf("register %s: %w", key, err)
			}
			if lifecycle != nil && lifecycle.onRoute != nil {
				lifecycle.onRoute(rt.Method, contract.Path)
			}
		}
	}

	for _, route := range direct {
		if route.Handler == nil || route.Method == "" || !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("invalid direct route %s %s: method, absolute path and handler are required", route.Method, route.Path)
		}
		key := route.Method + " " + route.Path
		if seen[key] { return fmt.Errorf("duplicate route %s", key) }
		seen[key] = true
		middleware := append([]httpadapter.Middleware(nil), pipeline.middleware...)
		middleware = append(middleware, pipeline.routeMiddleware[key]...)
		handler := route.Handler
		wrapped := httpadapter.Route{Method: route.Method, Path: route.Path, Handler: func(ctx httpadapter.Context) error {
			exec := httpadapter.ExecutionContext{Request: ctx, Handler: key}
			managed := func() error {
				for _, guard := range pipeline.guards {
					ok, err := guard.CanActivate(exec)
					if err != nil { return handleHTTPException(err, exec, pipeline.filters) }
					if !ok { return handleHTTPException(httpadapter.NewHTTPException(403, "Forbidden"), exec, pipeline.filters) }
				}
				bound := ctx
				if len(pipeline.pipes) != 0 { bound = &pipeContext{Context: ctx, pipes: pipeline.pipes} }
				return handleHTTPException(handler(bound), exec, pipeline.filters)
			}
			return handleHTTPException(runMiddleware(ctx, middleware, managed), exec, pipeline.filters)
		}}
		if err := adapter.Register(wrapped); err != nil { return fmt.Errorf("register %s: %w", key, err) }
		if lifecycle != nil && lifecycle.onRoute != nil { lifecycle.onRoute(route.Method, route.Path) }
	}
	return nil
}

func respondError(ctx httpadapter.Context, status int, message string) error {
	return httpadapter.WriteError(ctx, httpadapter.NewHTTPException(status, message))
}
