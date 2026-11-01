package compiler

import (
	"strings"
	"testing"
)

func TestGenerateRequestPipelineDecorators(t *testing.T) {
	source:=[]byte(`package example
@Injectable()
type AuthGuard struct{}
@Injectable()
type ValidationPipe struct{}
@Injectable()
type ErrorFilter struct{}
@UseGuards(AuthGuard)
@UsePipes([ValidationPipe])
@Controller("/items")
type Items struct{}
@UseFilters(ErrorFilter)
@Get("/{id}")
func (c *Items) Find(@Param("id") id string) string {return id}
`)
	f,err:=Parse("items.mgo",source)
	if err!=nil {t.Fatal(err)}
	result,err:=Generate([]*File{f})
	if err!=nil {t.Fatal(err)}
	for _,want:=range []string{
		"Guards: []reflect.Type{reflect.TypeOf((*AuthGuard)(nil))}",
		"Pipes: []reflect.Type{reflect.TypeOf((*ValidationPipe)(nil))}",
		"Filters: []reflect.Type{reflect.TypeOf((*ErrorFilter)(nil))}",
		"httpadapter.BindParam(ctx, \"id\", 0)",
	} {
		if !strings.Contains(string(result),want) {t.Errorf("missing %q in generated:\n%s",want,result)}
	}
}

func TestRejectUnsafeRequestPipelineDecorators(t *testing.T){
	for _,tc:=range []struct{name,source,want string}{
		{"missing provider",`package x
@Controller("/")
type C struct{}
@UseGuards(G)
@Get("/")
func (c *C) Go() string {return "ok"}`, "local @Injectable provider"},
		{"wrong scope",`package x
@Injectable()
@UseGuards(P)
type P struct{}`, "require @Controller"},
		{"duplicate",`package x
@Injectable()
type G struct{}
@Controller("/")
type C struct{}
@UseGuards(G,G)
@Get("/")
func (c *C) Go() string {return "ok"}`, "duplicate provider"},
		{"empty",`package x
@Controller("/")
type C struct{}
@UsePipes()
@Get("/")
func (c *C) Go() string {return "ok"}`, "requires one or more"},
	}{
		t.Run(tc.name,func(t *testing.T){
			f,err:=Parse("bad.mgo",[]byte(tc.source))
			if err==nil {_,err=Generate([]*File{f})}
			if err==nil || !strings.Contains(err.Error(),tc.want){t.Fatalf("want %q, got %v",tc.want,err)}
		})
	}
}
