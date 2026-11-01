package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileDirRejectsOrphanQueryBeforeWrites(t *testing.T) {
	dir := t.TempDir()
	source := `package orphan
import "context"
type User struct { ID int64 }
type UserRepository struct{}
@Query("SELECT id FROM users WHERE id=:id")
func (r *UserRepository) Find(ctx context.Context, id int64) (*User, error) { return nil, nil }
`
	if err := os.WriteFile(filepath.Join(dir, "orphan.mgo"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	err := CompileDir(dir)
	if err == nil || !strings.Contains(err.Error(), "@Query requires a local @Repository receiver") {
		t.Fatalf("expected orphan query diagnostic, got %v", err)
	}
	for _, file := range []string{"orphan_generated.go", "orphan_repository_generated.go", "zz_modforge_metadata.go"} {
		if _, err := os.Stat(filepath.Join(dir, file)); !os.IsNotExist(err) {
			t.Fatalf("unexpected output after failed compilation: %s", file)
		}
	}
}
