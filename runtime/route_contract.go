package runtime

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/http"
	"github.com/kevenmiano/nestgo/metadata"
)

// routeContract is the transport-independent route validation result.
type routeContract struct {
	Path     string
	Key      string
	Bindings map[int]metadata.Parameter
}

type routeContractError struct {
	kind    string
	handler string
	index   int
}

func (e *routeContractError) Error() string {
	return formatRouteContractError(e, false).Error()
}

// formatRouteContractError preserves the original public diagnostics while
// both bootstrap paths share the same validation rules.
func formatRouteContractError(err error, legacy bool) error {
	e, ok := err.(*routeContractError)
	if !ok {
		return err
	}
	switch e.kind {
	case "signature":
		if legacy {
			return fmt.Errorf("unsupported signature %s", e.handler)
		}
		return fmt.Errorf("invalid signature %s", e.handler)
	case "index":
		return fmt.Errorf("invalid parameter index %d", e.index)
	case "duplicate":
		return fmt.Errorf("duplicate parameter index %d in %s", e.index, e.handler)
	case "type":
		return fmt.Errorf("parameter %d of %s requires a string for Param/Query", e.index, e.handler)
	case "decorator":
		return fmt.Errorf("unsupported parameter decorator at index %d of %s", e.index, e.handler)
	case "response":
		return fmt.Errorf("@Res argument must accept *http.Writer exactly once in %s", e.handler)
	case "missing":
		if legacy {
			return fmt.Errorf("missing decorator for argument %d of %s", e.index, e.handler)
		}
		return fmt.Errorf("missing parameter decorator for %s", e.handler)
	}
	return fmt.Errorf("invalid route %s", e.handler)
}

func compileRouteContract(prefix string, route metadata.Route, typ reflect.Type) (routeContract, error) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	// Plain handlers use (T) or (T, error). Explicit @Res handlers may
	// additionally return void or the error interface; concrete error values
	// are rejected because their reflected nil semantics are unsafe.
	if typ.NumOut() > 2 || (typ.NumOut() == 2 && typ.Out(1) != errType) {
		return routeContract{}, &routeContractError{kind: "signature", handler: route.Handler}
	}
	path := strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(route.Path, "/")
	bindings := make(map[int]metadata.Parameter, len(route.Parameters))
	for _, p := range route.Parameters {
		if p.Index < 0 || p.Index >= typ.NumIn() {
			return routeContract{}, &routeContractError{kind: "index", handler: route.Handler, index: p.Index}
		}
		if _, duplicate := bindings[p.Index]; duplicate {
			return routeContract{}, &routeContractError{kind: "duplicate", handler: route.Handler, index: p.Index}
		}
		switch p.Kind {
		case "Body":
		case "Param", "Query", "Headers":
			if typ.In(p.Index).Kind() != reflect.String {
				return routeContract{}, &routeContractError{kind: "type", handler: route.Handler, index: p.Index}
			}
		case "Res", "Response":
			if reflect.TypeOf((*http.Writer)(nil)) != typ.In(p.Index) {
				return routeContract{}, &routeContractError{kind: "response", handler: route.Handler, index: p.Index}
			}
		default:
			return routeContract{}, &routeContractError{kind: "decorator", handler: route.Handler, index: p.Index}
		}
		bindings[p.Index] = p
	}
	explicit := false
	for i := 0; i < typ.NumIn(); i++ {
		if p, ok := bindings[i]; ok && (p.Kind == "Res" || p.Kind == "Response") {
			if explicit {
				return routeContract{}, &routeContractError{kind: "response", handler: route.Handler, index: i}
			}
			explicit = true
		} else if !ok && typ.In(i).Kind() != reflect.String {
			return routeContract{}, &routeContractError{kind: "missing", handler: route.Handler, index: i}
		}
	}
	if typ.NumOut() == 0 && !explicit {
		return routeContract{}, &routeContractError{kind: "signature", handler: route.Handler}
	}
	if typ.NumOut() == 1 && typ.Out(0).Implements(errType) &&
		(!explicit || typ.Out(0) != errType) {
		return routeContract{}, &routeContractError{kind: "signature", handler: route.Handler}
	}
	return routeContract{Path: path, Key: route.Method + " " + path, Bindings: bindings}, nil
}
