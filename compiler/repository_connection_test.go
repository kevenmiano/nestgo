package compiler

import (
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

func TestRepositoryNamedConnectionBinding(t *testing.T) {
	entity, connection, err := parseRepositoryBinding(`User, "users"`)
	if err != nil || entity != "User" || connection != "users" {
		t.Fatalf("parsed entity=%q connection=%q err=%v", entity, connection, err)
	}
	for _, bad := range []string{`User, ""`, `User, billing-db`, `User, "users", "other"`} {
		if _, _, err := parseRepositoryBinding(bad); err == nil {
			t.Fatalf("accepted invalid binding %q", bad)
		}
	}
	output, err := GenerateRepository(RepositoryDefinition{
		Package:    "users",
		Name:       "UsersRepository",
		Entity:     "User",
		Connection: "users",
		Fields:     []RepositoryField{{Name: "ID"}},
		Methods:    []RepositoryMethod{{Name: "Find", SQL: "SELECT id FROM users WHERE id=:id", Parameters: []RepositoryParameter{{Name: "id", Type: "int64"}}}},
		Dialect:    sqlgen.PostgreSQL,
	})
	if err != nil {
		t.Fatal(err)
	}
	source := string(output)
	if !strings.Contains(source, "NewUsersRepositoryGeneratedFromConnections") || !strings.Contains(source, `registry.Get("users")`) {
		t.Fatalf("named pool binding missing: %s", source)
	}
}
