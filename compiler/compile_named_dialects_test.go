package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

func TestCompileDirWithNamedConnectionDialects(t *testing.T) {
	dir := t.TempDir()
	fixtures := map[string]string{
		"users.mgo": `package shop
import "context"
@Entity({table: "users"})
type User struct{ ID int64 }
@Repository(User, "users")
type UsersRepository struct{}
@Query("SELECT id FROM users WHERE id=:id")
func (r *UsersRepository) Find(ctx context.Context, id int64) (*User, error) { return nil, nil }
`,
		"billing.mgo": `package shop
import "context"
@Entity({table: "invoices"})
type Invoice struct{ ID int64 }
@Repository(Invoice, "billing")
type BillingRepository struct{}
@Query("SELECT id FROM invoices WHERE id=:id")
func (r *BillingRepository) Find(ctx context.Context, id int64) (*Invoice, error) { return nil, nil }
`,
	}
	for name, source := range fixtures {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	dialects := map[string]sqlgen.Dialect{"users": sqlgen.PostgreSQL, "billing": sqlgen.Question}
	if err := CompileDirWithDialects(dir, dialects); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ file, query string }{
		{"users_repository_generated.go", "WHERE id=$1"},
		{"billing_repository_generated.go", "WHERE id=?"},
	} {
		output, err := os.ReadFile(filepath.Join(dir, tc.file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(output), tc.query) {
			t.Fatalf("%s missing %q: %s", tc.file, tc.query, output)
		}
	}
	if err := CompileDirWithDialects(dir, map[string]sqlgen.Dialect{"users": sqlgen.PostgreSQL}); err == nil || !strings.Contains(err.Error(), "missing SQL dialect for connection") {
		t.Fatalf("expected missing billing dialect, got %v", err)
	}
}

func TestCompileDirWithDialectsRejectsInvalidMappings(t *testing.T) {
	for _, mapping := range []map[string]sqlgen.Dialect{
		nil,
		{"billing-prod": sqlgen.PostgreSQL},
		{"users": sqlgen.Dialect(255)},
	} {
		if err := CompileDirWithDialects(t.TempDir(), mapping); err == nil {
			t.Fatalf("expected invalid mapping rejection for %#v", mapping)
		}
	}
}
