package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/scanner"

	"github.com/kevenmiano/nestgo/compiler"
	"github.com/spf13/cobra"
)

const diagnosticSchemaVersion = 1

// DiagnosticEntry is the stable JSON representation of a compiler problem.
// Line and column use one-based Go source coordinates when available.
type DiagnosticEntry struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

// DiagnosticReport is emitted once per single-shot compile/check invocation.
type DiagnosticReport struct {
	SchemaVersion int               `json:"schema_version"`
	Command       string            `json:"command"`
	OK            bool              `json:"ok"`
	Diagnostics   []DiagnosticEntry `json:"diagnostics"`
}

// ReportedError tells the executable that a JSON failure has already been
// printed to stdout, so it must not add an unstructured error to stderr.
type ReportedError struct{ Cause error }

func (e *ReportedError) Error() string { return e.Cause.Error() }
func (e *ReportedError) Unwrap() error { return e.Cause }

// IsReported reports whether the CLI has already emitted an error envelope.
func IsReported(err error) bool {
	var reported *ReportedError
	return errors.As(err, &reported)
}

func diagnosticEntries(err error) []DiagnosticEntry {
	if err == nil {
		return []DiagnosticEntry{}
	}
	var outdated *compiler.OutdatedError
	if errors.As(err, &outdated) {
		items := make([]DiagnosticEntry, 0, len(outdated.Artifacts))
		for _, artifact := range outdated.Artifacts {
			code := "NG2002"
			switch artifact.Reason {
			case compiler.ArtifactMissing:
				code = "NG2001"
			case compiler.ArtifactObsolete:
				code = "NG2003"
			}
			items = append(items, DiagnosticEntry{
				Code: code, Severity: "error", Message: "generated artifact " + string(artifact.Reason), File: artifact.Path,
			})
		}
		return items
	}
	var source compiler.Diagnostic
	if errors.As(err, &source) {
		return []DiagnosticEntry{{Code: "NG1001", Severity: "error", Message: source.Message, File: source.File, Line: source.Line}}
	}
	var parseErrors scanner.ErrorList
	if errors.As(err, &parseErrors) && len(parseErrors) > 0 {
		items := make([]DiagnosticEntry, 0, len(parseErrors))
		for _, syntax := range parseErrors {
			items = append(items, DiagnosticEntry{
				Code: "NG1002", Severity: "error", Message: syntax.Msg,
				File: syntax.Pos.Filename, Line: syntax.Pos.Line, Column: syntax.Pos.Column,
			})
		}
		return items
	}
	return []DiagnosticEntry{{Code: "NG9000", Severity: "error", Message: err.Error()}}
}

func finishCompilerCommand(cmd *cobra.Command, format, command, successText string, err error) error {
	switch format {
	case "text":
		if err != nil {
			return err
		}
		cmd.Println(successText)
		return nil
	case "json":
		report := DiagnosticReport{
			SchemaVersion: diagnosticSchemaVersion,
			Command:       command,
			OK:            err == nil,
			Diagnostics:   diagnosticEntries(err),
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetEscapeHTML(false)
		if encodeErr := encoder.Encode(report); encodeErr != nil {
			return fmt.Errorf("write JSON diagnostics: %w", encodeErr)
		}
		if err != nil {
			return &ReportedError{Cause: err}
		}
		return nil
	default:
		return fmt.Errorf("unsupported --format %q (expected text or json)", format)
	}
}
