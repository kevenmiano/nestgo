package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompiledCrossPackageModulesBootstrap(t *testing.T) {
	root := t.TempDir()
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	module := "module example.com/nestgo-cross-package-test\n\ngo 1.23.0\n\nrequire github.com/kevenmiano/nestgo v0.0.0\n\nreplace github.com/kevenmiano/nestgo => " + filepath.ToSlash(repo) + "\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(root, "external")
	app := filepath.Join(root, "app")
	for _, dir := range []string{external, app} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(external, "module.mgo"): `package external

@Injectable()
type Repository struct{}

type Greeter interface { Greet() string }
type Unused interface { Unused() string }

@Module({providers:[Repository],exports:[Repository]})
type RepositoryModule struct{}
`,
		filepath.Join(app, "module.mgo"): `package app

import "example.com/nestgo-cross-package-test/external"

@Injectable()
type GreeterService struct{}

func (g *GreeterService) Greet() string { return "from-external-interface" }

@Controller("/ready")
type AppController struct {
    Greeter external.Greeter
    Optional external.Unused NESTGO_OPTIONAL_TAG
}

@Get("/")
func (c *AppController) Ready() string { return c.Greeter.Greet() }

@Module({imports:[external.RepositoryModule],providers:[GreeterService],controllers:[AppController]})
type AppModule struct{}
`,
		filepath.Join(app, "bootstrap_test.go"): `package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	_ "example.com/nestgo-cross-package-test/external"
	"github.com/kevenmiano/nestgo/runtime"
)

func TestCompiledCrossPackageBootstrap(t *testing.T) {
	application, err := runtime.NewApp(context.Background(), runtime.WithRegistry("app"))
	if err != nil { t.Fatal(err) }
	defer application.Close()
	h := application.HTTPHandler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready/", nil))
	if w.Code != 200 || w.Body.String() != "\"from-external-interface\"\n" { t.Fatalf("status %d, body %q", w.Code, w.Body.String()) }
	if _, err := runtime.ApplicationFromRegistry(runtime.ApplicationOptions{}, "app"); err != nil { t.Fatal(err) }
}
`,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, "NESTGO_OPTIONAL_TAG", "`nestgo:\"optional\"`")), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{external, app} {
		if err := CompileDir(dir); err != nil {
			t.Fatalf("compile %s: %v", dir, err)
		}
	}
	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled external modules must build and bootstrap: %v\n%s", err, strings.TrimSpace(string(output)))
	}
}
