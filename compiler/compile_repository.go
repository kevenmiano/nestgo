package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

// CompileRepositoryFile emits a typed implementation from a single .mgo file.
// Unlike CompileDir, it does not register providers or generate metadata yet.
func CompileRepositoryFile(sourceFile string, dialect sqlgen.Dialect) (string, error) {
	if filepath.Ext(sourceFile) != ".mgo" {
		return "", fmt.Errorf("repository: expected .mgo source")
	}
	raw, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", err
	}
	file, err := Parse(sourceFile, raw)
	if err != nil {
		return "", err
	}
	def, err := RepositoryFromMGO(file, dialect)
	if err != nil {
		return "", err
	}
	generated, err := GenerateRepository(def)
	if err != nil {
		return "", err
	}
	target := strings.TrimSuffix(sourceFile, ".mgo") + "_repository_generated.go"
	if err := writeIfChanged(target, generated); err != nil {
		return "", err
	}
	return target, nil
}
