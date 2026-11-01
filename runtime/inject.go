package runtime

import (
	"fmt"
	"reflect"
)

// InjectionError carries stable programmatic context for a rejected field
// binding. Errors expose no provider values or request data.
type InjectionError struct {
	Code       string
	Target     reflect.Type
	Field      string
	Dependency reflect.Type
	Message    string
}

func (e *InjectionError) Error() string { return e.Message }

func rejectInjection(code string, target reflect.Type, field string, dependency reflect.Type, format string, args ...any) error {
	return &InjectionError{
		Code: code, Target: target, Field: field,
		Dependency: dependency, Message: fmt.Sprintf(format, args...),
	}
}

// injectionAssignment is applied only after all fields have been validated.
// A malformed dependency must not leave a partially constructed provider.
type injectionAssignment struct {
	field reflect.Value
	value reflect.Value
}

// nilDependency detects both nil pointers and nil pointers wrapped in an
// interface. reflect.Value.IsNil must never be called on unrelated kinds.
func nilDependency(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// injectFields resolves dependencies before mutating any field. The explicit
// nestgo:"inject" marker declares a required dependency; "optional" permits
// an absent provider and "-" excludes a field from injection. Untagged fields
// are never treated as dependencies; every injection must be explicitly marked.
func injectFields(v reflect.Value, providers map[reflect.Type]reflect.Value, order []reflect.Type, visibility map[reflect.Type]map[reflect.Type]bool) error {
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return rejectInjection("invalid_target", nil, "", nil, "invalid injection target: expected non-nil pointer to struct")
	}
	allowed, scoped := visibility[v.Type()]
	var assignments []injectionAssignment
	for i := 0; i < v.Elem().NumField(); i++ {
		field := v.Elem().Field(i)
		sf := v.Elem().Type().Field(i)
		marker := sf.Tag.Get("nestgo")
		switch marker {
		case "", "inject", "optional", "-":
		default:
			return rejectInjection("invalid_tag", v.Elem().Type(), sf.Name, field.Type(), "invalid nestgo injection tag %q on %s.%s", marker, v.Elem().Type(), sf.Name)
		}
		if marker == "" || marker == "-" {
			continue
		}
		if sf.Anonymous || !field.CanSet() {
			if marker != "" {
				return rejectInjection("inaccessible_field", v.Elem().Type(), sf.Name, field.Type(), "cannot inject inaccessible field %s.%s", v.Elem().Type(), sf.Name)
			}
			continue
		}
		if field.Kind() != reflect.Ptr && field.Kind() != reflect.Interface {
			if marker != "" {
				return rejectInjection("invalid_field_type", v.Elem().Type(), sf.Name, field.Type(), "cannot inject non-pointer field %s.%s", v.Elem().Type(), sf.Name)
			}
			continue
		}
		dep, found := providers[field.Type()]
		if scoped && !allowed[field.Type()] {
			found = false
		}
		if !found && field.Kind() == reflect.Interface {
			for _, candidateType := range order {
				if scoped && !allowed[candidateType] {
					continue
				}
				candidate, present := providers[candidateType]
				if !present || !candidateType.Implements(field.Type()) {
					continue
				}
				if found {
					return rejectInjection("ambiguous_dependency", v.Elem().Type(), sf.Name, field.Type(), "ambiguous dependency %s.%s", v.Elem().Type(), sf.Name)
				}
				dep, found = candidate, true
			}
		}
		if !found {
			if marker == "optional" {
				continue
			}
			return rejectInjection("missing_dependency", v.Elem().Type(), sf.Name, field.Type(), "missing required dependency %s.%s (%s)", v.Elem().Type(), sf.Name, field.Type())
		}
		if nilDependency(dep) {
			return rejectInjection("nil_dependency", v.Elem().Type(), sf.Name, field.Type(), "nil dependency %s.%s (%s)", v.Elem().Type(), sf.Name, field.Type())
		}
		if !dep.Type().AssignableTo(field.Type()) {
			return rejectInjection("incompatible_dependency", v.Elem().Type(), sf.Name, field.Type(), "incompatible dependency %s.%s (%s)", v.Elem().Type(), sf.Name, field.Type())
		}
		assignments = append(assignments, injectionAssignment{field: field, value: dep})
	}
	for _, a := range assignments {
		a.field.Set(a.value)
	}
	return nil
}
