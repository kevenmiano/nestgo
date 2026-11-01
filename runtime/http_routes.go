package runtime

import (
	"encoding/json"
	"fmt"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/httpresponse"
	"github.com/kevenmiano/nestgo/metadata"
	"reflect"
	"sort"
	"strings"
)

// registerRoutes resolves injectable fields once during bootstrap and registers
// engine-independent handlers. Generated static invokers can replace reflect.Call.
func registerRoutes(adapter httpadapter.Adapter, packages []metadata.Package) error {
	return registerRoutesWithLifecycle(adapter, packages, nil)
}

func registerRoutesWithLifecycle(adapter httpadapter.Adapter, packages []metadata.Package, lifecycle *lifecycleRegistry) error {
	providers := map[reflect.Type]reflect.Value{}
	controllers := []metadata.Controller{}
	for _, pkg := range packages {
		for _, p := range pkg.Providers {
			if p.Type == nil || p.Type.Kind() != reflect.Ptr || p.Type.Elem().Kind() != reflect.Struct {
				return fmt.Errorf("invalid provider %v", p.Type)
			}
			if _, ok := providers[p.Type]; ok {
				return fmt.Errorf("duplicate provider %v", p.Type)
			}
			providers[p.Type] = reflect.New(p.Type.Elem())
		}
		controllers = append(controllers, pkg.Controllers...)
	}
	visibility, err := moduleVisibility(packages)
	if err != nil {
		return err
	}
	orderedProviders := make([]reflect.Type, 0, len(providers))
	for providerType := range providers {
		orderedProviders = append(orderedProviders, providerType)
	}
	sort.Slice(orderedProviders, func(i, j int) bool {
		return orderedProviders[i].String() < orderedProviders[j].String()
	})
	inject := func(v reflect.Value) error {
		return injectFields(v, providers, orderedProviders, visibility)
	}
	for _, providerType := range orderedProviders {
		if err := inject(providers[providerType]); err != nil {
			return err
		}
	}
	if lifecycle != nil {
		for _, typ := range orderedProviders {
			lifecycle.instances = append(lifecycle.instances, providers[typ].Interface())
		}
	}
	seen := map[string]bool{}
	for _, controller := range controllers {
		if controller.Type == nil || controller.Type.Kind() != reflect.Ptr || controller.Type.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("invalid controller %v", controller.Type)
		}
		instance := reflect.New(controller.Type.Elem())
		if err := inject(instance); err != nil {
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
			if typ.NumOut() > 2 || (typ.NumOut() == 2 && !typ.Out(1).Implements(errType)) {
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
			explicit, passthrough := false, false
			for i := 0; i < typ.NumIn(); i++ {
				if p, ok := bindings[i]; ok {
					if p.Kind == "Res" || p.Kind == "Response" {
						if explicit { return fmt.Errorf("multiple @Res arguments for %s", route.Handler) }
						if !reflect.TypeOf((*httpresponse.Writer)(nil)).AssignableTo(typ.In(i)) {
							return fmt.Errorf("@Res argument must accept *httpresponse.Writer in %s", route.Handler)
						}
						explicit, passthrough = true, p.Name == "passthrough"
					}
				} else if typ.In(i).Kind() != reflect.String {
					return fmt.Errorf("missing parameter decorator for %s", route.Handler)
				}
			}
			handler := method
			rt := route
			params := bindings
			methodType := typ
			err := adapter.Register(httpadapter.Route{Method: rt.Method, Path: path, Handler: func(ctx httpadapter.Context) error {
				writer := httpresponse.NewWriter(ctx)
				routeResponseHeaders(ctx, rt)
				var result any
				if rt.StaticInvoke != nil && methodType.NumIn() == 0 {
					value, err := rt.StaticInvoke(instance.Interface())
					if err != nil {
						return httpresponse.WriteError(ctx, err)
					}
					result = value
					if handlerError, ok := value.(error); ok && handlerError != nil {
						return httpresponse.WriteError(ctx, handlerError)
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
								if t.Kind() != reflect.String { return respondError(ctx, 500, "header parameter must be string") }
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
						return httpresponse.WriteError(ctx, results[1].Interface().(error))
					}
					if len(results) == 1 && results[0].Type().Implements(errType) {
						if !results[0].IsNil() { return httpresponse.WriteError(ctx, results[0].Interface().(error)) }
					} else if len(results) > 0 { result = results[0].Interface() }
				}
				return finishHTTPResponse(ctx, rt, result, writer, explicit, passthrough)
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
