package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
)

// validateModuleProviderVisibility applies to modules with explicit providers and
// controllers lists. Legacy modules without lists keep their historical behavior.
func validateModuleProviderVisibility(files []*File, modules map[string]moduleOptions) error {
	visible := make(map[string]map[string]bool, len(modules))
	for name, opts := range modules {
		if !opts.Defined["providers"] && !opts.Defined["controllers"] {
			continue
		}
		allowed := make(map[string]bool)
		for _, provider := range opts.Providers {
			allowed[provider] = true
		}
		for _, imported := range opts.Imports {
			if importedModule, ok := modules[imported]; ok {
				for _, exported := range importedModule.Exports {
					allowed[exported] = true
				}
			}
		}
		visible[name] = allowed
	}
	// A decorated provider can belong to multiple modules; it must resolve in
	// each explicitly configured module that declares it.
	for _, file := range files {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file.Path, file.Source, 0)
		if err != nil {
			return err
		}
		for _, declaration := range node.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typ.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range structType.Fields.List {
					if field.Tag != nil {
						tag, err := strconv.Unquote(field.Tag.Value)
						if err == nil && reflect.StructTag(tag).Get("nestgo") == "optional" {
							continue
						}
					}
					ptr, ok := field.Type.(*ast.StarExpr)
					if !ok {
						continue
					}
					dependency, ok := ptr.X.(*ast.Ident)
					if !ok {
						continue
					}
					for _, moduleName := range sortedModuleContracts(modules) {
						opts := modules[moduleName]
						if !containsModuleType(opts.Providers, typ.Name.Name) && !containsModuleType(opts.Controllers, typ.Name.Name) {
							continue
						}
						allowed, strict := visible[moduleName]
						if !strict {
							continue
						}
						// Reject only references to known local Injectable providers.
						if !isDeclaredModuleProvider(modules, dependency.Name) || allowed[dependency.Name] {
							continue
						}
						fieldName := dependency.Name
						if len(field.Names) > 0 {
							fieldName = field.Names[0].Name
						}
						return Diagnostic{File: file.Path, Line: fset.Position(field.Pos()).Line, Message: "module " + moduleName + ": provider " + typ.Name.Name + "." + fieldName + " requires inaccessible provider " + dependency.Name + " (check imports/exports)"}
					}
				}
			}
		}
	}
	return nil
}

func containsModuleType(types []string, name string) bool {
	for _, candidate := range types {
		if candidate == name {
			return true
		}
	}
	return false
}

func isDeclaredModuleProvider(modules map[string]moduleOptions, name string) bool {
	for _, opts := range modules {
		if containsModuleType(opts.Providers, name) {
			return true
		}
	}
	return false
}

func sortedModuleContracts(modules map[string]moduleOptions) []string {
	keys := make([]string, 0, len(modules))
	for name := range modules {
		keys = append(keys, name)
	}
	sortStrings(keys)
	return keys
}
