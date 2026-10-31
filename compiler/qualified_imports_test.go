package compiler

import (
	"strings"
	"testing"
)

func TestQualifiedModuleImportGenerated(t *testing.T) {
	src := "package demo\nimport tm \"time\"\n@Module({imports:[tm.Time]})\ntype AppModule struct { Created tm.Time }\n"
	file, err := Parse("qualified.mgo", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Generate([]*File{file})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"import tm \"time\"", "reflect.TypeOf((*tm.Time)(nil))"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("metadata missing %q: %s", want, out)
		}
	}
}

func TestQualifiedModuleImportRequiresGoAlias(t *testing.T) {
	src := "package demo\n@Module({imports:[external.ServiceModule]})\ntype AppModule struct{}\n"
	file, err := Parse("missing_alias.mgo", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate([]*File{file})
	if err == nil || !strings.Contains(err.Error(), "not declared by a Go import") {
		t.Fatalf("expected missing alias error, got %v", err)
	}
}
