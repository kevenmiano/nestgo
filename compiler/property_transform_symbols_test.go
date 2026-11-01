package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A transformer implemented in an ordinary .go file is resolved as a local
// named struct without requiring struct literals in .mgo annotations.
func TestNamedStructTransformerAcrossGoSourceFiles(t *testing.T) {
	dir := t.TempDir()
	dto := "package dto\ntype User struct {\n@TransformInput(TrimName)\nName string\n}\n"
	implementation := "package dto\n\nimport \"strings\"\n\ntype TrimName struct{}\nfunc (*TrimName) Transform(value string) (string, error) { return strings.TrimSpace(value), nil }\n"
	for name, contents := range map[string]string{"dto.mgo": dto, "transformer.go": implementation} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	gen, err := os.ReadFile(filepath.Join(dir, "dto_transformers_generated.go"))
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(gen), "new(TrimName)") || !strings.Contains(string(gen), "Transformer[string]") {
		t.Fatalf("cross-file type was not statically resolved: %s", gen)
	}
	root, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	mod := "module dto\n\ngo 1.23\n\nrequire github.com/kevenmiano/nestgo v0.0.0\nreplace github.com/kevenmiano/nestgo => " + filepath.ToSlash(filepath.Dir(root)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil { t.Fatal(err) }
	testSource := "package dto\nimport (\"encoding/json\"; \"testing\")\nfunc TestTrimName(t *testing.T) { var user User; if err := json.Unmarshal([]byte(`{\"Name\":\"  Ada  \"}`), &user); err != nil { t.Fatal(err) }; if user.Name != \"Ada\" { t.Fatalf(\"got %q\", user.Name) } }\n"
	if err := os.WriteFile(filepath.Join(dir, "dto_test.go"), []byte(testSource), 0644); err != nil { t.Fatal(err) }
	cmd := exec.Command("go", "test", "-mod=mod", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil { t.Fatalf("cross-file generated DTO failed: %v\n%s", err, output) }
}
