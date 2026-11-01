package compiler

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var dtoValidationDecorators = map[string]bool{
	"IsEmail": true, "IsCPF": true, "IsCNPJ": true, "IsCEP": true,
	"IsCNH": true, "IsPIS": true, "IsPlate": true, "IsPhone": true,
	"IsURL": true, "IsUUID": true, "IsNotEmpty": true, "IsOptional": true,
	"Matches": true, "Regex": true, "MinLength": true, "MaxLength": true,
	"Length": true, "Min": true, "Max": true, "Validate": true,
}

func isDTOValidator(name string) bool { return dtoValidationDecorators[name] }

func hasDTOValidators(dto DTO) bool {
	for _, field := range dto.Fields {
		for _, annotation := range field.Annotations {
			if isDTOValidator(annotation.Name) {
				return true
			}
		}
	}
	return false
}

func anyDTOValidators(file *File) bool {
	for _, dto := range file.DTOs {
		if hasDTOValidators(dto) {
			return true
		}
	}
	return false
}

func hasRegexValidators(file *File) bool {
	for _, dto := range file.DTOs {
		for _, field := range dto.Fields {
			for _, a := range field.Annotations {
				if a.Name == "Regex" || a.Name == "Matches" {
					return true
				}
			}
		}
	}
	return false
}

func validateDTOValidators(path string, field DTOField) error {
	hasValidation := false
	optional := false
	for _, annotation := range field.Annotations {
		if !isDTOValidator(annotation.Name) {
			continue
		}
		hasValidation = true
		a := annotation
		if a.Name == "IsOptional" {
			optional = true
			if field.Type != "*string" && !strings.HasPrefix(field.Type, "*") {
				return Diagnostic{path, a.Line, "@IsOptional requires a pointer field to distinguish absent from zero values"}
			}
		}
		switch a.Name {
		case "Min", "Max":
			switch field.Type {
			case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
			default:
				return Diagnostic{path, a.Line, "@" + a.Name + " requires a numeric Go field"}
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(a.Argument), 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return Diagnostic{path, a.Line, "@" + a.Name + " expects a numeric literal"}
			}
		case "Validate":
			if _, err := transformerSpec(a.Argument); err != nil {
				return Diagnostic{path, a.Line, "@Validate expects a named Go function with signature func(T) error"}
			}
		case "MinLength", "MaxLength":
			n, err := strconv.Atoi(strings.TrimSpace(a.Argument))
			if err != nil || n < 0 {
				return Diagnostic{path, a.Line, "@" + a.Name + " expects a nonnegative integer"}
			}
		case "Length":
			parts := strings.Split(a.Argument, ",")
			if len(parts) != 2 {
				return Diagnostic{path, a.Line, "@Length expects min,max"}
			}
			min, errMin := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, errMax := strconv.Atoi(strings.TrimSpace(parts[1]))
			if errMin != nil || errMax != nil || min < 0 || max < min {
				return Diagnostic{path, a.Line, "@Length expects 0 <= min <= max"}
			}
		case "Regex", "Matches":
			pattern, err := strconv.Unquote(a.Argument)
			if err != nil {
				return Diagnostic{path, a.Line, "@" + a.Name + " expects a quoted regexp"}
			}
			if _, err := regexp.Compile(pattern); err != nil {
				return Diagnostic{path, a.Line, "invalid regexp: " + err.Error()}
			}
		default:
			if a.Argument != "" {
				return Diagnostic{path, a.Line, "@" + a.Name + " does not accept arguments"}
			}
		}
		switch a.Name {
		case "Min", "Max", "Validate", "IsOptional":
			// Numeric validation and custom validators have their own type checks.
		default:
			if field.Type != "string" && field.Type != "*string" {
				return Diagnostic{path, a.Line, "@" + a.Name + " requires string or *string field"}
			}
		}
	}
	if hasValidation && field.excluded("input") {
		return Diagnostic{path, field.Annotations[0].Line, "cannot validate a field excluded from input"}
	}
	if optional && field.Type != "*string" {
		// Generic optional pointer types are supported only with custom
		// validators or numeric Min/Max, never with string validators.
		for _, annotation := range field.Annotations {
			if isDTOValidator(annotation.Name) && annotation.Name != "IsOptional" && annotation.Name != "Validate" && annotation.Name != "Min" && annotation.Name != "Max" {
				return Diagnostic{path, annotation.Line, "optional field has incompatible validator"}
			}
		}
	}
	return nil
}

// generatedDTOValidation returns a direct-field-access validation method.
// Regexes are checked by the compiler and compiled once at package init.
func generatedDTOValidation(file *File) string {
	var out strings.Builder
	for _, dto := range file.DTOs {
		if !hasDTOValidators(dto) {
			continue
		}
		fmt.Fprintf(&out, "\nfunc (value %s) Validate() error {\n", dto.Name)
		out.WriteString("var issues __nestgo_validation.Errors\n")
		for _, field := range dto.Fields {
			if field.excluded("input") {
				continue
			}
			hasRules := false
			optional := false
			for _, a := range field.Annotations {
				if isDTOValidator(a.Name) {
					hasRules = true
					if a.Name == "IsOptional" {
						optional = true
					}
				}
			}
			if !hasRules {
				continue
			}
			out.WriteString("{\n")
			nullable := strings.HasPrefix(field.Type, "*")
			if nullable {
				if optional {
					fmt.Fprintf(&out, "if value.%s != nil {\n", field.Name)
				} else {
					fmt.Fprintf(&out, "if value.%s == nil { issues = append(issues, __nestgo_validation.FieldError{Field:%q,Rule:%q,Message:%q}) } else {\n", field.Name, field.jsonName(), "IsDefined", "value is required")
				}
				fmt.Fprintf(&out, "item := *value.%s\n", field.Name)
			} else {
				fmt.Fprintf(&out, "item := value.%s\n", field.Name)
			}
			for idx, a := range field.Annotations {
				if !isDTOValidator(a.Name) || a.Name == "IsOptional" {
					continue
				}
				name := a.Name
				expression := ""
				switch name {
				case "IsEmail", "IsCPF", "IsCNPJ", "IsCEP", "IsCNH", "IsPIS", "IsPlate", "IsPhone", "IsURL", "IsUUID", "IsNotEmpty":
					expression = fmt.Sprintf("__nestgo_validation.%s(item)", name)
				case "MinLength", "MaxLength":
					expression = fmt.Sprintf("__nestgo_validation.%s(item,%s)", name, strings.TrimSpace(a.Argument))
				case "Length":
					args := strings.Split(a.Argument, ",")
					expression = fmt.Sprintf("(__nestgo_validation.MinLength(item,%s) && __nestgo_validation.MaxLength(item,%s))", strings.TrimSpace(args[0]), strings.TrimSpace(args[1]))
				case "Min":
					expression = fmt.Sprintf("(item >= %s)", strings.TrimSpace(a.Argument))
				case "Max":
					expression = fmt.Sprintf("(item <= %s)", strings.TrimSpace(a.Argument))
				case "Matches", "Regex":
					pattern, _ := strconv.Unquote(a.Argument)
					// Use a quoted pattern and a precompiled package variable below.
					_ = pattern
					expression = fmt.Sprintf("__nestgo_validation.MatchRegexp(item,__nestgo_re_%s_%s_%d)", dto.Name, field.Name, idx)
				case "Validate":
					fmt.Fprintf(&out, "if err := __nestgo_validation.ApplyFunc[%s](item,%s); err != nil {\n", strings.TrimPrefix(field.Type, "*"), strings.TrimSpace(a.Argument))
					fmt.Fprintf(&out, "issues = append(issues, __nestgo_validation.FieldError{Field:%q,Rule:%q,Message:%q})\n}\n", field.jsonName(), name, "custom validation failed")
					continue
				}
				fmt.Fprintf(&out, "if !%s { issues = append(issues, __nestgo_validation.FieldError{Field:%q,Rule:%q,Message:%q}) }\n", expression, field.jsonName(), name, "validation failed: "+name)
			}
			if nullable {
				out.WriteString("}\n")
			}
			out.WriteString("}\n")
		}
		out.WriteString("if len(issues)>0 {return issues}\nreturn nil\n}\n")
	}
	return out.String()
}

func generatedRegexDeclarations(file *File) string {
	var out strings.Builder
	for _, dto := range file.DTOs {
		if !hasDTOValidators(dto) {
			continue
		}
		for _, field := range dto.Fields {
			for index, annotation := range field.Annotations {
				if annotation.Name != "Matches" && annotation.Name != "Regex" {
					continue
				}
				pattern, _ := strconv.Unquote(annotation.Argument)
				fmt.Fprintf(&out, "var __nestgo_re_%s_%s_%d = __nestgo_regexp.MustCompile(%q)\n", dto.Name, field.Name, index, pattern)
			}
		}
	}
	return out.String()
}
