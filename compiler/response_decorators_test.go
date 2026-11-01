package compiler

import (
	"strings"
	"testing"
)

func TestResponseDecoratorsGenerateMetadata(t *testing.T) {
	src := `package demo
@Controller("/users")
type UsersController struct{}
@Get("/redirect")
@HttpCode(301)
@Header("Cache-Control", "no-store")
@Redirect("/new", 301)
func (c *UsersController) Go(@Res({passthrough:true}) response string) string { return response }
`
	f, err := Parse("response.mgo", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Declarations) != 2 || len(f.Declarations[1].Params) != 1 {
		t.Fatalf("unexpected declarations: %+v", f.Declarations)
	}
	if p := f.Declarations[1].Params[0]; p.Kind != "Res" || p.Name != "passthrough" {
		t.Fatalf("unexpected @Res: %+v", p)
	}
	out, err := Generate([]*File{f})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`Name: "Cache-Control"`, `Value: "no-store"`, `RedirectURL: "/new"`, `RedirectStatus: 301`, `Kind: "Res"`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %q: %s", want, out)
		}
	}
}
func TestResponseDecoratorValidation(t *testing.T) {
	for _, tc := range []struct{ decorator, want string }{
		{`@Header("X-One")`, "@Header"},
		{`@Header("X-Test", "bad\nvalue")`, "@Header"},
		{`@Redirect("/go", 200)`, "@Redirect"},
		{`@Redirect("")`, "@Redirect"},
		{`@Res({passthrough:false})`, "@Res"},
	} {
		src := "package demo\n@Controller(\"/\")\ntype C struct{}\n@Get(\"/\")\n"
		if strings.HasPrefix(tc.decorator, "@Res") {
			src += "func (c *C) Get(" + tc.decorator + " response string) string {return response}\n"
		} else {
			src += tc.decorator + "\nfunc (c *C) Get() string {return \"ok\"}\n"
		}
		f, err := Parse("bad.mgo", []byte(src))
		if err == nil {
			_, err = Generate([]*File{f})
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v", tc.decorator, err)
		}
	}
}
