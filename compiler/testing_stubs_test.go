package compiler

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const stubFixture = `package example
@Injectable()
type Repo struct{}

@Controller("/users")
type UsersController struct {
	Repo *Repo ` + "`nestgo:\"inject\"`" + `
}

@Get("/{id}")
func (c *UsersController) Find(@Param("id") id string) string { return id }

@Module({providers:[Repo],controllers:[UsersController],exports:[Repo]})
type UsersModule struct{}
`

func TestWiringTestSourceDeterministicAndParseable(t *testing.T) {
	file, err := Parse("users.mgo", []byte(stubFixture))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate([]*File{file}); err != nil {
		t.Fatal(err)
	}
	first, err := wiringTestSource([]*File{file})
	if err != nil {
		t.Fatal(err)
	}
	second, err := wiringTestSource([]*File{file})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("wiring test generation is nondeterministic")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "generated_test.go", first, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TestNestGoCompiledWiring", "UsersController", "Find", "Repo", "UsersModule", "GET", "/{id}"} {
		if !strings.Contains(string(first), want) {
			t.Errorf("generated wiring test is missing %s", want)
		}
	}
}

func TestCompileDirCreatesAndPreservesEditableStubs(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "users.mgo")
	if err := os.WriteFile(source, []byte(stubFixture), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	wiring := filepath.Join(dir, "zz_nestgo_wiring_generated_test.go")
	stubs := filepath.Join(dir, "zz_nestgo_stubs_test.go")
	for _, path := range []string{wiring, stubs} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), path, raw, parser.AllErrors); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadFile(stubs)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TestRepo", "TestUsersController", "TestUsersController_Find"} {
		if !strings.Contains(string(before), "func "+name+"(") {
			t.Errorf("missing unit test stub %s", name)
		}
	}
	if !strings.Contains(string(before), "Repo *Repo") {
		t.Fatal("compiled dependency field missing from stub")
	}
	custom := append(append([]byte(nil), before...), []byte("\n// Developer custom assertion preserved.\n")...)
	if err := os.WriteFile(stubs, custom, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	recompiled, err := os.ReadFile(stubs)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(recompiled, custom) {
		t.Fatal("recompilation changed developer-owned test stub")
	}
	changed := stubFixture + "\n@Post(\"/\")\nfunc (c *UsersController) Create() string { return \"ok\" }\n"
	if err := os.WriteFile(source, []byte(changed), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(stubs)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(after, custom) || !strings.Contains(string(after), "func TestUsersController_Create(") {
		t.Fatalf("new route stub missing or developer edits lost: %s", after)
	}
	if strings.Count(string(after), "func TestUsersController_Find(") != 1 {
		t.Fatal("existing test stub duplicated")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), stubs, after, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
}

func TestCompileDirDoesNotDuplicateUserTest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "users.mgo"), []byte(stubFixture), 0644); err != nil {
		t.Fatal(err)
	}
	manual := "package example\nimport \"testing\"\nfunc TestUsersController_Find(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(dir, "manual_test.go"), []byte(manual), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	stubs, err := os.ReadFile(filepath.Join(dir, "zz_nestgo_stubs_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stubs), "func TestUsersController_Find(") {
		t.Fatal("generator duplicated an existing developer test")
	}
}

func TestCompileDirRejectsInvalidUserTestBeforeAppend(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "users.mgo"), []byte(stubFixture), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken_test.go"), []byte("package example\nfunc TestBroken(\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err == nil {
		t.Fatal("expected malformed existing test to be reported")
	}
}

func TestCompilerSourceGofmt(t *testing.T) {
	output, err := exec.Command("gofmt", "-d", "testing_stubs.go").CombinedOutput()
	if err != nil || len(output) != 0 {
		t.Fatalf("compiler source gofmt diff:\n%s (err=%v)", output, err)
	}
}
