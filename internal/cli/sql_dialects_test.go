package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/compiler"
)

func TestCompileAndCheckNamedSQLDialects(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"users.mgo": `package example
import "context"
@Entity({table:"users"})
type User struct { ID int64 }
@Repository(User, "users")
type UsersRepository struct{}
@Query("SELECT id FROM users WHERE id=:id")
func (r *UsersRepository) Find(ctx context.Context, id int64) (*User, error) { return nil, nil }
`,
		"billing.mgo": `package example
import "context"
@Entity({table:"invoices"})
type Invoice struct { ID int64 }
@Repository(Invoice, "billing")
type BillingRepository struct{}
@Query("SELECT id FROM invoices WHERE id=:id")
func (r *BillingRepository) Find(ctx context.Context, id int64) (*Invoice, error) { return nil, nil }
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	flags := []string{"--connection=users:postgres", "--connection=billing:mysql"}
	compileArgs := append([]string{"compile", dir}, flags...)
	if _, err := executeForTest(compileArgs...); err != nil {
		t.Fatalf("compile with mixed SQL dialects: %v", err)
	}
	for name, want := range map[string]string{
		"users_repository_generated.go": "WHERE id=$1",
		"billing_repository_generated.go": "WHERE id=?",
	} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !strings.Contains(string(data), want) {
			t.Fatalf("%s missing %s: %v", name, want, err)
		}
	}
	checkArgs := append([]string{"check", dir}, flags...)
	if _, err := executeForTest(checkArgs...); err != nil {
		t.Fatalf("matching dialect check: %v", err)
	}
	if _, err := executeForTest("check", dir); !errors.Is(err, compiler.ErrOutdated) {
		t.Fatalf("default dialect incorrectly accepted mixed SQL output: %v", err)
	}
}

func TestSQLDialectFlagsRejectMalformedOrDuplicate(t *testing.T) {
	for _, flags := range [][]string{
		{"--connection="},
		{"--connection=users"},
		{"--connection=users:sqlite"},
		{"--connection=users:postgres", "--connection=users:mysql"},
	} {
		args := append([]string{"compile", t.TempDir()}, flags...)
		if _, err := executeForTest(args...); err == nil {
			t.Fatalf("accepted invalid dialect flags %v", flags)
		}
	}
}
