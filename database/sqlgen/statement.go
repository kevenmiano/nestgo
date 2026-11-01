package sqlgen

import "fmt"

// Statement is an immutable, precompiled SQL template. Parameter names are
// compiler-supplied metadata rather than inferred through runtime reflection.
type Statement struct {
	query string
	order []int
	names []string
}

// Compile validates the named SQL template once and resolves the positional
// binding plan. Declared names must come from trusted generated source.
func Compile(query string, names []string, dialect Dialect) (Statement, error) {
	seen := make(map[string]bool, len(names))
	values := make(map[string]any, len(names))
	for i, name := range names {
		if !validName(name) || seen[name] {
			return Statement{}, fmt.Errorf("sqlgen: invalid or duplicate declared parameter %q", name)
		}
		seen[name] = true
		values[name] = i
	}
	bound, args, err := Bind(query, values, dialect)
	if err != nil {
		return Statement{}, err
	}
	order := make([]int, len(args))
	for i, arg := range args {
		order[i] = arg.(int)
	}
	return Statement{query: bound, order: order, names: append([]string(nil), names...)}, nil
}

func validName(name string) bool {
	if len(name) == 0 || !identStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !identPart(name[i]) {
			return false
		}
	}
	return true
}

func (s Statement) SQL() string { return s.query }

// Bind returns arguments only; SQL text cannot change based on user inputs.
func (s Statement) Bind(params map[string]any) ([]any, error) {
	if len(params) != len(s.names) {
		return nil, fmt.Errorf("sqlgen: expected %d parameters, got %d", len(s.names), len(params))
	}
	values := make([]any, len(s.names))
	for i, name := range s.names {
		v, ok := params[name]
		if !ok {
			return nil, fmt.Errorf("sqlgen: missing parameter %q", name)
		}
		values[i] = v
	}
	args := make([]any, len(s.order))
	for i, idx := range s.order {
		args[i] = values[idx]
	}
	return args, nil
}
