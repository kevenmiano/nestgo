package compiler

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

// ErrOutdated indicates that at least one compiler-owned generated artifact
// differs from its expected contents, is missing, or is no longer needed.
var ErrOutdated = errors.New("generated artifacts are out of date")

// ArtifactReason identifies why a compiler-owned artifact failed freshness checks.
type ArtifactReason string

const (
	ArtifactMissing  ArtifactReason = "missing"
	ArtifactOutdated ArtifactReason = "outdated"
	ArtifactObsolete ArtifactReason = "obsolete"
)

// OutdatedArtifact is a single, machine-readable compiler output discrepancy.
type OutdatedArtifact struct {
	Path   string         `json:"path"`
	Reason ArtifactReason `json:"reason"`
}

// OutdatedError exposes all discrepancies while preserving errors.Is(ErrOutdated).
type OutdatedError struct {
	Directory string
	Artifacts []OutdatedArtifact
}

func (e *OutdatedError) Unwrap() error { return ErrOutdated }

func (e *OutdatedError) Error() string {
	issues := make([]string, 0, len(e.Artifacts))
	for _, item := range e.Artifacts {
		issues = append(issues, fmt.Sprintf("%s (%s)", item.Path, item.Reason))
	}
	return fmt.Sprintf("%s: %s; regenerate with nestgo compile %s", ErrOutdated, strings.Join(issues, ", "), e.Directory)
}

func sortedOutdatedArtifacts(artifacts []OutdatedArtifact) []OutdatedArtifact {
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Path == artifacts[j].Path {
			return artifacts[i].Reason < artifacts[j].Reason
		}
		return artifacts[i].Path < artifacts[j].Path
	})
	return artifacts
}

// CheckDir validates annotated sources and compares the deterministic,
// compiler-owned artifacts without creating or modifying files.
// Developer-editable TODO test stubs are deliberately not checked for equality.
func CheckDir(dir string) error {
	return compileDirWithDialectPlanMode(dir, sqlgen.PostgreSQL, nil, true)
}
