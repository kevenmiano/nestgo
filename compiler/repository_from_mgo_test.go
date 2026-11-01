package compiler

import (
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

func TestRepositoryFromMGO(t *testing.T) {
	file, err := Parse("users.mgo", []byte(`package users
import "context"
@Entity({table: "users"})
type User struct { ID int64; Name string }
@Repository(User)
type UsersRepository struct{}
@Query("SELECT id,name FROM users WHERE id=:id")
func (r *UsersRepository) FindByID(ctx context.Context, id int64) (*User, error) { return nil, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	def, err := RepositoryFromMGO(file, sqlgen.PostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	if def.Name != "UsersRepository" || len(def.Fields) != 2 || len(def.Methods) != 1 {
		t.Fatalf("bad plan: %+v", def)
	}
	gen, err := GenerateRepository(def)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"WHERE id=$1", "QueryContext(ctx, query, id)", "rows.Scan(targets...)", "case \"id\":", "case \"name\":"} {
		if !strings.Contains(string(gen), expected) {
			t.Fatalf("missing %s in %s", expected, gen)
		}
	}
}


func TestRepositoryFromMGOWriteMethod(t *testing.T) {
	file, err := Parse("users.mgo", []byte(`package users
import "context"
@Entity({table: "users"})
type User struct { ID int64; Name string }
@Repository(User, "users")
type UsersRepository struct{}
@Query("UPDATE users SET name=:name WHERE id=:id")
func (r *UsersRepository) Update(ctx context.Context, name string, id int64) (int64, error) { return 0, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	def, err := RepositoryFromMGO(file, sqlgen.PostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Methods) != 1 || !def.Methods[0].Exec {
		t.Fatalf("expected DML repository method: %#v", def.Methods)
	}
	generated, err := GenerateRepository(def)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "ExecContext(ctx, query, name, id)") {
		t.Fatalf("missing generated write call: %s", generated)
	}
}
