package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalConcreteDependencyRequiresModuleImport(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(root, "external")
	app := filepath.Join(root, "app")
	for _, dir := range []string{external, app} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/nestgo-concrete-negative\n\ngo 1.23.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "provider.go"), []byte("package external\ntype Repository struct{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := `package app
import "example.com/nestgo-concrete-negative/external"

@Controller("/test")
type Controller struct { Repo *external.Repository }

@Get("/")
func (*Controller) Read() string { return "ok" }

@Module({controllers:[Controller]})
type AppModule struct{}
`
	if err := os.WriteFile(filepath.Join(app, "app.mgo"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	err := CompileDir(app)
	if err == nil || !strings.Contains(err.Error(), "missing module import") || !strings.Contains(err.Error(), "Controller.Repo") {
		t.Fatalf("expected source-attributed external concrete DI diagnostic, got %v", err)
	}
	for _, name := range []string{"app_generated.go", "zz_modforge_metadata.go"} {
		if _, statErr := os.Stat(filepath.Join(app, name)); !os.IsNotExist(statErr) {
			t.Fatalf("unexpected generated artifact %s: %v", name, statErr)
		}
	}
}
