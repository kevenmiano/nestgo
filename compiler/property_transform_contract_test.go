package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These failures must be caught by the Go compiler, never by reflection
// or a panic on the HTTP request path.
func TestGeneratedTransformerContractsRejectInvalidImplementations(t *testing.T) {
	tests := []struct {
		name       string
		decorator  string
		definition string
	}{
		{"wrong-function-input", "Wrong", "func Wrong(value int) (int, error) { return value, nil }"},
		{"wrong-function-output", "Wrong", "func Wrong(value string) string { return value }"},
		{"struct-missing-interface", "Wrong", "type Wrong struct{}"},
		{"struct-wrong-field-type", "Wrong", "type Wrong struct{}\nfunc (Wrong) Transform(v int) (int, error) { return v, nil }"},
		{"struct-wrong-return-type", "Wrong", "type Wrong struct{}\nfunc (Wrong) Transform(v string) string { return v }"},
		{"struct-wrong-pointer-method", "Wrong", "type Wrong struct{}\nfunc (*Wrong) Transform(v *string) (string, error) { return *v, nil }"},
	}
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.ToSlash(filepath.Dir(workingDir))
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src := "package contracttest\n\ntype User struct {\n@TransformInput(" + tc.decorator + ")\nName string\n}\n\n" + tc.definition + "\n"
			if err := os.WriteFile(filepath.Join(dir, "dto.mgo"), []byte(src), 0644); err != nil {
				t.Fatal(err)
			}
			if err := CompileDir(dir); err != nil {
				t.Fatalf("valid decorator syntax should generate static Go checks: %v", err)
			}
			mod := "module contracttest\n\ngo 1.23\n\nrequire github.com/kevenmiano/nestgo v0.0.0\nreplace github.com/kevenmiano/nestgo => " + root + "\n"
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "test", "-mod=mod", ".")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOWORK=off")
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("invalid transformer compiled successfully: %s", tc.name)
			}
			if !strings.Contains(string(output), "Wrong") {
				t.Fatalf("expected a compile-time transformer contract error, got:\n%s", output)
			}
		})
	}
}

func TestTransformerStructExpressionValidation(t *testing.T) {
	for _, tc := range []struct {
		name, source string
	}{
		{"constructor-calls", "MakeTransformer()"},
		{"configured-literal", "Trim{Enabled: true}"},
		{"struct-instance", "Trim{}"},
		{"pointer-instance", "&Trim{}"},
		{"anonymous-struct", "struct{}{}"},
		{"nil-instance", "nil"},
		{"function-call", "Trim()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "package dto\ntype User struct {\n@TransformInput(" + tc.source + ")\nName string\n}\n"
			_, err := Parse("bad.mgo", []byte(source))
			if err == nil {
				t.Fatalf("dangerous or ambiguous transformer expression %s was accepted", tc.source)
			}
		})
	}
}
