package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedPipelineInterfaceAssertions(t *testing.T) {
	source := []byte(`package sample
@Injectable()
type Auth struct{}
@Injectable()
type Trim struct{}
@Injectable()
type Filter struct{}
@UseGuards(Auth)
@UsePipes([Trim])
@Controller("/")
type C struct{}
@UseGuards(Auth)
@UseFilters(Filter)
@Get("/")
func (*C) Get() string { return "ok" }
`)
	file, err := Parse("sample.mgo", source)
	if err != nil {
		t.Fatal(err)
	}
	output, err := Generate([]*File{file})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"var _ httpadapter.Guard = (*Auth)(nil)",
		"var _ httpadapter.Pipe = (*Trim)(nil)",
		"var _ httpadapter.ExceptionFilter = (*Filter)(nil)",
	} {
		if count := strings.Count(string(output), line); count != 1 {
			t.Fatalf("assertion %q generated %d times; expected one:\n%s", line, count, output)
		}
	}
}

func TestPipelineInterfacesRejectedByGoCompiler(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.ToSlash(filepath.Dir(cwd))
	cases := []struct {
		name, decorator, method, want string
	}{
		{"guard", "UseGuards", "func (*Bad) CanActivate(httpadapter.ExecutionContext) bool { return true }", "httpadapter.Guard"},
		{"pipe", "UsePipes", "func (*Bad) Transform(any, httpadapter.ArgumentMetadata) any { return nil }", "httpadapter.Pipe"},
		{"filter", "UseFilters", "func (*Bad) Catch(error, httpadapter.ExecutionContext) bool { return false }", "httpadapter.ExceptionFilter"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := "package testpipeline\nimport \"github.com/kevenmiano/nestgo/httpadapter\"\n" +
				"@Injectable()\ntype Bad struct{}\n" + tc.method + "\n" +
				"@" + tc.decorator + "(Bad)\n@Controller(\"/\")\ntype TestController struct{}\n" +
				"@Get(\"/\")\nfunc (*TestController) Get() string { return \"ok\" }\n" +
				"@Module({controllers: [TestController], providers: [Bad]})\ntype RootModule struct{}\n"
			if err := os.WriteFile(filepath.Join(dir, "test.mgo"), []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			if err := CompileDir(dir); err != nil {
				t.Fatalf("generate: %v", err)
			}
			mod := "module testpipeline\n\ngo 1.27.0\n\nrequire github.com/kevenmiano/nestgo v0.0.0\nreplace github.com/kevenmiano/nestgo => " + root + "\n"
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "test", "-mod=mod", ".")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOWORK=off")
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("incompatible %s provider compiled successfully:\n%s", tc.name, output)
			}
			if !strings.Contains(string(output), tc.want) || !strings.Contains(string(output), "does not implement") {
				t.Fatalf("want a static interface failure for %s; got:\n%s", tc.name, output)
			}
		})
	}
}
