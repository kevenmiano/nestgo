package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// collectPackageTransformerStructs resolves local type names once during
// compilation, not at request time. Source can be a .mgo or ordinary .go file.
// A bare struct name in @TransformInput(Name) becomes a typed new(Name) call;
// a bare function name keeps using the existing function adapter.
func collectPackageTransformerStructs(dir string, files []*File) (map[string]bool, error) {
	structs := make(map[string]bool)
	for _, file := range files {
		for name := range file.StructTypes {
			structs[name] = true
		}
	}

	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return structs, nil
	}
	for _, path := range paths {
		base := filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_generated.go") ||
			strings.HasPrefix(base, "zz_modforge_") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return nil, err
		}
		if parsed.Name.Name != files[0].Package {
			continue
		}
		for _, declaration := range parsed.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				named, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, isStruct := named.Type.(*ast.StructType); isStruct {
					structs[named.Name.Name] = true
				}
			}
		}
	}
	return structs, nil
}
