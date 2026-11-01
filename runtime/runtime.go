package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kevenmiano/nestgo/logging"
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
	diStart := time.Now()
	graph, err := prepareBootstrap(packages, func(typ reflect.Type) {
		logDebug("InstanceLoader", "Provider instantiated", slog.String("provider", typ.String()))
	}, func() {
		logger.Info(ctx, "InstanceLoader", "Package providers registered")
	})
	if err != nil {
		return fail(err)
	}
	logger.Startup(ctx, "InstanceLoader", "Provider dependencies resolved", diStart)
	routeStart := time.Now()
	mux := http.NewServeMux()
	seen := map[string]bool{}
	for _, c := range graph.controllers {
		instance, err := graph.newController(c.Type)
		if err != nil {
			return fail(err)
		}
		logger.Info(ctx, "RoutesResolver", "Controller registered", slog.String("controller", c.Type.String()), slog.String("prefix", c.Prefix))
		for _, route := range c.Routes {
			method := instance.MethodByName(route.Handler)
			if !method.IsValid() {
				return fail(fmt.Errorf("missing handler %s.%s", c.Type, route.Handler))
			}
			typ := method.Type()
			if typ.NumOut() < 1 || typ.NumOut() > 2 || (typ.NumOut() == 2 && !typ.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem())) {
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
			for i := 0; i < typ.NumIn(); i++ {
				if _, ok := bindings[i]; !ok && typ.In(i).Kind() != reflect.String {
					return fail(fmt.Errorf("missing decorator for argument %d of %s", i, route.Handler))
				}
			}
			handler := method
			spec := route
			mux.HandleFunc(key, func(w http.ResponseWriter, r *http.Request) {
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
					if err != nil {
						http.Error(w, err.Error(), 500)
						return
					}
					result = value
				} else {
					results := handler.Call(args)
					if len(results) == 2 && !results[1].IsNil() {
						http.Error(w, results[1].Interface().(error).Error(), 500)
						return
					}
					result = results[0].Interface()
				}
				status := spec.Status
				if status == 0 {
					if spec.Method == "POST" {
						status = 201
					} else {
						status = 200
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(result)
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
