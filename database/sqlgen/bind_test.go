package sqlgen

import (
	"reflect"
	"strings"
	"testing"
)

func TestBindPostgres(t *testing.T) {
	q, args, err := Bind("SELECT ':ignored', \"column:name\" FROM users WHERE id=:id AND alt=:id AND x=:name::text -- :comment\n/* :block */", map[string]any{"id": 42, "name": "Ada"}, PostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT ':ignored', \"column:name\" FROM users WHERE id=$1 AND alt=$2 AND x=$3::text -- :comment\n/* :block */"
	if q != want {
		t.Fatalf("got %q want %q", q, want)
	}
	if !reflect.DeepEqual(args, []any{42, 42, "Ada"}) {
		t.Fatalf("args %#v", args)
	}
}

func TestQuestion(t *testing.T) {
	q, args, err := Bind("SELECT * FROM users WHERE id=:id", map[string]any{"id": 7}, Question)
	if err != nil || q != "SELECT * FROM users WHERE id=?" || len(args) != 1 || args[0] != 7 {
		t.Fatalf("%q %#v %v", q, args, err)
	}
}

func TestValidation(t *testing.T) {
	for _, tc := range []struct {
		q string
		p map[string]any
	}{
		{"SELECT :missing", nil},
		{"SELECT 1", map[string]any{"unused": 1}},
		{"SELECT 'unterminated", nil},
		{"/* unterminated", nil},
		{"SELECT $$unterminated", nil},
	} {
		_, _, err := Bind(tc.q, tc.p, PostgreSQL)
		if err == nil {
			t.Fatalf("expected error for %q", tc.q)
		}
	}
	_, _, err := Bind("SELECT 1", nil, Dialect(255))
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatal(err)
	}
}

func TestInjectionRemainsArgument(t *testing.T) {
	payload := "' OR 1=1; DROP TABLE users; --"
	q, args, err := Bind("SELECT id FROM users WHERE username=:username", map[string]any{"username": payload}, PostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	if q != "SELECT id FROM users WHERE username=$1" || !reflect.DeepEqual(args, []any{payload}) {
		t.Fatalf("SQL or arguments changed: %q %#v", q, args)
	}
	if strings.Contains(q, payload) {
		t.Fatal("payload interpolated into SQL")
	}
}

func TestDollarQuotedLiterals(t *testing.T) {
	for _, query := range []string{
		"SELECT $tag$:admin$tag$, :id",
		"SELECT $$:admin$$, :id",
		"SELECT /* outer /* :ignored */ outer */ :id",
		"SELECT ':ignored', :id",
	} {
		q, args, err := Bind(query, map[string]any{"id": 9}, PostgreSQL)
		if err != nil {
			t.Fatalf("%q: %v", query, err)
		}
		if !strings.HasSuffix(q, "$1") || len(args) != 1 || args[0] != 9 {
			t.Fatalf("unexpected binding: %q %#v", q, args)
		}
		if strings.Contains(q, "$2") || strings.Contains(q, ":id") {
			t.Fatalf("unexpected replacement: %q", q)
		}
	}
}

func FuzzBind(f *testing.F) {
	f.Add("x' OR 1=1 --")
	f.Add("'; DROP TABLE users; --")
	f.Add("\x00:admin$$ /*")
	f.Fuzz(func(t *testing.T, payload string) {
		sql, args, err := Bind("SELECT id FROM users WHERE name=:name", map[string]any{"name": payload}, PostgreSQL)
		if err != nil || sql != "SELECT id FROM users WHERE name=$1" || len(args) != 1 || args[0] != payload {
			t.Fatalf("unsafe binding: %q %#v %v", sql, args, err)
		}
	})
}

func TestEscapedSQLLiteralsAndComments(t *testing.T) {
	for _, tc := range []struct {
		name, query, want string
		dialect           Dialect
	}{
		{"postgres E-string", `SELECT E'it\'s :hidden', :id`, `SELECT E'it\'s :hidden', $1`, PostgreSQL},
		{"mysql single quote", `SELECT 'it\'s :hidden', :id`, `SELECT 'it\'s :hidden', ?`, Question},
		{"mysql hash comment", `SELECT :id # :ignored`, `SELECT ? # :ignored`, Question},
		{"postgres plain quotes", `SELECT 'it''s :hidden', :id`, `SELECT 'it''s :hidden', $1`, PostgreSQL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query, args, err := Bind(tc.query, map[string]any{"id": 17}, tc.dialect)
			if err != nil {
				t.Fatal(err)
			}
			if query != tc.want || !reflect.DeepEqual(args, []any{17}) {
				t.Fatalf("query=%q, args=%v, want %q", query, args, tc.want)
			}
		})
	}
}
