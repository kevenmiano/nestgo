package compiler

import (
	"errors"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

// ErrOutdated indicates that at least one generated artifact differs from the
// current compiler output or has not yet been generated.
var ErrOutdated = errors.New("generated artifacts are out of date")

// CheckDir validates annotated sources and compares the deterministic,
// compiler-owned artifacts without creating or modifying files.
// Developer-editable TODO test stubs are deliberately not checked for equality.
func CheckDir(dir string) error {
	return compileDirWithDialectPlanMode(dir, sqlgen.PostgreSQL, nil, true)
}
