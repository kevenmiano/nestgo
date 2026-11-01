package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/compiler"
)

func TestCheckCommandIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "provider.mgo")
	if err := os.WriteFile(source, []byte("package example\n@Injectable()\ntype Provider struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest("check", dir); !errors.Is(err, compiler.ErrOutdated) {
		t.Fatalf("expected missing generated artifact: %v", err)
	}
	output, err := executeForTest("compile", dir)
	if err != nil || !strings.Contains(output, "compiled") {
		t.Fatalf("compile failed: %v %s", err, output)
	}
	output, err = executeForTest("check", source)
	if err != nil || !strings.Contains(output, "up to date") {
		t.Fatalf("check failed: %v %s", err, output)
	}
	if _, err := executeForTest("check"); err == nil {
		t.Fatal("check accepted missing path")
	}
	if _, err := executeForTest("check", dir, dir); err == nil {
		t.Fatal("check accepted too many paths")
	}
	if _, err := executeForTest("check", "does-not-exist.mgo"); err == nil {
		t.Fatal("check accepted absent source")
	}
}
