package runtime

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/metadata"
)

// routeContract is the transport-independent route validation result.
type routeContract struct {
	Path string
	Key string
	Bindings map[int]metadata.Parameter
}

type routeContractError struct {
	kind string
	handler string
	index int
}

func (e *routeContractError) Error() string {
	return formatRouteContractError(e, false).Error()
}

// formatRouteContractError preserves the original public diagnostics while
// both bootstrap paths share the same validation rules.
func formatRouteContractError(err error, legacy bool) error {
	e, ok := err.(*routeContractError)
	if !ok { return err }
	switch e.kind {
	case "signature":
		if legacy { return fmt.Errorf("unsupported signature %s", e.handler) }
		return fmt.Errorf("invalid signature %s", e.handler)
	case "index":
		return fmt.Errorf("invalid parameter index %d", e.index)
	case "missing":
		if legacy { return fmt.Errorf("missing decorator for argument %d of %s", e.index, e.handler) }
		return fmt.Errorf("missing parameter decorator for %s", e.handler)
	}
	return fmt.Errorf("invalid route %s", e.handler)
}

func compileRouteContract(prefix string, route metadata.Route, typ reflect.Type) (routeContract, error) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if typ.NumOut() < 1 || typ.NumOut() > 2 || (typ.NumOut() == 2 && !typ.Out(1).Implements(errType)) {
		return routeContract{}, &routeContractError{kind: "signature", handler: route.Handler}
	}
	path := strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(route.Path, "/")
	bindings := make(map[int]metadata.Parameter, len(route.Parameters))
	for _, p := range route.Parameters {
		if p.Index < 0 || p.Index >= typ.NumIn() {
			return routeContract{}, &routeContractError{kind: "index", handler: route.Handler, index: p.Index}
		}
		bindings[p.Index] = p
	}
	for i := 0; i < typ.NumIn(); i++ {
		if _, ok := bindings[i]; !ok && typ.In(i).Kind() != reflect.String {
			return routeContract{}, &routeContractError{kind: "missing", handler: route.Handler, index: i}
		}
	}
	return routeContract{Path: path, Key: route.Method + " " + path, Bindings: bindings}, nil
}
