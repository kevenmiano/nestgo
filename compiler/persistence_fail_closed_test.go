package compiler

import (
	"strings"
	"testing"
)

// A successful parse is not sufficient: unsupported persistence generation
// must fail loudly so an application cannot silently ship empty repositories.
func TestPersistenceGenerationFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name, source, decorator string
	}{
		{"entity", "package demo\n@Entity({table: \"users\"})\ntype User struct{ ID int64 }\n", "@Entity"},
		{"repository", "package demo\n@Repository(User)\ntype Repo struct{}\n", "@Repository"},
		{"query", "package demo\ntype Repo struct{}\n@Query(\"SELECT 1\")\nfunc (r *Repo) Find() int { return 1 }\n", "@Query"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := Parse(tc.name+".mgo", []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			_, err = Generate([]*File{file})
			if err == nil || !strings.Contains(err.Error(), "not yet implemented") || !strings.Contains(err.Error(), tc.decorator) {
				t.Fatalf("expected explicit unsupported decorator diagnostic, got %v", err)
			}
		})
	}
}
