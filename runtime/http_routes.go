package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
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
	graph, err := prepareBootstrap(packages, nil, nil)
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
		}
		for _, route := range controller.Routes {
			method := instance.MethodByName(route.Handler)
			if !method.IsValid() {
				return fmt.Errorf("handler %s not found", route.Handler)
			}
			typ := method.Type()
			errType := reflect.TypeOf((*error)(nil)).Elem()
			if typ.NumOut() < 1 || typ.NumOut() > 2 || (typ.NumOut() == 2 && !typ.Out(1).Implements(errType)) {
				return fmt.Errorf("invalid signature %s", route.Handler)
			}
			path := strings.TrimSuffix(controller.Prefix, "/") + "/" + strings.TrimPrefix(route.Path, "/")
			key := route.Method + " " + path
			if seen[key] {
				return fmt.Errorf("duplicate route %s", key)
			}
			seen[key] = true
			bindings := map[int]metadata.Parameter{}
			for _, p := range route.Parameters {
				if p.Index < 0 || p.Index >= typ.NumIn() {
					return fmt.Errorf("invalid parameter index %d", p.Index)
				}
				bindings[p.Index] = p
			}
			for i := 0; i < typ.NumIn(); i++ {
				if _, ok := bindings[i]; !ok && typ.In(i).Kind() != reflect.String {
					return fmt.Errorf("missing parameter decorator for %s", route.Handler)
				}
			}
			handler := method
			rt := route
			params := bindings
			methodType := typ
			err := adapter.Register(httpadapter.Route{Method: rt.Method, Path: path, Handler: func(ctx httpadapter.Context) error {
				var result any
				if rt.StaticInvoke != nil && methodType.NumIn() == 0 {
					value, err := rt.StaticInvoke(instance.Interface())
					if err != nil {
						return respondError(ctx, 500, err.Error())
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
						return respondError(ctx, 500, results[1].Interface().(error).Error())
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
				ctx.Status(status)
				ctx.SetHeader("Content-Type", "application/json")
				data, err := json.Marshal(result)
				if err != nil {
					return errors.New("failed to encode HTTP response: " + err.Error())
				}
				return ctx.Write(append(data, '\n'))
			}})
			if err != nil {
				return fmt.Errorf("register %s: %w", key, err)
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
