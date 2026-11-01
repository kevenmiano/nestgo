package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

// RepositoryFromMGO extracts a typed repository plan from decorated Go source.
// It rejects unsupported signatures rather than silently changing semantics.
// Initial syntax: @Repository(User) on a struct, @Query on its methods.
func RepositoryFromMGO(file *File, dialect sqlgen.Dialect) (RepositoryDefinition, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file.Path, file.Source, 0)
	if err != nil {
		return RepositoryDefinition{}, err
	}
	var result RepositoryDefinition
	entities := make(map[string]*ast.StructType)
	repositoryTypes := make(map[string]string)
	connectionNames := make(map[string]string)
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
			if st, ok := typ.Type.(*ast.StructType); ok {
				entities[typ.Name.Name] = st
			}
		}
	}
	for _, declaration := range file.Declarations {
		for _, annotation := range declaration.Annotations {
			if annotation.Name != "Repository" {
				continue
			}
			entityName, connectionName, parseErr := parseRepositoryBinding(annotation.Argument)
			if declaration.Kind != "type" || parseErr != nil {
				return RepositoryDefinition{}, fmt.Errorf("repository: expected @Repository(Entity) on a struct")
			}
			repositoryTypes[declaration.Name] = entityName
			connectionNames[declaration.Name] = connectionName
		}
	}
	if len(repositoryTypes) != 1 {
		return RepositoryDefinition{}, fmt.Errorf("repository: expected exactly one decorated repository in %s", file.Path)
	}
	for name, entity := range repositoryTypes {
		if entities[entity] == nil {
			return RepositoryDefinition{}, fmt.Errorf("repository: entity %s must be a local struct", entity)
		}
		result = RepositoryDefinition{Package: file.Package, Name: name, Entity: entity, Connection: connectionNames[name], Dialect: dialect}
		for _, field := range entities[entity].Fields.List {
			for _, id := range field.Names {
				if !id.IsExported() {
					return RepositoryDefinition{}, fmt.Errorf("repository: entity field %s must be exported", id.Name)
				}
				result.Fields = append(result.Fields, RepositoryField{Name: id.Name})
			}
		}
	}
	for _, declaration := range node.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		receiver := ""
		switch typ := fn.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if id, ok := typ.X.(*ast.Ident); ok {
				receiver = id.Name
			}
		case *ast.Ident:
			receiver = typ.Name
		}
		if receiver != result.Name {
			continue
		}
		var sqlText string
		for _, declaration := range file.Declarations {
			if declaration.Kind != "func" || declaration.Receiver != receiver || declaration.Name != fn.Name.Name {
				continue
			}
			for _, ann := range declaration.Annotations {
				if ann.Name == "Query" {
					sqlText, err = strconv.Unquote(ann.Argument)
					if err != nil {
						return RepositoryDefinition{}, fmt.Errorf("repository: @Query expects a quoted SQL string")
					}
				}
			}
		}
		if sqlText == "" {
			continue
		}
		method := RepositoryMethod{Name: fn.Name.Name, SQL: strings.TrimSpace(sqlText)}
		if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
			return RepositoryDefinition{}, fmt.Errorf("repository: %s requires context.Context", method.Name)
		}
		first := fn.Type.Params.List[0]
		if len(first.Names) != 1 || first.Names[0].Name != "ctx" {
			return RepositoryDefinition{}, fmt.Errorf("repository: %s first parameter must be ctx context.Context", method.Name)
		}
		if selector, ok := first.Type.(*ast.SelectorExpr); !ok || selector.Sel.Name != "Context" {
			return RepositoryDefinition{}, fmt.Errorf("repository: %s first argument must be context.Context", method.Name)
		}
		for _, parameter := range fn.Type.Params.List[1:] {
			if len(parameter.Names) != 1 {
				return RepositoryDefinition{}, fmt.Errorf("repository: %s parameters must be named individually", method.Name)
			}
			typ, ok := parameter.Type.(*ast.Ident)
			if !ok || !repositorySafeType(typ.Name) {
				return RepositoryDefinition{}, fmt.Errorf("repository: unsupported type in %s", method.Name)
			}
			method.Parameters = append(method.Parameters, RepositoryParameter{Name: parameter.Names[0].Name, Type: typ.Name})
		}
		if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 {
			return RepositoryDefinition{}, fmt.Errorf("repository: %s must return (*Entity, error), ([]Entity, error), or (int64, error) for rows affected", method.Name)
		}
		resultType := fn.Type.Results.List[0].Type
		switch typ := resultType.(type) {
		case *ast.Ident:
			if typ.Name != "int64" {
				return RepositoryDefinition{}, fmt.Errorf("repository: %s returns unsupported scalar type", method.Name)
			}
			method.Exec = true
		case *ast.StarExpr:
			id, ok := typ.X.(*ast.Ident)
			if !ok || id.Name != result.Entity {
				return RepositoryDefinition{}, fmt.Errorf("repository: %s returns wrong entity", method.Name)
			}
		case *ast.ArrayType:
			id, ok := typ.Elt.(*ast.Ident)
			if !ok || id.Name != result.Entity {
				return RepositoryDefinition{}, fmt.Errorf("repository: %s returns wrong entity", method.Name)
			}
			method.Many = true
		default:
			return RepositoryDefinition{}, fmt.Errorf("repository: %s returns unsupported type", method.Name)
		}
		if e, ok := fn.Type.Results.List[1].Type.(*ast.Ident); !ok || e.Name != "error" {
			return RepositoryDefinition{}, fmt.Errorf("repository: %s must return error", method.Name)
		}
		result.Methods = append(result.Methods, method)
	}
	if len(result.Methods) == 0 {
		return RepositoryDefinition{}, fmt.Errorf("repository: no @Query methods for %s", result.Name)
	}
	return result, nil
}

// parseRepositoryBinding accepts @Repository(Entity) and
// @Repository(Entity, "connection"). The connection name is static at compile time.
func parseRepositoryBinding(argument string) (string, string, error) {
	parts := strings.Split(argument, ",")
	if len(parts) < 1 || len(parts) > 2 {
		return "", "", fmt.Errorf("repository: expected entity and optional connection name")
	}
	entity := strings.TrimSpace(parts[0])
	if !schemaIdentifier(entity) {
		return "", "", fmt.Errorf("repository: invalid entity name")
	}
	if len(parts) == 1 {
		return entity, "", nil
	}
	name, err := strconv.Unquote(strings.TrimSpace(parts[1]))
	if err != nil || !schemaIdentifier(name) {
		return "", "", fmt.Errorf("repository: connection must be a quoted identifier")
	}
	return entity, name, nil
}
