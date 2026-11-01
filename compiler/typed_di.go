package compiler

import (
	"fmt"
	"go/ast"
	"go/importer"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"reflect"
	"strconv"
)

// validateTypedInterfaceDependencies resolves local interfaces using go/types.
// Cross-package/dynamic provider registration is not inferred from untrusted names.
func validateTypedInterfaceDependencies(files []*File, modules map[string]moduleOptions, providers map[string]bool, controllers map[string]string) error {
	fset := token.NewFileSet()
	parsed := make([]*ast.File, 0, len(files))
	for _, file := range files {
		node, err := parser.ParseFile(fset, file.Path, file.Source, 0)
		if err != nil {
			return err
		}
		parsed = append(parsed, node)
	}
	// Source may contain incomplete application code. Check declarations for
	// reliable interfaces and implementations without requiring function bodies.
	cfg := types.Config{Importer: moduleAwareImporter(fset, filepath.Dir(files[0].Path)), IgnoreFuncBodies: true, DisableUnusedImportCheck: true}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object), Types: make(map[ast.Expr]types.TypeAndValue)}
	pkg, err := cfg.Check(files[0].Package, fset, parsed, info)
	if err != nil {
		return fmt.Errorf("type-check dependency graph: %w", err)
	}
	if pkg == nil {
		return fmt.Errorf("type-check dependency graph: no package returned")
	}
	objects := make(map[string]*types.Named)
	for _, name := range pkg.Scope().Names() {
		obj, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if ok {
			objects[name] = named
		}
	}
	for _, node := range parsed {
		for _, declaration := range node.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok || (!providers[typ.Name.Name] && !isController(controllers, typ.Name.Name)) {
					continue
				}
				st, ok := typ.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					if field.Tag != nil {
						raw, err := strconv.Unquote(field.Tag.Value)
						if err == nil && reflect.StructTag(raw).Get("nestgo") == "optional" {
							continue
						}
					}
					fieldType := info.TypeOf(field.Type)
					if fieldType == nil {
						continue
					}
					iface, ok := types.Unalias(fieldType).Underlying().(*types.Interface)
					if !ok {
						continue
					}
					iface.Complete()
					moduleNames := sortedModuleContracts(modules)
					if len(moduleNames) == 0 {
						moduleNames = []string{""} // legacy packages have no explicit modules
					}
					for _, moduleName := range moduleNames {
						mod := modules[moduleName]
						if moduleName != "" && !containsModuleType(mod.Providers, typ.Name.Name) && !containsModuleType(mod.Controllers, typ.Name.Name) {
							continue
						}
						visible := make(map[string]bool)
						for _, name := range mod.Providers {
							visible[name] = true
						}
						for _, imported := range mod.Imports {
							for _, exported := range modules[imported].Exports {
								visible[exported] = true
							}
						}
						candidates := make([]string, 0)
						for _, provider := range sortedProviderNames(providers) {
							if (mod.Defined["providers"] || mod.Defined["controllers"]) && !visible[provider] {
								continue
							}
							candidate := objects[provider]
							if candidate != nil && types.Implements(types.NewPointer(candidate), iface) {
								candidates = append(candidates, provider)
							}
						}
						if len(candidates) == 1 {
							continue
						}
						fieldName := types.TypeString(fieldType, func(p *types.Package) string { return p.Name() })
						if len(field.Names) > 0 {
							fieldName = field.Names[0].Name
						}
						reason := "missing"
						if len(candidates) > 1 {
							reason = "ambiguous"
						}
						return Diagnostic{File: fset.Position(field.Pos()).Filename, Line: fset.Position(field.Pos()).Line, Message: fmt.Sprintf("%s interface dependency %s.%s (%s) in module %s: candidates %v", reason, typ.Name.Name, fieldName, fieldType.String(), moduleName, candidates)}
					}
				}
			}
		}
	}
	return nil
}

func sortedProviderNames(providers map[string]bool) []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sortStrings(names)
	return names
}

 
// moduleAwareImporter loads compiled export data in the source module context.
// go/importer.Default cannot resolve Go module replace directives or local modules.
func moduleAwareImporter(fset *token.FileSet, dir string) types.Importer {
	return importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		cmd := exec.Command("go", "list", "-export", "-f", "{{.Export}}", path)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("resolve import %q: %w: %s", path, err, strings.TrimSpace(string(output)))
		}
		exportFile := strings.TrimSpace(string(output))
		if exportFile == "" {
			return nil, fmt.Errorf("resolve import %q: no compiled export data", path)
		}
		return os.Open(exportFile)
	})
}
