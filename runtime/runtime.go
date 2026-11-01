package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kevenmiano/nestgo/http"
	"github.com/kevenmiano/nestgo/logging"
	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/platform/nethttp"
	"log/slog"
	stdhttp "net/http"
	"reflect"
	"strings"
	"time"
)

func New(packages ...metadata.Package) (stdhttp.Handler, error) {
	return NewWithOptions(Options{}, packages...)
}

// Options controls framework lifecycle diagnostics without affecting request handling.
type Options struct {
	Logger *logging.Service
	Debug  bool
}

func NewWithOptions(options Options, packages ...metadata.Package) (stdhttp.Handler, error) {
	logger := options.Logger
	if logger == nil {
		logger = logging.New(logging.Config{MinLevel: logging.LevelInfo})
	}
	ctx := context.Background()
	boot := time.Now()
	logger.Info(ctx, "NestFactory", "Starting NestGo application...")
	fail := func(err error) (stdhttp.Handler, error) {
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
	mux := stdhttp.NewServeMux()
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
			contract, contractErr := compileRouteContract(c.Prefix, route, typ)
			if contractErr != nil {
				return fail(formatRouteContractError(contractErr, true))
			}
			path := contract.Path
			key := contract.Key
			if seen[key] {
				return fail(fmt.Errorf("duplicate route %s", key))
			}
			seen[key] = true
			bindings := contract.Bindings
			explicit, passthrough := false, false
			for _, p := range bindings {
				if p.Kind == "Res" || p.Kind == "Response" {
					explicit, passthrough = true, p.Name == "passthrough"
				}
			}
			handler := method
			spec := route
			mux.HandleFunc(key, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				responseCtx := nethttp.NewContext(w, r)
				writer := http.NewWriter(responseCtx)
				routeResponseHeaders(responseCtx, spec)
				replyRoute := spec
				args := make([]reflect.Value, typ.NumIn())
				if spec.StaticHTTP == nil {
					for i := 0; i < typ.NumIn(); i++ {
						t := typ.In(i)
						v := reflect.New(t)
						if p, ok := bindings[i]; ok {
							switch p.Kind {
							case "Body":
								if err := json.NewDecoder(stdhttp.MaxBytesReader(w, r.Body, 1<<20)).Decode(v.Interface()); err != nil {
									stdhttp.Error(w, "invalid JSON: "+err.Error(), 400)
									return
								}
							case "Param":
								if t.Kind() != reflect.String {
									stdhttp.Error(w, "path parameter must be string", 500)
									return
								}
								v.Elem().SetString(r.PathValue(p.Name))
							case "Query":
								if t.Kind() != reflect.String {
									stdhttp.Error(w, "query parameter must be string", 500)
									return
								}
								v.Elem().SetString(r.URL.Query().Get(p.Name))
							case "Headers":
								if t.Kind() != reflect.String {
									stdhttp.Error(w, "header parameter must be string", 500)
									return
								}
								v.Elem().SetString(r.Header.Get(p.Name))
							case "Res", "Response":
								v.Elem().Set(reflect.ValueOf(writer))
							default:
								stdhttp.Error(w, "unsupported parameter decorator", 500)
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
				}
				var result any
				if spec.StaticHTTP != nil {
					value, status, err := spec.StaticHTTP(instance.Interface(), responseCtx)
					if err != nil {
						if status != 0 {
							_ = respondError(responseCtx, status, err.Error())
						} else {
							_ = http.WriteError(responseCtx, err)
						}
						return
					}
					result = value
					if handlerError, ok := value.(error); ok && handlerError != nil {
						_ = http.WriteError(responseCtx, handlerError)
						return
					}
					if status != 0 {
						replyRoute.Status = status
					}
				} else if spec.StaticInvoke != nil && typ.NumIn() == 0 {
					value, err := spec.StaticInvoke(instance.Interface())
					if err != nil {
						_ = http.WriteError(responseCtx, err)
						return
					}
					result = value
					if handlerError, ok := value.(error); ok && handlerError != nil {
						_ = http.WriteError(responseCtx, handlerError)
						return
					}
				} else {
					results := handler.Call(args)
					if len(results) == 2 && !results[1].IsNil() {
						_ = http.WriteError(responseCtx, results[1].Interface().(error))
						return
					}
					errType := reflect.TypeOf((*error)(nil)).Elem()
					if len(results) == 1 && results[0].Type().Implements(errType) {
						if !results[0].IsNil() {
							_ = http.WriteError(responseCtx, results[0].Interface().(error))
							return
						}
					} else if len(results) > 0 {
						result = results[0].Interface()
					}
				}
				if err := finishHTTPResponse(responseCtx, replyRoute, result, writer, explicit, passthrough); err != nil {
					_ = http.WriteError(responseCtx, err)
				}
			})
			logger.Info(ctx, "RouterExplorer", "Mapped route", slog.String("method", route.Method), slog.String("path", path))
		}
	}
	logger.Startup(ctx, "RouterExplorer", "Routes mapped", routeStart)
	logger.Startup(ctx, "NestApplication", "Application handler initialized", boot)
	return mux, nil
}
func FromRegistry(names ...string) (stdhttp.Handler, error) {
	pkgs, err := registryImportClosure(names)
	if err != nil {
		return nil, err
	}
	return New(pkgs...)
}
