package runtime

import (
    "fmt"
    "reflect"
)

// injectFields is shared by the legacy net/http and adapter-based runtimes.
// Dependency visibility and optional fields must behave the same in both paths.
func injectFields(v reflect.Value, providers map[reflect.Type]reflect.Value, order []reflect.Type, visibility map[reflect.Type]map[reflect.Type]bool) error {
    allowed, scoped := visibility[v.Type()]
    for i := 0; i < v.Elem().NumField(); i++ {
        field := v.Elem().Field(i)
        sf := v.Elem().Type().Field(i)
        if sf.Anonymous || !field.CanSet() || (field.Kind() != reflect.Ptr && field.Kind() != reflect.Interface) {
            continue
        }
        dep, found := providers[field.Type()]
        if scoped && !allowed[field.Type()] { found = false }
        if !found && field.Kind() == reflect.Interface {
            for _, candidateType := range order {
                if scoped && !allowed[candidateType] { continue }
                candidate := providers[candidateType]
                if candidate.Type().Implements(field.Type()) {
                    if found { return fmt.Errorf("ambiguous dependency %s.%s", v.Elem().Type(), sf.Name) }
                    dep, found = candidate, true
                }
            }
        }
        if !found {
            if sf.Tag.Get("nestgo") == "optional" { continue }
            return fmt.Errorf("missing required dependency %s.%s (%s)", v.Elem().Type(), sf.Name, field.Type())
        }
        if !dep.IsValid() || (dep.Kind() == reflect.Ptr && dep.IsNil()) {
            return fmt.Errorf("nil dependency %s.%s (%s)", v.Elem().Type(), sf.Name, field.Type())
        }
        field.Set(dep)
    }
    return nil
}

