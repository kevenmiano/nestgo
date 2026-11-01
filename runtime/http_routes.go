package runtime

import (
	"encoding/json"
	"fmt"
	"github.com/kevenmiano/nestgo/http"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	"reflect"
	"strings"
)

// registerRoutes resolves injectable fields once during bootstrap and registers
// engine-independent handlers. Generated static invokers can replace reflect.Call.
func registerRoutes(adapter httpadapter.Adapter, packages []metadata.Package) error {
	return registerRoutesWithLifecycle(adapter, packages, nil)
}

func registerRoutesWithLifecycle(adapter httpadapter.Adapter, packages []metadata.Package, lifecycle *lifecycleRegistry) error {
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
			bindings := contract.Bindings
			handler := method
			rt := route
			params := bindings
			methodType := typ
			explicit, passthrough := false, false
			for _, p := range bindings {
				if p.Kind == "Res" || p.Kind == "Response" {
					explicit, passthrough = true, p.Name == "passthrough"
				}
			}
			registrationErr := adapter.Register(httpadapter.Route{Method: rt.Method, Path: path, Handler: func(ctx httpadapter.Context) error {
				writer := http.NewWriter(ctx)
				routeResponseHeaders(ctx, rt)
				var result any
				var resultStatus int
				if rt.StaticHTTP != nil {
					value, status, err := rt.StaticHTTP(instance.Interface(), ctx)
					if err != nil {
						if status != 0 {
							return respondError(ctx, status, err.Error())
						}
						return http.WriteError(ctx, err)
					}
					result, resultStatus = value, status
					if handlerError, ok := value.(error); ok && handlerError != nil {
						return http.WriteError(ctx, handlerError)
					}
				} else if rt.StaticInvoke != nil && methodType.NumIn() == 0 {
					value, err := rt.StaticInvoke(instance.Interface())
					if err != nil {
						return http.WriteError(ctx, err)
					}
					result = value
					if handlerError, ok := value.(error); ok && handlerError != nil {
						return http.WriteError(ctx, handlerError)
					}
				} else {
					args := make([]reflect.Value, methodType.NumIn())
					for i := 0; i < methodType.NumIn(); i++ {
						t := methodType.In(i)
						v := reflect.New(t)
						if p, ok := params[i]; ok {
							switch p.Kind {
							case "Body":
								body := ctx.Body()
								if len(body) > 1<<20 {
									return respondError(ctx, 413, "request body too large")
								}
								if err := json.Unmarshal(body, v.Interface()); err != nil {
									return respondError(ctx, 400, "invalid JSON: "+err.Error())
								}
							case "Param":
								if t.Kind() != reflect.String {
									return respondError(ctx, 500, "path parameter must be string")
								}
								v.Elem().SetString(ctx.Param(p.Name))
							case "Query":
								if t.Kind() != reflect.String {
									return respondError(ctx, 500, "query parameter must be string")
								}
								v.Elem().SetString(ctx.Query(p.Name))
							case "Headers":
								if t.Kind() != reflect.String {
									return respondError(ctx, 500, "header parameter must be string")
								}
								v.Elem().SetString(ctx.Header(p.Name))
							case "Res", "Response":
								v.Elem().Set(reflect.ValueOf(writer))
							default:
								return respondError(ctx, 500, "unsupported parameter decorator")
							}
						} else {
							for _, part := range strings.Split(rt.Path, "/") {
								if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
									v.Elem().SetString(ctx.Param(strings.Trim(part, "{}")))
									break
								}
							}
						}
						args[i] = v.Elem()
					}
					results := handler.Call(args)
					if len(results) == 2 && !results[1].IsNil() {
						return http.WriteError(ctx, results[1].Interface().(error))
					}
					errType := reflect.TypeOf((*error)(nil)).Elem()
					if len(results) == 1 && results[0].Type().Implements(errType) {
						if !results[0].IsNil() {
							return http.WriteError(ctx, results[0].Interface().(error))
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
			}})
			if registrationErr != nil {
				return fmt.Errorf("register %s: %w", key, registrationErr)
			}
			if lifecycle != nil && lifecycle.onRoute != nil {
				lifecycle.onRoute(rt.Method, path)
			}
		}
	}
	return nil
}
func respondError(ctx httpadapter.Context, status int, message string) error {
	ctx.Status(status)
	ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
	return ctx.Write([]byte(message + "\n"))
}
