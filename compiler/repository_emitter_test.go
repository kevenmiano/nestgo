package compiler

import (
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

func TestGenerateRepositoryParameterized(t *testing.T) {
	source, err := GenerateRepository(RepositoryDefinition{
		Package: "users", Name: "UsersRepository", Entity: "User", Dialect: sqlgen.PostgreSQL,
		Fields: []RepositoryField{{Name: "ID"}, {Name: "Name"}},
		Methods: []RepositoryMethod{{
			Name: "FindByID", SQL: "SELECT id, name FROM users WHERE id=:id",
			Parameters: []RepositoryParameter{{Name: "id", Type: "int64"}},
		}},
	})
	if err != nil { t.Fatal(err) }
	for _, part := range []string{"type UsersRepositoryGenerated struct", "WHERE id=$1", "QueryRowContext(ctx, query, id)", "Scan(&row.ID, &row.Name)"} {
		if !strings.Contains(string(source), part) { t.Fatalf("missing %q in generated source:\n%s", part, source) }
	}
}

func TestGenerateRepositoryRejectsUnsafeSQLAndNames(t *testing.T) {
	spec := RepositoryDefinition{
		Package: "users", Name: "Repo", Entity: "User", Dialect: sqlgen.Question,
		Fields: []RepositoryField{{Name: "ID"}},
		Methods: []RepositoryMethod{{Name: "Find", SQL: "DELETE FROM users"}},
	}
	if _, err := GenerateRepository(spec); err == nil { t.Fatal("DELETE must be rejected") }
	spec.Methods[0].SQL = "SELECT id FROM users WHERE id=:id"
	spec.Methods[0].Parameters = []RepositoryParameter{{Name: "id", Type: "int64; malicious"}}
	if _, err := GenerateRepository(spec); err == nil { t.Fatal("unsafe codegen type must be rejected") }
}
