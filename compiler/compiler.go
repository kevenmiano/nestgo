package compiler

import (
    "fmt"
    "go/format"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

// CompileFile transforms a limited subset of .mgo decorator syntax into Go.
// This is an experimental implementation, not a complete NestGo compiler.
func CompileFile(path string) (string, error) {
    input, err := os.ReadFile(path)
    if err != nil { return "", err }
    if filepath.Ext(path) != ".mgo" { return "", fmt.Errorf("expected .mgo file: %s", path) }
    decorator := regexp.MustCompile(`(?m)^[ \t]*@(Controller|Injectable|Module|Get|Post|Put|Patch|Delete|HttpCode)\([^\n]*\)[ \t]*$`)
    source := decorator.ReplaceAllString(string(input), "")
    parameter := regexp.MustCompile(`@(Body|Param|Query)\([^()]*\)[ \t]*`)
    source = parameter.ReplaceAllString(source, "")
    formatted, err := format.Source([]byte(source))
    if err != nil { return "", fmt.Errorf("invalid generated Go: %w", err) }
    if _, err := parser.ParseFile(token.NewFileSet(), path, formatted, parser.AllErrors); err != nil {
        return "", err
    }
    target := strings.TrimSuffix(path, ".mgo") + "_generated.go"
    if err := os.WriteFile(target, formatted, 0644); err != nil { return "", err }
    return target, nil
}
