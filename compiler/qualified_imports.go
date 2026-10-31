package compiler

import (
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

func qualifiedModuleImportPaths(files []*File, modules map[string]moduleOptions) ([]string, error) {
	required := make(map[string]bool)
	for _, opts := range modules {
		for _, name := range opts.Imports {
			if dot := strings.IndexByte(name, '.'); dot > 0 {
				required[name[:dot]] = true
			}
		}
	}
	if len(required) == 0 {
		return nil, nil
	}
	aliases := make(map[string]string)
	for _, file := range files {
		node, err := parser.ParseFile(token.NewFileSet(), file.Path, file.Source, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, imp := range node.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, err
			}
			alias := path.Base(importPath)
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			if !required[alias] {
				continue
			}
			if previous, ok := aliases[alias]; ok && previous != importPath {
				return nil, fmt.Errorf("conflicting qualified module import alias %s: %s and %s", alias, previous, importPath)
			}
			aliases[alias] = importPath
		}
	}
	list := make([]string, 0, len(required))
	for alias := range required {
		importPath, ok := aliases[alias]
		if !ok {
			return nil, fmt.Errorf("qualified module import %s is not declared by a Go import", alias)
		}
		list = append(list, fmt.Sprintf("%s %q", alias, importPath))
	}
	sortStrings(list)
	return list, nil
}
