package compiler

import (
	"strings"
	"testing"
)

func TestPersistenceDecoratorParsing(t *testing.T) {
	source := []byte(`package demo

@Entity({table: "users"})
type User struct { ID int64 }

@Repository(User)
type UsersRepository struct{}

@Query("SELECT id FROM users WHERE id=:id")
func (r *UsersRepository) FindByID(id int64) (int64, error) {
	return 0, nil
}
`)
	file, err := Parse("repository.mgo", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Declarations) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(file.Declarations))
	}
	if file.Declarations[2].Annotations[0].Name != "Query" {
		t.Fatal("method @Query was not recognized")
	}
}

func TestHTTPQueryParameterPreserved(t *testing.T) {
	file, err := Parse("http.mgo", []byte(`package demo
@Controller("/users")
type UsersController struct{}
@Get("/")
func (c *UsersController) List(@Query("name") name string) string { return name }
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Declarations) != 2 || len(file.Declarations[1].Params) != 1 {
		t.Fatalf("HTTP query binding lost: %#v", file.Declarations)
	}
	if file.Declarations[1].Params[0].Name != "name" {
		t.Fatal("incorrect HTTP query parameter")
	}
}

func TestRepositoryDecoratorCannotDecorateFunction(t *testing.T) {
	_, err := Parse("invalid.mgo", []byte(`package demo
@Repository(User)
func Make() {}
`))
	if err == nil || !strings.Contains(err.Error(), "cannot decorate a function") {
		t.Fatalf("expected diagnostic, got %v", err)
	}
}
