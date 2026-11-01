package compiler

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// externalProviderExported validates the exact imported module when its .mgo
// source is available. Published binary-only modules remain runtime-validated.
func externalProviderExported(sourcePath, qualifier, provider string, imports []string) (bool, error) {
	var selected []string
	for _, module := range imports {
		if strings.HasPrefix(module, qualifier+".") {
			selected = append(selected, strings.TrimPrefix(module, qualifier+"."))
		}
	}
	if len(selected) == 0 {
		return false, nil
	}
	var importPath string
	fileSource, err := os.ReadFile(sourcePath)
	if err != nil {
		return false, err
	}
	node, err := parser.ParseFile(token.NewFileSet(), sourcePath, fileSource, parser.ImportsOnly)
	if err != nil {
		return false, err
	}
	for _, imp := range node.Imports {
		uri, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return false, err
		}
		alias := path.Base(uri)
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		if alias == qualifier {
			importPath = uri
			break
		}
	}
	if importPath == "" {
		return false, fmt.Errorf("missing Go import for %s", qualifier)
	}
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", importPath)
	cmd.Dir = filepath.Dir(sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("resolve module source %s: %w: %s", importPath, err, strings.TrimSpace(string(output)))
	}
	dir := strings.TrimSpace(string(output))
	paths, err := filepath.Glob(filepath.Join(dir, "*.mgo"))
	if err != nil {
		return false, err
	}
	if len(paths) == 0 {
		return true, nil // no source contract: runtime remains fail-closed
	}
	for _, filename := range paths {
		raw, err := os.ReadFile(filename)
		if err != nil {
			return false, err
		}
		file, err := Parse(filename, raw)
		if err != nil {
			return false, err
		}
		for _, decl := range file.Declarations {
			for _, moduleName := range selected {
				if decl.Kind != "type" || decl.Name != moduleName {
					continue
				}
				for _, ann := range decl.Annotations {
					if ann.Name != "Module" {
						continue
					}
					options, err := parseModuleOptions(filename, ann, decl.Name)
					if err != nil {
						return false, err
					}
					for _, exported := range options.Exports {
						if exported == provider && containsModuleType(options.Providers, provider) {
							return true, nil
						}
					}
				}
			}
		}
	}
	return false, nil
}
