package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/compiler"
)

func runJSONCommand(t *testing.T, args ...string) (DiagnosticReport, error) {
	t.Helper()
	output, err := executeForTest(args...)
	var report DiagnosticReport
	if decodeErr := json.Unmarshal([]byte(output), &report); decodeErr != nil {
		t.Fatalf("invalid JSON output for %v: %v (%q)", args, decodeErr, output)
	}
	if report.SchemaVersion != 1 {
		t.Fatalf("unsupported schema version: %+v", report)
	}
	if report.Diagnostics == nil {
		t.Fatalf("diagnostics must be an array, not null: %q", output)
	}
	return report, err
}

func TestJSONDiagnosticsForMissingAndOutdatedOutputs(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "provider.mgo")
	if err := os.WriteFile(source, []byte("package sample\n@Injectable()\ntype Provider struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	report, err := runJSONCommand(t, "check", "--format=json", dir)
	if !errors.Is(err, compiler.ErrOutdated) || !IsReported(err) || report.OK {
		t.Fatalf("missing output not reported: %+v, %v", report, err)
	}
	if report.Command != "check" || len(report.Diagnostics) < 2 {
		t.Fatalf("wrong check report: %+v", report)
	}
	for i, diagnostic := range report.Diagnostics {
		if diagnostic.Code != "NG2001" || diagnostic.Severity != "error" || !strings.HasSuffix(diagnostic.Message, "missing") {
			t.Fatalf("invalid missing artifact at %d: %+v", i, diagnostic)
		}
		if i > 0 && report.Diagnostics[i-1].File > diagnostic.File {
			t.Fatalf("diagnostics not sorted by path: %+v", report.Diagnostics)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("read-only check created files: %v %v", entries, err)
	}
	report, err = runJSONCommand(t, "compile", "--format", "json", dir)
	if err != nil || !report.OK || report.Command != "compile" || len(report.Diagnostics) != 0 {
		t.Fatalf("compile JSON success: %+v, %v", report, err)
	}
	report, err = runJSONCommand(t, "check", "--format=json", source)
	if err != nil || !report.OK || len(report.Diagnostics) != 0 {
		t.Fatalf("check JSON success: %+v, %v", report, err)
	}
	metadata := filepath.Join(dir, "zz_modforge_metadata.go")
	if err := os.WriteFile(metadata, []byte("package sample\n// outdated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	report, err = runJSONCommand(t, "check", "--format=json", dir)
	if !errors.Is(err, compiler.ErrOutdated) || len(report.Diagnostics) != 1 ||
		report.Diagnostics[0].Code != "NG2002" || report.Diagnostics[0].File != metadata {
		t.Fatalf("outdated artifact should have stable code: %+v %v", report, err)
	}
}

func TestJSONDiagnosticsForDecoratorsAndSyntax(t *testing.T) {
	for _, tc := range []struct {
		source, want string
		line         int
	}{
		{"package sample\n@NoSuchDecorator()\ntype Foo struct{}\n", "NG1001", 2},
		{"package sample\n@Injectable()\ntype Foo struct {\n", "NG1002", 3},
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.mgo")
		if err := os.WriteFile(path, []byte(tc.source), 0644); err != nil {
			t.Fatal(err)
		}
		report, err := runJSONCommand(t, "compile", "--format=json", dir)
		if err == nil || !IsReported(err) || report.OK || len(report.Diagnostics) == 0 {
			t.Fatalf("missing error: %+v %v", report, err)
		}
		issue := report.Diagnostics[0]
		if issue.Code != tc.want || issue.File != path || issue.Line < tc.line || issue.Message == "" {
			t.Fatalf("missing source position: %+v", issue)
		}
		if tc.want == "NG1002" && issue.Column <= 0 {
			t.Fatalf("syntax position missing column: %+v", issue)
		}
	}
}

func TestJSONCompileDisallowsWatchAndDoesNotStart(t *testing.T) {
	dir := t.TempDir()
	report, err := runJSONCommand(t, "compile", "--format=json", "--watch", dir)
	if err == nil || !IsReported(err) || report.OK || len(report.Diagnostics) != 1 ||
		report.Diagnostics[0].Code != "NG9000" {
		t.Fatalf("watch JSON should be rejected cleanly: %+v %v", report, err)
	}
}

func TestJSONMissingPathAndTextCompatibility(t *testing.T) {
	report, err := runJSONCommand(t, "check", "--format=json", "does-not-exist.mgo")
	if err == nil || !IsReported(err) || report.OK || len(report.Diagnostics) != 1 ||
		report.Diagnostics[0].Code != "NG9000" {
		t.Fatalf("missing path did not yield generic JSON diagnostic: %+v %v", report, err)
	}
	out, err := executeForTest("check", "does-not-exist.mgo")
	if err == nil || IsReported(err) || out != "" {
		t.Fatalf("text mode unexpectedly changed: %q %v", out, err)
	}
}
