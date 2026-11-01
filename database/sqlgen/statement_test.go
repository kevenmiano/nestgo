package sqlgen

import (
	"reflect"
	"strings"
	"testing"
)

func TestCompileStatement(t *testing.T) {
	stmt, err := Compile("SELECT id FROM users WHERE name=:name AND id=:id AND alias=:name", []string{"id", "name"}, PostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.SQL() != "SELECT id FROM users WHERE name=$1 AND id=$2 AND alias=$3" {
		t.Fatalf("unexpected SQL %q", stmt.SQL())
	}
	attack := "' OR 1=1; DROP TABLE users; --"
	args, err := stmt.Bind(map[string]any{"id": 7, "name": attack})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, []any{attack, 7, attack}) || strings.Contains(stmt.SQL(), attack) {
		t.Fatalf("unsafe binding: sql=%q args=%#v", stmt.SQL(), args)
	}
}

func TestCompileStatementValidation(t *testing.T) {
	for _, tc := range []struct {
		sql string
		names []string
	}{
		{"SELECT :id", nil},
		{"SELECT :id", []string{"id", "unused"}},
		{"SELECT :id", []string{"id", "id"}},
		{"SELECT :id", []string{"not-valid"}},
		{"SELECT $$:ignored$$, :id", []string{"ignored", "id"}},
	} {
		if _, err := Compile(tc.sql, tc.names, PostgreSQL); err == nil {
			t.Fatalf("expected error for %q / %#v", tc.sql, tc.names)
		}
	}
	stmt, err := Compile("SELECT :id", []string{"id"}, Question)
	if err != nil { t.Fatal(err) }
	if _, err := stmt.Bind(map[string]any{"other": 1}); err == nil {
		t.Fatal("expected missing parameter")
	}
	if _, err := stmt.Bind(map[string]any{"id": 1, "other": 2}); err == nil {
		t.Fatal("expected extra parameter rejection")
	}
}
