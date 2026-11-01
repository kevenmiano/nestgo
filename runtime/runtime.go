package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kevenmiano/nestgo/logging"
	"github.com/kevenmiano/nestgo/httpresponse"
	"github.com/kevenmiano/nestgo/platform/nethttp"
	"github.com/kevenmiano/nestgo/metadata"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"time"
)

func New(packages ...metadata.Package) (http.Handler, error) {
	return NewWithOptions(Options{}, packages...)
}

// Options controls framework lifecycle diagnostics without affecting request handling.
type Options struct {
	Logger *logging.Service
	Debug  bool
}

func NewWithOptions(options Options, packages ...metadata.Package) (http.Handler, error) {
	logger := options.Logger
	if logger == nil {
		logger = logging.New(logging.Config{MinLevel: logging.LevelInfo})
	}
	ctx := context.Background()
	boot := time.Now()
	logger.Info(ctx, "NestFactory", "Starting NestGo application...")
	fail := func(err error) (http.Handler, error) {
		logger.Error(ctx, "NestFactory", "Bootstrap failed", slog.String("error", err.Error()))
		return nil, err
	}
	logDebug := func(component, message string, attrs ...slog.Attr) {
		if options.Debug {
			logger.Debug(ctx, component, message, attrs...)
		}
	}
	providers := map[reflect.Type]reflect.Value{}
	orderedProviders := make([]reflect.Type, 0)
	controllers := []metadata.Controller{}
	for _, pkg := range packages {
		moduleStart := time.Now()
		for _, p := range pkg.Providers {
			if p.Type == nil || p.Type.Kind() != reflect.Ptr || p.Type.Elem().Kind() != reflect.Struct {
				return fail(fmt.Errorf("provider must be pointer to struct: %v", p.Type))
			}
			if _, exists := providers[p.Type]; exists {
				return fail(fmt.Errorf("duplicate provider: %v", p.Type))
			}
			providers[p.Type] = reflect.New(p.Type.Elem())
			orderedProviders = append(orderedProviders, p.Type)
			logDebug("InstanceLoader", "Provider instantiated", slog.String("provider", p.Type.String()))
		}
		controllers = append(controllers, pkg.Controllers...)
		logger.Startup(ctx, "InstanceLoader", "Package providers registered", moduleStart)
	}
	visibility, err := moduleVisibility(packages)
	if err != nil {
		return fail(err)
	}
	inject := func(v reflect.Value) error {
		return injectFields(v, providers, orderedProviders, visibility)
	}
	diStart := time.Now()
	for _, providerType := range orderedProviders {
		v := providers[providerType]
		if err := inject(v); err != nil {
			return fail(err)
		}
	}
	logger.Startup(ctx, "InstanceLoader", "Provider dependencies resolved", diStart)
	routeStart := time.Now()
	mux := http.NewServeMux()
	seen := map[string]bool{}
	for _, c := range controllers {
		if c.Type == nil || c.Type.Kind() != reflect.Ptr || c.Type.Elem().Kind() != reflect.Struct {
			return fail(fmt.Errorf("invalid controller %v", c.Type))
		}
		instance := reflect.New(c.Type.Elem())
		if err := inject(instance); err != nil {
			return fail(err)
		}
		logger.Info(ctx, "RoutesResolver", "Controller registered", slog.String("controller", c.Type.String()), slog.String("prefix", c.Prefix))
		for _, route := range c.Routes {
			method := instance.MethodByName(route.Handler)
			if !method.IsValid() {
				return fail(fmt.Errorf("missing handler %s.%s", c.Type, route.Handler))
			}
			typ := method.Type()
			if typ.NumOut() > 2 || (typ.NumOut() == 2 && !typ.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem())) {
				return fail(fmt.Errorf("unsupported signature %s", route.Handler))
			}
			path := strings.TrimSuffix(c.Prefix, "/") + "/" + strings.TrimPrefix(route.Path, "/")
			if path == "" {
				path = "/"
			}
			key := route.Method + " " + path
			if seen[key] {
				return fail(fmt.Errorf("duplicate route %s", key))
			}
			seen[key] = true
			bindings := map[int]metadata.Parameter{}
			for _, p := range route.Parameters {
				if p.Index < 0 || p.Index >= typ.NumIn() {
					return fail(fmt.Errorf("invalid parameter index %d", p.Index))
				}
				bindings[p.Index] = p
			}
			explicit, passthrough := false, false
			for i := 0; i < typ.NumIn(); i++ {
				if p, ok := bindings[i]; ok {
					if p.Kind == "Res" || p.Kind == "Response" {
						if explicit { return fail(fmt.Errorf("multiple @Res arguments for %s",route.Handler)) }
						if !reflect.TypeOf((*httpresponse.Writer)(nil)).AssignableTo(typ.In(i)) {
							return fail(fmt.Errorf("@Res argument must accept *httpresponse.Writer in %s",route.Handler))
						}
						explicit, passthrough = true, p.Name == "passthrough"
					}
				} else if typ.In(i).Kind() != reflect.String {
					return fail(fmt.Errorf("missing decorator for argument %d of %s", i, route.Handler))
				}
			}
			handler := method
			spec := route
			mux.HandleFunc(key, func(w http.ResponseWriter, r *http.Request) {
				responseCtx := nethttp.NewContext(w,r)
				writer := httpresponse.NewWriter(responseCtx)
				routeResponseHeaders(responseCtx, spec)
				args := make([]reflect.Value, typ.NumIn())
				for i := 0; i < typ.NumIn(); i++ {
					t := typ.In(i)
					v := reflect.New(t)
					if p, ok := bindings[i]; ok {
						switch p.Kind {
						case "Body":
							if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v.Interface()); err != nil {
								http.Error(w, "invalid JSON: "+err.Error(), 400)
								return
							}
						case "Param":
							if t.Kind() != reflect.String {
								http.Error(w, "path parameter must be string", 500)
								return
							}
							v.Elem().SetString(r.PathValue(p.Name))
						case "Query":
							if t.Kind() != reflect.String {
								http.Error(w, "query parameter must be string", 500)
								return
							}
							v.Elem().SetString(r.URL.Query().Get(p.Name))
						case "Headers":
							if t.Kind() != reflect.String { http.Error(w, "header parameter must be string", 500); return }
							v.Elem().SetString(r.Header.Get(p.Name))
						case "Res", "Response":
							v.Elem().Set(reflect.ValueOf(writer))
						default:
							http.Error(w, "unsupported parameter decorator", 500)
							return
						}
					} else {
						for _, part := range strings.Split(spec.Path, "/") {
							if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
								v.Elem().SetString(r.PathValue(strings.Trim(part, "{}")))
								break
							}
						}
					}
					args[i] = v.Elem()
				}
				var result any
				if spec.StaticInvoke != nil && typ.NumIn() == 0 {
					value, err := spec.StaticInvoke(instance.Interface())
					if err != nil { _ = httpresponse.WriteError(responseCtx, err); return }
					result = value
					if handlerError, ok := value.(error); ok && handlerError != nil {
						_ = httpresponse.WriteError(responseCtx, handlerError); return
					}
				} else {
					results := handler.Call(args)
					if len(results) == 2 && !results[1].IsNil() {
						_ = httpresponse.WriteError(responseCtx, results[1].Interface().(error)); return
					}
					if len(results) == 1 && results[0].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
						if !results[0].IsNil() { _ = httpresponse.WriteError(responseCtx, results[0].Interface().(error)); return }
					} else if len(results) > 0 { result = results[0].Interface() }
				}
				if err := finishHTTPResponse(responseCtx, spec, result, writer, explicit, passthrough); err != nil {
					_ = httpresponse.WriteError(responseCtx, err)
				}
			})
			logger.Info(ctx, "RouterExplorer", "Mapped route", slog.String("method", route.Method), slog.String("path", path))
		}
	}
	logger.Startup(ctx, "RouterExplorer", "Routes mapped", routeStart)
	logger.Startup(ctx, "NestApplication", "Application handler initialized", boot)
	return mux, nil
}
func FromRegistry(names ...string) (http.Handler, error) {
	pkgs, err := registryImportClosure(names)
	if err != nil {
		return nil, err
	}
	return New(pkgs...)
}
