package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDtoDecoratorGeneratesOnlyOptedInStubs(t *testing.T) {
	dir := t.TempDir()
	source := `package sample

@Dto()
type CreateUserDTO struct {
	Name string ` + "`json:\"name\" validate:\"required\"`" + `
	Age int ` + "`json:\"age\"`" + `
}

type InternalModel struct {
	Code string ` + "`json:\"code\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "dto.mgo"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "zz_nestgo_stubs_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TestCreateUserDTO_Serialization", "TestCreateUserDTO_Deserialization", "TestCreateUserDTO_Validation", "Name validate:\"required\""} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing DTO scaffold hint %q:\n%s", want, raw)
		}
	}
	if strings.Contains(string(raw), "InternalModel") {
		t.Fatalf("generated stubs for a plain struct: %s", raw)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(dir, "zz_nestgo_stubs_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(again) {
		t.Fatal("DTO stub generation was not idempotent")
	}
}

func TestDtoWithoutValidationContractDoesNotInventValidationTests(t *testing.T) {
	dir := t.TempDir()
	source := "package sample\n@Dto()\ntype PlainDTO struct { Name string }\n"
	if err := os.WriteFile(filepath.Join(dir, "dto.mgo"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "zz_nestgo_stubs_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "TestPlainDTO_Validation") {
		t.Fatal("generated validation test with no declared validation contract")
	}
	for _, want := range []string{"TestPlainDTO_Serialization", "TestPlainDTO_Deserialization"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("missing DTO test %s", want)
		}
	}
}

func TestDtoRejectsInvalidDecoratorArguments(t *testing.T) {
	for _, source := range []string{
		"package sample\n@Dto(\"x\")\ntype A struct{}\n",
		"package sample\n@Dto()\n@Dto()\ntype A struct{}\n",
		"package sample\n@Dto()\nfunc Foo() {}\n",
	} {
		parsed, err := Parse("bad.mgo", []byte(source))
		if err == nil {
			_, err = Generate([]*File{parsed})
		}
		if err == nil {
			t.Fatalf("accepted invalid DTO annotation: %s", source)
		}
	}
}
