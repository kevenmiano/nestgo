package metadata

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/kevenmiano/nestgo/httpadapter"
)

type Parameter struct {
	Index int
	Kind  string
	Name  string
}
type Route struct {
	StaticInvoke          func(any) (any, error)
	StaticHTTP            func(any, httpadapter.Context) (any, int, error)
	Method, Path, Handler string
	Status                int
	Parameters            []Parameter
}
type Controller struct {
	Type   reflect.Type
	Prefix string
	Routes []Route
}
type Provider struct {
	Type    reflect.Type
	Scope   string
	Factory func(resolve func(reflect.Type) (any, bool)) (any, error)
}

// Module declares an optional static module dependency contract.
// Imports are pointer-to-struct module types. Exports are provider types.
// Old generated Module{Type, Name} initializers remain valid.
type Module struct {
	Type        reflect.Type
	Name        string
	Imports     []reflect.Type
	Providers   []reflect.Type
	Controllers []reflect.Type
	Exports     []reflect.Type
}
type Package struct {
	Controllers []Controller
	Providers   []Provider
	Modules     []Module
}

var mu sync.RWMutex
var packages = map[string]Package{}

func Register(name string, p Package) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := packages[name]; ok {
		panic(fmt.Sprintf("nestgo: duplicate metadata package %s", name))
	}
	packages[name] = p
}
func Lookup(name string) (Package, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := packages[name]
	return p, ok
}
func All() map[string]Package {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]Package, len(packages))
	for k, v := range packages {
		result[k] = v
	}
	return result
}
