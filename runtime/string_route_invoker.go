package runtime

import (
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

// compileStringRouteInvoker binds supported string arguments once at bootstrap.
// Other signatures retain the reflection fallback; generated StaticHTTP remains
// the preferred path for complex and JSON body bindings.
func compileStringRouteInvoker(method reflect.Value, route metadata.Route, bindings map[int]metadata.Parameter) func(httpadapter.Context) (any, error) {
	typeOf := method.Type()
	stringType := reflect.TypeOf("")
	if typeOf.NumIn() != 1 || typeOf.In(0) != stringType {
		return nil
	}
	var read func(httpadapter.Context) string
	if binding, ok := bindings[0]; ok {
		switch binding.Kind {
		case "Param":
			name := binding.Name
			read = func(ctx httpadapter.Context) string { return ctx.Param(name) }
		case "Query":
			name := binding.Name
			read = func(ctx httpadapter.Context) string { return ctx.Query(name) }
		case "Headers":
			name := binding.Name
			read = func(ctx httpadapter.Context) string { return ctx.Header(name) }
		default:
			return nil
		}
	} else {
		for _, part := range strings.Split(route.Path, "/") {
			if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
				name := strings.Trim(part, "{}")
				read = func(ctx httpadapter.Context) string { return ctx.Param(name) }
				break
			}
		}
		if read == nil {
			read = func(httpadapter.Context) string { return "" }
		}
	}
	switch {
	case typeOf.NumOut() == 1 && typeOf.Out(0) == stringType:
		fn, ok := method.Interface().(func(string) string)
		if !ok {
			return nil
		}
		return func(ctx httpadapter.Context) (any, error) { return fn(read(ctx)), nil }
	case typeOf.NumOut() == 2 && typeOf.Out(0) == stringType && typeOf.Out(1) == reflect.TypeOf((*error)(nil)).Elem():
		fn, ok := method.Interface().(func(string) (string, error))
		if !ok {
			return nil
		}
		return func(ctx httpadapter.Context) (any, error) {
			value, err := fn(read(ctx))
			return value, err
		}
	}
	return nil
}
