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
	if err != nil { t.Fatal(err) }
	def, err := RepositoryFromMGO(file, sqlgen.PostgreSQL)
	if err != nil { t.Fatal(err) }
	if def.Name != "UsersRepository" || len(def.Fields) != 2 || len(def.Methods) != 1 { t.Fatalf("bad plan: %+v", def) }
	gen, err := GenerateRepository(def)
	if err != nil { t.Fatal(err) }
	for _, expected := range []string{"WHERE id=$1", "QueryRowContext(ctx, query, id)", "Scan(&row.ID, &row.Name)"} {
		if !strings.Contains(string(gen), expected) { t.Fatalf("missing %s in %s", expected, gen) }
	}
}
