package compiler

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestDecoratorDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		source string
		message string
		line int
	}{
		{"unknown", "package demo\n@Unknown()\ntype A struct{}\n", "unknown decorator", 2},
		{"dangling", "package demo\n@Controller(\"/\")\n", "dangling decorator", 2},
		{"unclosed", "package demo\n@Controller(\"/\"\ntype A struct{}\n", "unclosed decorator argument", 2},
		{"inline", "package demo\n@Controller(\"/\") type A struct{}\n", "declaration decorator must be on its own line", 2},
		{"invalid parameter", "package demo\n@Controller(\"/\")\ntype A struct{}\n@Post(\"/\")\nfunc (a *A) Create(@Body(unquoted) dto string) {}\n", "parameter decorator expects quoted string", 5},
		{"route on type", "package demo\n@Get(\"/\")\ntype A struct{}\n", "cannot decorate a type", 2},
		{"controller on function", "package demo\n@Controller(\"/\")\ntype A struct{}\n@Controller(\"/\")\nfunc (a *A) List() {}\n", "cannot decorate a function", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("fixture.mgo", []byte(tt.source))
			if err == nil { t.Fatal("expected diagnostic") }
			var d Diagnostic
			if !errors.As(err, &d) {
				t.Fatalf("expected Diagnostic, got %T: %v", err, err)
			}
			if d.File != "fixture.mgo" || d.Line != tt.line || !strings.Contains(d.Message, tt.message) {
				t.Fatalf("unexpected diagnostic: %+v", d)
			}
		})
	}
}

func TestGenerationValidation(t *testing.T) {
	tests := []struct {
		name string
		source string
		message string
	}{
		{"orphan route", "package demo\ntype A struct{}\n@Get(\"/\")\nfunc (a *A) List() {}\n", "not decorated with @Controller"},
		{"duplicate route", "package demo\n@Controller(\"/\")\ntype A struct{}\n@Get(\"/\")\n@Post(\"/\")\nfunc (a *A) List() {}\n", "multiple HTTP method decorators"},
		{"invalid http code", "package demo\n@Controller(\"/\")\ntype A struct{}\n@Get(\"/\")\n@HttpCode(700)\nfunc (a *A) List() {}\n", "invalid HTTP status"},
		{"invalid controller argument", "package demo\n@Controller(unquoted)\ntype A struct{}\n", "expects quoted string"},
		{"invalid module argument", "package demo\n@Module(unquoted)\ntype A struct{}\n", "expects quoted string"},
		{"route on free function", "package demo\n@Get(\"/\")\nfunc List() {}\n", "requires local controller method"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := Parse("fixture.mgo", []byte(tt.source))
			if err == nil {
				_, err = Generate([]*File{file})
			}
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("wanted %q, got %v", tt.message, err)
			}
		})
	}
}

func TestGenerateMultiFileDeterministic(t *testing.T) {
	first, err := Parse("a.mgo", []byte("package demo\n@Controller(\"/alpha\")\ntype Alpha struct{}\n@Get(\"/\")\nfunc (a *Alpha) List() string { return \"ok\" }\n"))
	if err != nil { t.Fatal(err) }
	second, err := Parse("b.mgo", []byte("package demo\n@Injectable()\ntype Provider struct{}\n@Module(\"app\")\ntype App struct{}\n"))
	if err != nil { t.Fatal(err) }
	out, err := Generate([]*File{first, second})
	if err != nil { t.Fatal(err) }
	repeated, err := Generate([]*File{first, second})
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(out, repeated) { t.Fatal("generated output is not deterministic") }
	for _, expected := range []string{"Alpha", "Provider", "App", "/alpha", "List"} {
		if !bytes.Contains(out, []byte(expected)) { t.Errorf("missing %q in output", expected) }
	}
	other, err := Parse("other.mgo", []byte("package other\ntype Other struct{}\n"))
	if err != nil { t.Fatal(err) }
	if _, err := Generate([]*File{first, other}); err == nil || !strings.Contains(err.Error(), "mixed packages") {
		t.Fatalf("expected mixed-package error, got %v", err)
	}
}

func TestParameterDecorators(t *testing.T) {
	src := "package demo\n@Controller(\"/\")\ntype A struct{}\n@Post(\"/\")\nfunc (a *A) Create(@Param(\"id\") id string, @Query(\"q\") q string, @Headers(\"x-test\") header string, @Body() body string) {}\n"
	file, err := Parse("params.mgo", []byte(src))
	if err != nil { t.Fatal(err) }
	if len(file.Declarations) != 2 { t.Fatalf("unexpected declarations: %+v", file.Declarations) }
	params := file.Declarations[1].Params
	if len(params) != 4 { t.Fatalf("expected four parameter decorators, got %+v", params) }
	kinds := []string{"Param", "Query", "Headers", "Body"}
	names := []string{"id", "q", "x-test", ""}
	for i, param := range params {
		if param.Index != i || param.Kind != kinds[i] || param.Name != names[i] {
			t.Errorf("parameter %d: %+v", i, param)
		}
	}
}
