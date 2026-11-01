package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
)

// validateRequiredConcreteDependencies rejects unregistered concrete pointer dependencies
// declared in the compiled source set. Dynamic and interface dependencies remain subject
// to the runtime injection contract.
func validateRequiredConcreteDependencies(files []*File, providers map[string]bool, controllers map[string]string) error {
	localStructs := make(map[string]bool)
	for _, file := range files {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file.Path, file.Source, 0)
		if err != nil {
			return err
		}
		for _, decl := range node.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				_, ok = typ.Type.(*ast.StructType)
				if ok {
					localStructs[typ.Name.Name] = true
				}
			}
		}
	}
	for _, file := range files {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file.Path, file.Source, 0)
		if err != nil {
			return err
		}
		for _, decl := range node.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok || (!providers[typ.Name.Name] && !isController(controllers, typ.Name.Name)) {
					continue
				}
				strct, ok := typ.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range strct.Fields.List {
					// Only explicit required bindings participate in static
					// provider resolution. Ordinary pointers are user state.
					tag := ""
					if field.Tag != nil {
						raw, err := strconv.Unquote(field.Tag.Value)
						if err != nil {
							return Diagnostic{File: file.Path, Line: fset.Position(field.Pos()).Line, Message: "invalid injection field tag"}
						}
						tag = reflect.StructTag(raw).Get("nestgo")
					}
					if tag != "inject" {
						continue
					}
					ptr, ok := field.Type.(*ast.StarExpr)
					if !ok {
						continue
					}
					dependency, ok := ptr.X.(*ast.Ident)
					if !ok || !localStructs[dependency.Name] || providers[dependency.Name] {
						continue
					}
					fieldName := dependency.Name
					if len(field.Names) != 0 {
						fieldName = field.Names[0].Name
					}
					return Diagnostic{
						File: file.Path, Line: fset.Position(field.Pos()).Line,
						Message: "missing required provider for " + typ.Name.Name + "." + fieldName + " (*" + dependency.Name + ")",
					}
				}
			}
		}
	}
	return nil
}

func isController(controllers map[string]string, name string) bool {
	_, ok := controllers[name]
	return ok
}
