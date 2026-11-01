package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileDirEmitsNamedRepository(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.mgo")
	src := `package users
import "context"
@Entity({table: "users"})
type User struct { ID int64; Name string }
@Repository(User, "users")
type UsersRepository struct{}
@Query("SELECT id,name FROM users WHERE id=:id")
func (r *UsersRepository) FindByID(ctx context.Context, id int64) (*User, error) { return nil, nil }
`
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join(dir, "users_repository_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), `registry.Get("users")`) {
		t.Fatal("named connection binding not generated")
	}
	if err := CompileDir(dir); err != nil {
		t.Fatalf("recompile not idempotent: %v", err)
	}
}
