package compiler

import (
	"strings"
	"testing"
)

func TestGenerateTypedHTTPDispatcher(t *testing.T) {
	source := []byte(`package sample
@Controller("/users")
type Users struct{}

@Get("/search")
func (c *Users) Search(@Query("q") q string) string { return q }

@Get("/{id}")
func (c *Users) Get(@Param("id") id string) (string, error) { return id, nil }

@Post("/")
func (c *Users) Create(@Body() dto CreateDTO) string { return dto.Name }

type CreateDTO struct { Name string }
`)
	file, err := Parse("users.mgo", source)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := Generate([]*File{file})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"github.com/kevenmiano/nestgo/httpadapter",
		"StaticHTTP:",
		"httpadapter.BindQuery(ctx, \"q\", 0)",
		"httpadapter.BindParam(ctx, \"id\", 0)",
		"httpadapter.DecodeBodyAt[CreateDTO](ctx, 0)",
		"return value, 0, err",
	} {
		if !strings.Contains(string(generated), expected) {
			t.Errorf("generated dispatcher missing %q:\n%s", expected, generated)
		}
	}
}

func TestGenerateFallbackForUnsupportedParameters(t *testing.T) {
	source := []byte(`package sample
@Controller("/")
type C struct{}
@Get("/")
func (c *C) Find(id string) string { return id }
`)
	file, err := Parse("users.mgo", source)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := Generate([]*File{file})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "StaticHTTP:") {
		t.Fatal("undecorated parameter should keep reflection fallback")
	}
}
