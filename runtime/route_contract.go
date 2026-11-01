package runtime

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/metadata"
)

// routeContract is the HTTP-engine-independent validation result.
type routeContract struct {
	Path     string
	Key      string
	Bindings map[int]metadata.Parameter
}

func compileRouteContract(prefix string, route metadata.Route, method reflect.Type) (routeContract, error) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if method.NumOut() < 1 || method.NumOut() > 2 || (method.NumOut() == 2 && !method.Out(1).Implements(errType)) {
		return routeContract{}, fmt.Errorf("invalid signature %s", route.Handler)
	}
	path := strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(route.Path, "/")
	bindings := make(map[int]metadata.Parameter, len(route.Parameters))
	for _, p := range route.Parameters {
		if p.Index < 0 || p.Index >= method.NumIn() {
			return routeContract{}, fmt.Errorf("invalid parameter index %d", p.Index)
		}
		bindings[p.Index] = p
	}
	for i := 0; i < method.NumIn(); i++ {
		if _, ok := bindings[i]; !ok && method.In(i).Kind() != reflect.String {
			return routeContract{}, fmt.Errorf("missing parameter decorator for %s", route.Handler)
		}
	}
	return routeContract{Path: path, Key: route.Method + " " + path, Bindings: bindings}, nil
}
