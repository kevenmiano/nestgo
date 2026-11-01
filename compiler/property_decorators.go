package compiler

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

// DTOField is a statically known property in a decorated Go DTO.
type DTOField struct {
	Name, Type, JSONTag string
	Annotations        []Annotation
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
						anns = append(anns, Annotation{Name: mark.Name, Argument: mark.Argument, Line: fset.Position(token.Pos(0)).Line})
						anns[len(anns)-1].Line = lineOfOffset(fset, file, mark.Offset)
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
	var expose, exclude bool
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
		case "Exclude":
			if exclude {
				return Diagnostic{path, annotation.Line, "duplicate @Exclude"}
			}
			exclude = true
			if annotation.Argument != "" && annotation.Argument != "\"toPlainOnly\"" && annotation.Argument != "\"toClassOnly\"" {
				return Diagnostic{path, annotation.Line, "@Exclude expects \"toPlainOnly\" or \"toClassOnly\""}
			}
		case "Transform":
			if _, _, err := transformerSpec(annotation.Argument); err != nil {
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
	return nil
}

// transform arguments are a Go function identifier (or package selector), with
// an optional direction. Reject arbitrary expressions in generated code.
func transformerSpec(argument string) (name, direction string, err error) {
	direction = "both"
	parts := strings.Split(argument, ",")
	if len(parts) < 1 || len(parts) > 2 {
		return "", "", fmt.Errorf("@Transform expects a function and optional direction")
	}
	name = strings.TrimSpace(parts[0])
	expr, parseErr := parserExpr(name)
	if parseErr != nil {
		return "", "", fmt.Errorf("@Transform expects a function identifier")
	}
	switch expression := expr.(type) {
	case *ast.Ident:
		if expression.Name == "_" {
			return "", "", fmt.Errorf("@Transform expects a function identifier")
		}
	case *ast.SelectorExpr:
		if _, ok := expression.X.(*ast.Ident); !ok {
			return "", "", fmt.Errorf("@Transform expects a function identifier")
		}
	default:
		return "", "", fmt.Errorf("@Transform expects a function identifier")
	}
	if len(parts) == 2 {
		value, unquoteErr := strconv.Unquote(strings.TrimSpace(parts[1]))
		if unquoteErr != nil || (value != "toClassOnly" && value != "toPlainOnly") {
			return "", "", fmt.Errorf("@Transform direction must be \"toClassOnly\" or \"toPlainOnly\"")
		}
		direction = value
	}
	return name, direction, nil
}

// Kept separate from Parse so the lexer can retain the offsets of property markers.
func parserExpr(source string) (ast.Expr, error) {
	return parser.ParseExpr(source)
}

func (field DTOField) excluded(direction string) bool {
	for _, a := range field.Annotations {
		if a.Name != "Exclude" {
			continue
		}
		if a.Argument == "" || a.Argument == strconv.Quote(direction) {
			return true
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
	return tag
}

// propertyMethods emits direct field accesses and typed custom callback calls.
// Generated DTO methods are consumed automatically by encoding/json, including
// HTTP request binding and response serialization, regardless of HTTP adapter.
func propertyMethods(file *File) string {
	var out strings.Builder
	for _, dto := range file.DTOs {
		fmt.Fprintf(&out, "\nfunc (value %s) MarshalJSON() ([]byte, error) {\n", dto.Name)
		writeDTOJSONMethod(&out, dto, "toPlainOnly")
		fmt.Fprintf(&out, "\nfunc (value *%s) UnmarshalJSON(data []byte) error {\n", dto.Name)
		out.WriteString("if value == nil { return __nestgo_fmt.Errorf(\"cannot unmarshal into nil DTO\") }\n")
		writeDTOJSONMethod(&out, dto, "toClassOnly")
	}
	return out.String()
}

func writeDTOJSONMethod(out *strings.Builder, dto DTO, direction string) {
	encoding := direction == "toPlainOnly"
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
			if a.Name != "Transform" {
				continue
			}
			name, use, _ := transformerSpec(a.Argument)
			if use != "both" && use != direction {
				continue
			}
			fmt.Fprintf(out, "next%d_%d, err%d_%d := %s(p%d)\n", i, n, i, n, name, i)
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
