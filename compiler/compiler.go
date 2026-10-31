package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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
	Kind           string
	Params         []Parameter
	StaticEligible bool
}
type Parameter struct {
	Index      int
	Kind, Name string
}
type File struct {
	Path, Package, Source string
	Declarations          []Declaration
}

var supported = map[string]bool{"Controller": true, "Injectable": true, "Module": true, "Get": true, "Post": true, "Put": true, "Patch": true, "Delete": true, "HttpCode": true, "Body": true, "Param": true, "Query": true, "Headers": true}

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
				if a.Name != "Controller" && a.Name != "Module" && a.Name != "Injectable" {
					return nil, Diagnostic{path, a.Line, "@" + a.Name + " cannot decorate a type"}
				}
			}
			result.Declarations = append(result.Declarations, d)
		case *ast.FuncDecl:
			d := Declaration{Name: n.Name.Name, Kind: "func", Annotations: ann}
			// Only zero-argument methods with exactly one return value are
			// statically invoked. Every other signature uses reflection.
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
				if !isRoute(a.Name) && a.Name != "HttpCode" {
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
	if len(files) == 0 {
		return nil, fmt.Errorf("no source files")
	}
	pkg := files[0].Package
	for _, f := range files {
		if f.Package != pkg {
			return nil, fmt.Errorf("mixed packages: %s and %s", pkg, f.Package)
		}
	}
	controllers := map[string]string{}
	providers := []string{}
	modules := []string{}
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
					providers = append(providers, fmt.Sprintf(`{Type: reflect.TypeOf((*%s)(nil)), Scope: "singleton"}`, d.Name))
				case "Module":
					name, err := arg(a, d.Name)
					if err != nil {
						return nil, err
					}
					modules = append(modules, fmt.Sprintf(`{Type: reflect.TypeOf((*%s)(nil)), Name: %q}`, d.Name, name))
				}
			}
		}
	}
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
	out.WriteString("// Code generated by NestGo compiler. DO NOT EDIT.\npackage " + pkg + "\n\nimport (\n\"reflect\"\n\"github.com/kevenmiano/nestgo/metadata\"\n)\n\n")
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
func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
func CompileDir(dir string) error {
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
	registry, err := Generate(files)
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
		source := []byte("// Code generated by NestGo compiler. DO NOT EDIT.\n" + file.Source)
		formatted, err := format.Source(source)
		if err != nil {
			return err
		}
		if err := writeIfChanged(target, formatted); err != nil {
			return err
		}
	}
	return writeIfChanged(filepath.Join(dir, "zz_modforge_metadata.go"), registry)
}
func writeIfChanged(path string, content []byte) error {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, content) {
		return nil
	}
	return os.WriteFile(path, content, 0644)
}
