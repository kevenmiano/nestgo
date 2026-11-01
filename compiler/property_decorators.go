package compiler

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

// DTOField is a statically known property in a decorated Go DTO.
type DTOField struct {
	Name, Type, JSONTag string
	Annotations         []Annotation
}

// DTO describes a struct requiring generated JSON transformer methods.
type DTO struct {
	Name   string
	Fields []DTOField
}

func collectDTOs(path string, file *ast.File, fset *token.FileSet, marks []marker) ([]DTO, error) {
	consumed := make(map[int]bool, len(marks))
	var dtos []DTO
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, item := range gen.Specs {
			spec, ok := item.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := spec.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				continue
			}
			open := fset.Position(st.Fields.Opening).Offset
			close := fset.Position(st.Fields.Closing).Offset
			hasDecorators := false
			for i, mark := range marks {
				if mark.Offset > open && mark.Offset < close {
					consumed[i] = true
					hasDecorators = true
				}
			}
			if !hasDecorators {
				continue
			}
			dto := DTO{Name: spec.Name.Name}
			last := open
			for _, field := range st.Fields.List {
				start := fset.Position(field.Pos()).Offset
				anns := []Annotation{}
				for _, mark := range marks {
					if mark.Offset > last && mark.Offset < start {
						anns = append(anns, Annotation{Name: mark.Name, Argument: mark.Argument, Line: lineOfOffset(fset, file, mark.Offset)})
					}
				}
				if len(field.Names) == 0 {
					return nil, Diagnostic{path, fset.Position(field.Pos()).Line, "embedded fields are not supported in decorated DTOs"}
				}
				if len(anns) > 0 && len(field.Names) != 1 {
					return nil, Diagnostic{path, fset.Position(field.Pos()).Line, "property decorators require one field per declaration"}
				}
				var typ strings.Builder
				if err := format.Node(&typ, fset, field.Type); err != nil {
					return nil, err
				}
				tag := ""
				if field.Tag != nil {
					unquoted, err := strconv.Unquote(field.Tag.Value)
					if err != nil {
						return nil, Diagnostic{path, fset.Position(field.Tag.Pos()).Line, "invalid field tag"}
					}
					tag = reflect.StructTag(unquoted).Get("json")
				}
				for _, name := range field.Names {
					if !name.IsExported() && len(anns) > 0 {
						return nil, Diagnostic{path, fset.Position(name.Pos()).Line, "decorated DTO properties must be exported"}
					}
					if !name.IsExported() {
						continue
					}
					prop := DTOField{Name: name.Name, Type: typ.String(), JSONTag: tag, Annotations: anns}
					if err := validateDTOField(path, prop); err != nil {
						return nil, err
					}
					dto.Fields = append(dto.Fields, prop)
				}
				last = fset.Position(field.End()).Offset
			}
			for _, mark := range marks {
				if mark.Offset > last && mark.Offset < close {
					return nil, Diagnostic{path, lineOfOffset(fset, file, mark.Offset), "property decorator must precede a DTO field"}
				}
			}
			dtos = append(dtos, dto)
		}
	}
	for i, mark := range marks {
		if !consumed[i] {
			return nil, Diagnostic{path, lineOfOffset(fset, file, mark.Offset), "property decorator must be inside a struct"}
		}
	}
	return dtos, nil
}

func lineOfOffset(fset *token.FileSet, file *ast.File, offset int) int {
	position := fset.File(file.Pos())
	if position == nil || offset < 0 || offset > position.Size() {
		return 1
	}
	return fset.Position(position.Pos(offset)).Line
}

func validateDTOField(path string, field DTOField) error {
	var expose, exclude, excludeInput, excludeOutput bool
	for _, annotation := range field.Annotations {
		switch annotation.Name {
		case "Expose":
			if expose {
				return Diagnostic{path, annotation.Line, "duplicate @Expose"}
			}
			expose = true
			if annotation.Argument != "" {
				alias, err := strconv.Unquote(annotation.Argument)
				if err != nil || alias == "" || strings.Contains(alias, ",") {
					return Diagnostic{path, annotation.Line, "@Expose expects a non-empty quoted JSON name"}
				}
			}
		case "Exclude", "ExcludeInput", "ExcludeOutput":
			if annotation.Argument != "" {
				return Diagnostic{path, annotation.Line, "@" + annotation.Name + " takes no arguments; use @ExcludeInput() or @ExcludeOutput() to choose a direction"}
			}
			switch annotation.Name {
			case "Exclude":
				if exclude {
					return Diagnostic{path, annotation.Line, "duplicate @Exclude"}
				}
				exclude = true
			case "ExcludeInput":
				if excludeInput {
					return Diagnostic{path, annotation.Line, "duplicate @ExcludeInput"}
				}
				excludeInput = true
			case "ExcludeOutput":
				if excludeOutput {
					return Diagnostic{path, annotation.Line, "duplicate @ExcludeOutput"}
				}
				excludeOutput = true
			}
		case "Transform", "TransformInput", "TransformOutput":
			if _, err := transformerSpec(annotation.Argument); err != nil {
				return Diagnostic{path, annotation.Line, err.Error()}
			}
		case "Type":
			target := strings.TrimSpace(annotation.Argument)
			if target == "" {
				return Diagnostic{path, annotation.Line, "@Type expects a Go type"}
			}
			parsed, err := parserExpr(target)
			if err != nil {
				return Diagnostic{path, annotation.Line, "@Type expects a Go type"}
			}
			_ = parsed
			actual := strings.TrimPrefix(strings.TrimPrefix(field.Type, "[]"), "*")
			actual = strings.TrimPrefix(actual, "*")
			if actual != target {
				return Diagnostic{path, annotation.Line, "@Type must match the statically declared Go field type"}
			}
		}
	}
	if expose && exclude {
		return Diagnostic{path, field.Annotations[0].Line, "@Expose and @Exclude cannot decorate the same field"}
	}
	if exclude && (excludeInput || excludeOutput) {
		return Diagnostic{path, field.Annotations[0].Line, "@Exclude cannot be combined with @ExcludeInput or @ExcludeOutput"}
	}
	return nil
}

// transformerSpec accepts typed functions, or zero-value struct instances
// such as Trim{} and &Normalize{}. Arbitrary runtime calls and expressions
// are rejected; generated Go invokes the typed Transformer[T] interface.
func transformerSpec(argument string) (string, error) {
	name := strings.TrimSpace(argument)
	expr, err := parserExpr(name)
	if err != nil {
		return "", fmt.Errorf("@Transform expects a function or struct instance, such as Normalize or Trim{}")
	}
	switch expression := expr.(type) {
	case *ast.Ident:
		if expression.Name == "_" || expression.Name == "nil" || expression.Name == "true" || expression.Name == "false" {
			return "", fmt.Errorf("@Transform expects a function or struct instance")
		}
	case *ast.SelectorExpr:
		if _, ok := expression.X.(*ast.Ident); !ok {
			return "", fmt.Errorf("@Transform expects a function or struct instance")
		}
	case *ast.CompositeLit:
		if !validTransformerStructLiteral(expression) {
			return "", fmt.Errorf("@Transform struct must use a zero-value literal like Trim{}")
		}
	case *ast.UnaryExpr:
		literal, ok := expression.X.(*ast.CompositeLit)
		if expression.Op != token.AND || !ok || !validTransformerStructLiteral(literal) {
			return "", fmt.Errorf("@Transform struct must use a zero-value literal like &Trim{}")
		}
	default:
		return "", fmt.Errorf("@Transform expects a function or struct instance")
	}
	return name, nil
}

func validTransformerStructLiteral(literal *ast.CompositeLit) bool {
	if len(literal.Elts) != 0 {
		return false
	}
	switch typ := literal.Type.(type) {
	case *ast.Ident:
		return typ.Name != "_"
	case *ast.SelectorExpr:
		_, ok := typ.X.(*ast.Ident)
		return ok
	default:
		return false
	}
}

func transformerStructLiteral(name string) bool {
	expr, err := parserExpr(name)
	if err != nil {
		return false
	}
	switch expr.(type) {
	case *ast.CompositeLit, *ast.UnaryExpr:
		return true
	}
	return false
}

// transformerQualifier discovers the package alias for qualified functions
// and struct values, including &other.Normalize{}.
func transformerQualifier(name string) string {
	expr, err := parserExpr(name)
	if err != nil {
		return ""
	}
	switch value := expr.(type) {
	case *ast.SelectorExpr:
		id, ok := value.X.(*ast.Ident)
		if ok { return id.Name }
	case *ast.CompositeLit:
		if valueType, ok := value.Type.(*ast.SelectorExpr); ok {
			if id, ok := valueType.X.(*ast.Ident); ok { return id.Name }
		}
	case *ast.UnaryExpr:
		if literal, ok := value.X.(*ast.CompositeLit); ok {
			if valueType, ok := literal.Type.(*ast.SelectorExpr); ok {
				if id, ok := valueType.X.(*ast.Ident); ok { return id.Name }
			}
		}
	}
	return ""
}

// Kept separate from Parse so the lexer can retain the offsets of property markers.
func parserExpr(source string) (ast.Expr, error) {
	return parser.ParseExpr(source)
}

func (field DTOField) excluded(direction string) bool {
	for _, annotation := range field.Annotations {
		switch annotation.Name {
		case "Exclude":
			return true
		case "ExcludeInput":
			if direction == "input" {
				return true
			}
		case "ExcludeOutput":
			if direction == "output" {
				return true
			}
		}
	}
	return false
}

func (field DTOField) jsonName() string {
	tag := field.JSONTag
	for _, a := range field.Annotations {
		if a.Name == "Expose" && a.Argument != "" {
			alias, _ := strconv.Unquote(a.Argument)
			if _, opts, found := strings.Cut(tag, ","); found {
				return alias + "," + opts
			}
			return alias
		}
	}
	if tag == "" {
		return field.Name
	}
	if strings.HasPrefix(tag, ",") {
		return field.Name + tag
	}
	return tag
}

// propertyMethods emits direct field accesses and typed custom callback calls.
// Generated DTO methods are consumed automatically by encoding/json, including
// HTTP request binding and response serialization, regardless of HTTP adapter.
func propertyMethods(file *File) string {
	var out strings.Builder
	for _, dto := range file.DTOs {
		fmt.Fprintf(&out, "\nfunc (value %s) MarshalJSON() ([]byte, error) {\n", dto.Name)
		writeDTOJSONMethod(&out, dto, "output")
		fmt.Fprintf(&out, "\nfunc (value *%s) UnmarshalJSON(data []byte) error {\n", dto.Name)
		out.WriteString("if value == nil { return __nestgo_fmt.Errorf(\"cannot unmarshal into nil DTO\") }\n")
		writeDTOJSONMethod(&out, dto, "input")
	}
	return out.String()
}

func writeDTOJSONMethod(out *strings.Builder, dto DTO, direction string) {
	encoding := direction == "output"
	type indexedField struct {
		index int
		field DTOField
	}
	var active []indexedField
	for i, field := range dto.Fields {
		if field.excluded(direction) || field.jsonName() == "-" {
			continue
		}
		active = append(active, indexedField{i, field})
	}
	if !encoding {
		out.WriteString("var plain struct {\n")
		for _, entry := range active {
			fmt.Fprintf(out, "P%d %s %s\n", entry.index, entry.field.Type, strconv.Quote("json:\""+entry.field.jsonName()+"\""))
		}
		out.WriteString("}\n")
		out.WriteString("if err := __nestgo_json.Unmarshal(data, &plain); err != nil { return err }\n")
	}
	for _, entry := range active {
		i := entry.index
		field := entry.field
		if encoding {
			fmt.Fprintf(out, "p%d := value.%s\n", i, field.Name)
		} else {
			fmt.Fprintf(out, "p%d := plain.P%d\n", i, i)
		}
		n := 0
		for _, a := range field.Annotations {
			if a.Name != "Transform" &&
				!(a.Name == "TransformInput" && direction == "input") &&
				!(a.Name == "TransformOutput" && direction == "output") {
				continue
			}
			name, _ := transformerSpec(a.Argument)
			if transformerStructLiteral(name) {
				fmt.Fprintf(out, "next%d_%d, err%d_%d := __nestgo_transform.Apply[%s](p%d, %s)\n", i, n, i, n, field.Type, i, name)
			} else {
				fmt.Fprintf(out, "next%d_%d, err%d_%d := __nestgo_transform.ApplyFunc[%s](p%d, %s)\n", i, n, i, n, field.Type, i, name)
			}
			fmt.Fprintf(out, "if err%d_%d != nil { return ", i, n)
			if encoding {
				out.WriteString("nil, ")
			}
			fmt.Fprintf(out, "__nestgo_fmt.Errorf(%q, err%d_%d) }\n", dto.Name+"."+field.Name+": %w", i, n)
			fmt.Fprintf(out, "p%d = next%d_%d\n", i, i, n)
			n++
		}
		if !encoding {
			fmt.Fprintf(out, "value.%s = p%d\n", field.Name, i)
		}
	}
	if encoding {
		out.WriteString("return __nestgo_json.Marshal(struct {\n")
		for _, entry := range active {
			fmt.Fprintf(out, "P%d %s %s\n", entry.index, entry.field.Type, strconv.Quote("json:\""+entry.field.jsonName()+"\""))
		}
		out.WriteString("}{\n")
		for _, entry := range active {
			fmt.Fprintf(out, "P%d: p%d,\n", entry.index, entry.index)
		}
		out.WriteString("})\n}\n")
	} else {
		out.WriteString("return nil\n}\n")
	}
}
