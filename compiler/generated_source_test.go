package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileDirQualifiedModuleDecoratorOnlyImport(t *testing.T) {
	dir := t.TempDir()
	src := "package demo\nimport tm \"time\"\n@Module({imports:[tm.Time]})\ntype AppModule struct{}\n"
	if err := os.WriteFile(filepath.Join(dir, "app.mgo"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join(dir, "app_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "\"time\"") {
		t.Fatalf("decorator-only import leaked into generated source: %s", generated)
	}
	registry, err := os.ReadFile(filepath.Join(dir, "zz_modforge_metadata.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(registry), "tm \"time\"") || !strings.Contains(string(registry), "(*tm.Time)(nil)") {
		t.Fatalf("missing qualified metadata reference: %s", registry)
	}
}

func TestGeneratedSourceRetainsUsedImport(t *testing.T) {
	src := "package demo\nimport tm \"time\"\n@Module({imports:[tm.Time]})\ntype AppModule struct { Now tm.Time }\n"
	file, err := Parse("app.mgo", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generatedSource(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\"time\"") {
		t.Fatalf("import still used in application type was removed: %s", out)
	}
}
