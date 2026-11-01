package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const dtoSource = `package demodto

import (
    "net/url"
    "strings"
)

type Address struct { City string ` + "\u0060json:\"city\"\u0060" + ` }

type User struct {
    @Expose("display_name")
    Name string ` + "\u0060json:\"name,omitempty\"\u0060" + `
    @Exclude()
    Password string ` + "\u0060json:\"password\"\u0060" + `
    @TransformInput(NormalizeEmail)
    Email string ` + "\u0060json:\"email\"\u0060" + `
    @TransformInput(url.PathUnescape)
    Slug string ` + "\u0060json:\"slug\"\u0060" + `
    @TransformOutput(Uppercase)
    @TransformOutput(&AddSuffix{})
    Label string ` + "\u0060json:\"label\"\u0060" + `
    @ExcludeOutput()
    Token string ` + "\u0060json:\"token\"\u0060" + `
    @ExcludeInput()
    ServerID string ` + "\u0060json:\"server_id\"\u0060" + `
    @TransformInput(TrimTenant{})
    Tenant string ` + "\u0060json:\"tenant\"\u0060" + `
    @Type(Address)
    Address Address ` + "\u0060json:\"address\"\u0060" + `
    Age int ` + "\u0060json:\"age,omitempty\"\u0060" + `
}

func NormalizeEmail(value string) (string,error) { return strings.ToLower(value),nil }
func Uppercase(value string) (string,error) { return strings.ToUpper(value),nil }
type TrimTenant struct{}
func (TrimTenant) Transform(value string) (string,error) { return strings.TrimSpace(value),nil }
type AddSuffix struct{}
func (*AddSuffix) Transform(value string) (string,error) { return value+"!",nil }
`

func TestPropertyDecoratorsCompileAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dto.mgo"), []byte(dtoSource), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	codec, err := os.ReadFile(filepath.Join(dir, "dto_transformers_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"MarshalJSON()", "UnmarshalJSON(data []byte)", "ApplyFunc[string](p", "Apply[string](p", "TrimTenant{}", "&AddSuffix{}", "display_name"} {
		if !strings.Contains(string(codec), expected) {
			t.Fatalf("missing %q from generated codec:\n%s", expected, codec)
		}
	}
	normal, err := os.ReadFile(filepath.Join(dir, "dto_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(normal), "\"net/url\"") {
		t.Fatalf("decorator-only import leaked into normal Go source: %s", normal)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Dir(workingDir)
	mod := "module demodto\n\ngo 1.23\n\nrequire github.com/kevenmiano/nestgo v0.0.0\nreplace github.com/kevenmiano/nestgo => " + filepath.ToSlash(moduleRoot) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil {
		t.Fatal(err)
	}
	testSource := `package demodto

import (
    "encoding/json"
    "strings"
    "testing"
)

func TestCodec(t *testing.T) {
    var u User
    if err := json.Unmarshal([]byte("{\"display_name\":\"Ada\",\"password\":\"secret\",\"email\":\"ADA@EXAMPLE.COM\",\"slug\":\"%2Fapi\",\"token\":\"inbound\",\"server_id\":\"user-controlled\",\"label\":\"hello\",\"tenant\":\"  premium  \",\"address\":{\"city\":\"London\"}}"), &u); err != nil {t.Fatal(err)}
    if u.Name!="Ada"||u.Password!=""||u.Email!="ada@example.com"||u.Slug!="/api"||u.Token!="inbound"||u.ServerID!=""||u.Label!="hello"||u.Tenant!="premium"||u.Address.City!="London" {t.Fatalf("decoded: %+v",u)}
    u.Password="private"
    u.ServerID="server-set"
    data,err:=json.Marshal(u)
    if err != nil {t.Fatal(err)}
    if strings.Contains(string(data),"private")||strings.Contains(string(data),"password") {t.Fatalf("password leaked: %s",data)}
    if !strings.Contains(string(data),"\"display_name\":\"Ada\"") {t.Fatalf("alias missing: %s",data)}
    if strings.Contains(string(data), "inbound") || !strings.Contains(string(data), "\"server_id\":\"server-set\"") || !strings.Contains(string(data), "\"label\":\"HELLO!\"") {t.Fatalf("output transformations/exclusions failed: %s", data)}
}
`
	if err := os.WriteFile(filepath.Join(dir, "dto_test.go"), []byte(testSource), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-mod=mod", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated DTO does not build or pass tests: %v\n%s", err, out)
	}
}

func TestPropertyDecoratorDiagnostics(t *testing.T) {
	tests := []struct{ name, source, errText string }{
		{"outside-struct", "package demo\n@Exclude()\ntype User struct{}", "property decorator"},
		{"missing-field", "package demo\ntype User struct {\n@Expose()\n}", "must precede"},
		{"bad-transform", "package demo\ntype User struct {\n@Transform(something())\nName string\n}", "function or struct instance"},
		{"conflict", "package demo\ntype User struct {\n@Expose()\n@Exclude()\nName string\n}", "cannot decorate"},
		{"type-mismatch", "package demo\ntype User struct {\n@Type(int)\nName string\n}", "must match"},
		{"old-direction-removed", "package demo\ntype User struct {\n@Transform(Fn, \"toPlainOnly\")\nName string\n}", "function or struct instance"},
		{"exclude-direction-removed", "package demo\ntype User struct {\n@Exclude(\"toClassOnly\")\nName string\n}", "takes no arguments"},
		{"duplicate-exclude-input", "package demo\ntype User struct {\n@ExcludeInput()\n@ExcludeInput()\nName string\n}", "duplicate @ExcludeInput"},
		{"conflicting-exclusion", "package demo\ntype User struct {\n@Exclude()\n@ExcludeOutput()\nName string\n}", "cannot be combined"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("dto.mgo", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.errText) {
				t.Fatalf("expected %q, got %v", tc.errText, err)
			}
		})
	}
}
