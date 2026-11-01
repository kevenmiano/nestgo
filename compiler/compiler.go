package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"github.com/kevenmiano/nestgo/database/sqlgen"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Diagnostic struct {
	File    string
	Line    int
	Message string
}

func (d Diagnostic) Error() string { return fmt.Sprintf("%s:%d: %s", d.File, d.Line, d.Message) }

type Annotation struct {
	Name, Argument string
	Line           int
}
type Declaration struct {
	Name, Receiver string
	Annotations    []Annotation
	StaticEligible bool
	Kind           string
	Params         []Parameter
}
type Parameter struct {
	Index      int
	Kind, Name string
}
type File struct {
	Path, Package, Source string
	Declarations          []Declaration
}

var supported = map[string]bool{"Controller": true, "Injectable": true, "Module": true, "Get": true, "Post": true, "Put": true, "Patch": true, "Delete": true, "HttpCode": true, "Body": true, "Param": true, "Query": true, "Headers": true, "Imports": true, "Exports": true, "Entity": true, "Repository": true}

func Parse(path string, src []byte) (*File, error) {
	source, annotations, paramMarkers, err := lexDecorators(path, src)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	result := &File{Path: path, Package: file.Name.Name, Source: source}
	typeNames := map[string]bool{}
	for _, node := range file.Decls {
		if gen, ok := node.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				if t, ok := spec.(*ast.TypeSpec); ok {
					typeNames[t.Name.Name] = true
				}
			}
		}
	}
	for _, node := range file.Decls {
		line := fset.Position(node.Pos()).Line
		ann := annotations[line]
		if len(ann) == 0 {
			continue
		}
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Tok != token.TYPE || len(n.Specs) != 1 {
				return nil, Diagnostic{path, line, "decorated type must be a single type declaration"}
			}
			spec, ok := n.Specs[0].(*ast.TypeSpec)
			if !ok {
				return nil, Diagnostic{path, line, "decorated type must be a struct"}
			}
			if _, ok := spec.Type.(*ast.StructType); !ok {
				return nil, Diagnostic{path, line, "decorated type must be a struct"}
			}
			d := Declaration{Name: spec.Name.Name, Kind: "type", Annotations: ann}
			for _, a := range ann {
				if a.Name != "Controller" && a.Name != "Module" && a.Name != "Injectable" && a.Name != "Imports" && a.Name != "Exports" && a.Name != "Entity" && a.Name != "Repository" {
					return nil, Diagnostic{path, a.Line, "@" + a.Name + " cannot decorate a type"}
				}
			}
			result.Declarations = append(result.Declarations, d)
		case *ast.FuncDecl:
			d := Declaration{Name: n.Name.Name, Kind: "func", Annotations: ann}
			// Statically invoke only zero-argument methods with exactly one result.
			d.StaticEligible = (n.Type.Params == nil || len(n.Type.Params.List) == 0) && n.Type.Results != nil && len(n.Type.Results.List) == 1 && len(n.Type.Results.List[0].Names) <= 1
			if n.Recv != nil && len(n.Recv.List) == 1 {
				switch r := n.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					if id, ok := r.X.(*ast.Ident); ok {
						d.Receiver = id.Name
					}
				case *ast.Ident:
					d.Receiver = r.Name
				}
			}
			for _, a := range ann {
				if !isRoute(a.Name) && a.Name != "HttpCode" && a.Name != "Query" {
					return nil, Diagnostic{path, a.Line, "@" + a.Name + " cannot decorate a function"}
				}
			}
			if d.Receiver == "" || !typeNames[d.Receiver] {
				return nil, Diagnostic{path, line, "route decorator requires local controller method"}
			}
			index := 0
			if n.Type.Params != nil {
				lowerBound := fset.Position(n.Type.Params.Opening).Offset
				for _, field := range n.Type.Params.List {
					count := len(field.Names)
					if count == 0 {
						count = 1
					}
					fieldStart := fset.Position(field.Pos()).Offset
					// Parameter decorators are stripped from the source before Go parsing.
					// Match only markers between the previous field and this field, rather
					// than repeatedly matching the first marker for every later field.
					for _, mark := range paramMarkers {
						if mark.Offset > lowerBound && mark.Offset < fieldStart {
							for j := 0; j < count; j++ {
								d.Params = append(d.Params, Parameter{Index: index + j, Kind: mark.Name, Name: mark.Argument})
							}
						}
					}
					index += count
					lowerBound = fset.Position(field.End()).Offset
				}
			}
			result.Declarations = append(result.Declarations, d)
		default:
			return nil, Diagnostic{path, line, "decorator not attached to supported declaration"}
		}
	}
	for line := range annotations {
		found := false
		for _, node := range file.Decls {
			if fset.Position(node.Pos()).Line == line {
				found = true
				break
			}
		}
		if !found {
			return nil, Diagnostic{path, line, "decorator not attached to supported declaration"}
		}
	}
	return result, nil
}
func isRoute(s string) bool {
	switch s {
	case "Get", "Post", "Put", "Patch", "Delete":
		return true
	}
	return false
}
func arg(a Annotation, def string) (string, error) {
	if a.Argument == "" {
		return def, nil
	}
	s, err := strconv.Unquote(a.Argument)
	if err != nil {
		return "", fmt.Errorf("line %d: decorator @%s expects quoted string", a.Line, a.Name)
	}
	return s, nil
}
func Generate(files []*File) ([]byte, error) {
	return generateMetadata(files, false)
}

func generateMetadata(files []*File, allowPersistence bool, repositories ...RepositoryDefinition) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no source files")
	}
	pkg := files[0].Package
	for _, f := range files {
		if f.Package != pkg {
			return nil, fmt.Errorf("mixed packages: %s and %s", pkg, f.Package)
		}
	}
	// Fail closed until the repository emitter and provider wiring are present.
	// Accepting annotations without generating executable repositories would
	// create a false sense of persistence support.
	for _, file := range files {
		for _, declaration := range file.Declarations {
			for _, annotation := range declaration.Annotations {
				if !allowPersistence && (annotation.Name == "Entity" || annotation.Name == "Repository" || (annotation.Name == "Query" && declaration.Kind == "func")) {
					return nil, Diagnostic{File: file.Path, Line: annotation.Line, Message: "persistence code generation is not yet implemented; refusing to silently ignore @" + annotation.Name}
				}
			}
		}
	}
	controllers := map[string]string{}
	providers := []string{}
	modules := []string{}
	moduleNames := map[string]bool{}
	moduleEdges := map[string][]string{}
	moduleLocations := map[string]Diagnostic{}
	moduleContracts := map[string]moduleOptions{}
	providerNames := map[string]bool{}
	routes := map[string][]string{}
	for _, f := range files {
		for _, d := range f.Declarations {
			if d.Kind != "type" {
				continue
			}
			for _, a := range d.Annotations {
				switch a.Name {
				case "Controller":
					p, err := arg(a, "/")
					if err != nil {
						return nil, err
					}
					controllers[d.Name] = p
				case "Injectable":
					providerNames[d.Name] = true
					providers = append(providers, fmt.Sprintf(`{Type: reflect.TypeOf((*%s)(nil)), Scope: "singleton"}`, d.Name))
				case "Module":
					moduleNames[d.Name] = true
				}
			}
		}
	}
	for _, repository := range repositories {
		if repository.Connection == "" {
			return nil, fmt.Errorf("repository: %s must declare a named connection for DI", repository.Name)
		}
		generatedType := repository.Name + "Generated"
		if providerNames[generatedType] {
			return nil, fmt.Errorf("repository: duplicate provider %s", generatedType)
		}
		providerNames[generatedType] = true
		providers = append(providers, fmt.Sprintf(`{Type:reflect.TypeOf((*%s)(nil)),Scope:"singleton",Factory:func(resolve func(reflect.Type)(any,bool))(any,error){value,ok:=resolve(reflect.TypeOf((*database.Connections)(nil)));if !ok{return nil,fmt.Errorf("repository %s: database connections not initialized")};return New%sGeneratedFromConnections(value.(*database.Connections))}}`, generatedType, repository.Name, repository.Name))
	}
	if err := validateRequiredConcreteDependencies(files, providerNames, controllers); err != nil {
		return nil, err
	}
	// Resolve module references after collecting all decorated types across files.
	for _, f := range files {
		for _, d := range f.Declarations {
			if d.Kind != "type" {
				continue
			}
			var moduleAnnotation *Annotation
			var imports, exports []Annotation
			for i := range d.Annotations {
				a := d.Annotations[i]
				switch a.Name {
				case "Module":
					if moduleAnnotation != nil {
						return nil, Diagnostic{f.Path, a.Line, "duplicate @Module decorator"}
					}
					moduleAnnotation = &d.Annotations[i]
				case "Imports":
					imports = append(imports, a)
				case "Exports":
					exports = append(exports, a)
				}
			}
			if len(imports)+len(exports) == 0 && moduleAnnotation == nil {
				continue
			}
			if moduleAnnotation == nil {
				a := append(imports, exports...)[0]
				return nil, Diagnostic{f.Path, a.Line, "@Imports and @Exports require @Module on the same type"}
			}
			if len(imports) > 1 || len(exports) > 1 {
				return nil, Diagnostic{f.Path, moduleAnnotation.Line, "only one @Imports and @Exports decorator allowed per module"}
			}
			options, err := parseModuleOptions(f.Path, *moduleAnnotation, d.Name)
			if err != nil {
				return nil, err
			}
			// A single local module owns generated repositories by default.
			// With multiple modules the declaration must choose the owner explicitly.
			for _, repository := range repositories {
				typ := repository.Name + "Generated"
				if len(moduleNames) == 1 {
					found := false
					for _, name := range options.Providers { if name == typ { found = true } }
					if !found { options.Providers = append(options.Providers, typ) }
					sortStrings(options.Providers)
				}
			}
			if err := validateModuleOptions(f.Path, moduleAnnotation.Line, options, moduleNames, providerNames, controllerTypeNames(controllers)); err != nil {
				return nil, err
			}
			if strings.HasPrefix(strings.TrimSpace(moduleAnnotation.Argument), "{") && (len(imports) > 0 || len(exports) > 0) {
				return nil, Diagnostic{f.Path, moduleAnnotation.Line, "do not mix @Module({...}) with legacy @Imports/@Exports decorators"}
			}
			name := options.Name
			var importNames, exportNames []string
			if len(imports) > 0 {
				importNames, err = moduleReferences(f.Path, imports[0], moduleNames, "module")
				if err != nil {
					return nil, err
				}
			}
			if len(exports) > 0 {
				exportNames, err = moduleReferences(f.Path, exports[0], providerNames, "provider")
				if err != nil {
					return nil, err
				}
			}
			if options.Defined["imports"] {
				importNames = options.Imports
			}
			if options.Defined["exports"] {
				exportNames = options.Exports
			}
			options.Imports = importNames
			options.Exports = exportNames
			moduleContracts[d.Name] = options
			for _, imported := range importNames {
				if !strings.Contains(imported, ".") {
					moduleEdges[d.Name] = append(moduleEdges[d.Name], imported)
				}
			}
			moduleLocations[d.Name] = Diagnostic{File: f.Path, Line: moduleAnnotation.Line}
			importsCode := []string{}
			for _, n := range importNames {
				importsCode = append(importsCode, fmt.Sprintf("reflect.TypeOf((*%s)(nil))", n))
			}
			exportsCode := []string{}
			for _, n := range exportNames {
				exportsCode = append(exportsCode, fmt.Sprintf("reflect.TypeOf((*%s)(nil))", n))
			}
			if len(imports) == 0 && len(exports) == 0 && !strings.HasPrefix(strings.TrimSpace(moduleAnnotation.Argument), "{") {
				// Preserve exact legacy generated metadata for plain @Module.
				modules = append(modules, fmt.Sprintf(`{Type: reflect.TypeOf((*%s)(nil)), Name: %q}`, d.Name, name))
			} else {
				providerCodes := []string{}
				for _, n := range options.Providers {
					providerCodes = append(providerCodes, fmt.Sprintf("reflect.TypeOf((*%s)(nil))", n))
				}
				controllerCodes := []string{}
				for _, n := range options.Controllers {
					controllerCodes = append(controllerCodes, fmt.Sprintf("reflect.TypeOf((*%s)(nil))", n))
				}
				modules = append(modules, fmt.Sprintf(`{Type:reflect.TypeOf((*%s)(nil)), Name:%q, Imports:[]reflect.Type{%s}, Providers:[]reflect.Type{%s}, Controllers:[]reflect.Type{%s}, Exports:[]reflect.Type{%s}}`,
					d.Name, name, strings.Join(importsCode, ","), strings.Join(providerCodes, ","), strings.Join(controllerCodes, ","), strings.Join(exportsCode, ",")))
			}
		}
	}
	if len(moduleNames) > 1 {
		for _, repository := range repositories {
			generated := repository.Name + "Generated"
			owners := 0
			for _, contract := range moduleContracts {
				for _, provider := range contract.Providers {
					if provider == generated { owners++ }
				}
			}
			if owners != 1 {
				return nil, fmt.Errorf("repository %s: expected exactly one owning @Module provider, got %d", generated, owners)
			}
		}
	}
	if err := validateModuleCycles(moduleEdges, moduleLocations); err != nil {
		return nil, err
	}
	if err := validateModuleProviderVisibility(files, moduleContracts); err != nil {
		return nil, err
	}
	if err := validateTypedInterfaceDependencies(files, moduleContracts, providerNames, controllers); err != nil {
		return nil, err
	}
	qualifiedImports, err := qualifiedModuleImportPaths(files, moduleContracts)
	if err != nil {
		return nil, err
	}
	// Module metadata order must not depend on source-file enumeration.
	sortStrings(modules)
	for _, f := range files {
		for _, d := range f.Declarations {
			if d.Kind != "func" {
				continue
			}
			status := 0
			routeCount := 0
			for _, a := range d.Annotations {
				if a.Name == "HttpCode" {
					n, err := strconv.Atoi(a.Argument)
					if err != nil || n < 100 || n > 599 {
						return nil, fmt.Errorf("%s:%d: invalid HTTP status", f.Path, a.Line)
					}
					status = n
				}
			}
			for _, a := range d.Annotations {
				if !isRoute(a.Name) {
					continue
				}
				routeCount++
				if _, ok := controllers[d.Receiver]; !ok {
					return nil, fmt.Errorf("%s:%d: %s is not decorated with @Controller", f.Path, a.Line, d.Receiver)
				}
				path, err := arg(a, "/")
				if err != nil {
					return nil, err
				}
				params := []string{}
				for _, p := range d.Params {
					params = append(params, fmt.Sprintf(`{Index:%d,Kind:%q,Name:%q}`, p.Index, p.Kind, p.Name))
				}
				staticInvoke := ""
				if d.StaticEligible {
					staticInvoke = fmt.Sprintf(",StaticInvoke:func(receiver any)(any,error){return receiver.(*%s).%s(),nil}", d.Receiver, d.Name)
				}
				routes[d.Receiver] = append(routes[d.Receiver], fmt.Sprintf(`{Method:%q,Path:%q,Handler:%q,Status:%d,Parameters:[]metadata.Parameter{%s}%s}`, strings.ToUpper(a.Name), path, d.Name, status, strings.Join(params, ","), staticInvoke))
			}
			if routeCount > 1 {
				return nil, fmt.Errorf("%s: multiple HTTP method decorators on %s", f.Path, d.Name)
			}
		}
	}
	var out strings.Builder
	out.WriteString("// Code generated by NestGo compiler. DO NOT EDIT.\npackage " + pkg + "\n\nimport (\n\"reflect\"\n\"github.com/kevenmiano/nestgo/metadata\"\n")
	if len(repositories) > 0 { out.WriteString("\"fmt\"\n\"github.com/kevenmiano/nestgo/database\"\n") }
	out.WriteString(")\n\n")
	for _, imported := range qualifiedImports {
		out.WriteString("import " + imported + "\n")
	}
	out.WriteString("func init() { metadata.Register(")
	out.WriteString(strconv.Quote(pkg))
	out.WriteString(", metadata.Package{\nControllers: []metadata.Controller{\n")
	names := make([]string, 0, len(controllers))
	for k := range controllers {
		names = append(names, k)
	}
	sortStrings(names)
	for _, name := range names {
		out.WriteString(fmt.Sprintf(`{Type:reflect.TypeOf((*%s)(nil)),Prefix:%q,Routes:[]metadata.Route{%s}},`, name, controllers[name], strings.Join(routes[name], ",")))
	}
	out.WriteString("},\nProviders: []metadata.Provider{")
	out.WriteString(strings.Join(providers, ","))
	out.WriteString("},\nModules: []metadata.Module{")
	out.WriteString(strings.Join(modules, ","))
	out.WriteString("},\n}) }\n")
	formatted, err := format.Source([]byte(out.String()))
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

// moduleReferences accepts only quoted comma-separated local Go identifiers.
// This deliberately excludes dynamic expressions and imported package aliases:
// cross-package modules need a future typed metadata/ownership contract.
func moduleReferences(path string, annotation Annotation, allowed map[string]bool, kind string) ([]string, error) {
	value, err := strconv.Unquote(annotation.Argument)
	if err != nil {
		return nil, Diagnostic{path, annotation.Line, "@" + annotation.Name + " expects a quoted comma-separated list of local type names"}
	}
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}
	seen := map[string]bool{}
	names := []string{}
	for _, raw := range strings.Split(value, ",") {
		name := strings.TrimSpace(raw)
		if name == "" || !token.IsIdentifier(name) || !allowed[name] {
			return nil, Diagnostic{path, annotation.Line, "unknown local " + kind + " " + strconv.Quote(name) + " in @" + annotation.Name}
		}
		if seen[name] {
			return nil, Diagnostic{path, annotation.Line, "duplicate " + kind + " " + name + " in @" + annotation.Name}
		}
		seen[name] = true
		names = append(names, name)
	}
	sortStrings(names)
	return names, nil
}

func controllerTypeNames(controllers map[string]string) map[string]bool {
	result := map[string]bool{}
	for name := range controllers {
		result[name] = true
	}
	return result
}
func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
func CompileDir(dir string) error {
	return CompileDirWithDialect(dir, sqlgen.PostgreSQL)
}

// CompileDirWithDialect compiles decorated repositories for a selected SQL dialect.
func CompileDirWithDialect(dir string, dialect sqlgen.Dialect) error {
	paths, err := filepath.Glob(filepath.Join(dir, "*.mgo"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no .mgo files in %s", dir)
	}
	files := []*File{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := Parse(path, raw)
		if err != nil {
			return err
		}
		files = append(files, file)
	}
	repositoryFiles := make(map[string][]byte)
	var repositoryDefinitions []RepositoryDefinition
	for _, file := range files {
		hasRepo := false
		for _, declaration := range file.Declarations {
			for _, annotation := range declaration.Annotations {
				switch annotation.Name {
				case "Repository":
					hasRepo = true
				case "Entity":
					// Entity metadata is not yet emitted. It is only supported as
					// a local model for a compiled repository in this file.
				case "Query":
					if declaration.Kind == "func" {
						matched := false
						for _, candidate := range file.Declarations {
							if candidate.Kind != "type" || candidate.Name != declaration.Receiver {
								continue
							}
							for _, a := range candidate.Annotations {
								if a.Name == "Repository" { matched = true }
							}
						}
						if !matched {
							return Diagnostic{File: file.Path, Line: annotation.Line, Message: "@Query requires a local @Repository receiver"}
						}
					}
				}
			}
		}
		if hasRepo {
			def, err := RepositoryFromMGO(file, dialect)
			if err != nil { return err }
			generated, err := GenerateRepository(def)
			if err != nil { return err }
			repositoryDefinitions = append(repositoryDefinitions, def)
			repositoryFiles[strings.TrimSuffix(file.Path, ".mgo")+"_repository_generated.go"] = generated
		}
	}
	if len(repositoryDefinitions) > 0 {
		providers, err := GenerateRepositoryProviders(files[0].Package, repositoryDefinitions)
		if err != nil { return err }
		repositoryFiles[filepath.Join(dir, "zz_modforge_repositories.go")] = providers
	}
	registry, err := generateMetadata(files, true, repositoryDefinitions...)
	if err != nil {
		return err
	}
	for _, file := range files {
		if _, err := parser.ParseFile(token.NewFileSet(), file.Path, file.Source, parser.AllErrors); err != nil {
			return err
		}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "zz_modforge_metadata.go", registry, parser.AllErrors); err != nil {
		return err
	}
	for _, file := range files {
		target := strings.TrimSuffix(file.Path, ".mgo") + "_generated.go"
		formatted, err := generatedSource(file)
		if err != nil {
			return err
		}
		if err := writeIfChanged(target, formatted); err != nil {
			return err
		}
	}
	if err := writeIfChanged(filepath.Join(dir, "zz_modforge_metadata.go"), registry); err != nil { return err }
	for target, source := range repositoryFiles {
		if err := writeIfChanged(target, source); err != nil { return err }
	}
	return nil
}
func writeIfChanged(path string, content []byte) error {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, content) {
		return nil
	}
	return os.WriteFile(path, content, 0644)
}
