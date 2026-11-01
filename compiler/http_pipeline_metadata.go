package compiler

import (
	"fmt"
	"go/token"
	"strings"
)

// pipelineMetadata encodes class references as concrete reflect.Type values.
// Only local @Injectable providers are accepted: no runtime name resolution.
func pipelineMetadata(path string, annotations []Annotation, providers map[string]bool) (string, error) {
	var fields strings.Builder
	for _, item := range []struct{ Decorator, Field string }{
		{"UseGuards", "Guards"}, {"UsePipes", "Pipes"}, {"UseFilters", "Filters"},
	} {
		names := []string{}
		seen := map[string]bool{}
		for _, a := range annotations {
			if a.Name != item.Decorator {
				continue
			}
			if a.Argument == "" {
				return "", Diagnostic{File: path, Line: a.Line, Message: "@" + a.Name + " requires one or more local provider types"}
			}
			// Accept both @UseGuards(A, B) and @UseGuards([A, B]).
			arg := strings.TrimSpace(a.Argument)
			if strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") {
				arg = strings.TrimSpace(arg[1 : len(arg)-1])
			}
			for _, raw := range strings.Split(arg, ",") {
				name := strings.TrimSpace(raw)
				if !token.IsIdentifier(name) || !providers[name] {
					return "", Diagnostic{File: path, Line: a.Line, Message: fmt.Sprintf("@%s: %q must be a local @Injectable provider", a.Name, name)}
				}
				if seen[name] {
					return "", Diagnostic{File: path, Line: a.Line, Message: fmt.Sprintf("@%s: duplicate provider %s", a.Name, name)}
				}
				seen[name] = true
				names = append(names, "reflect.TypeOf((*"+name+")(nil))")
			}
		}
		if len(names) > 0 {
			fmt.Fprintf(&fields, ",%s:[]reflect.Type{%s}", item.Field, strings.Join(names, ","))
		}
	}
	return fields.String(), nil
}
