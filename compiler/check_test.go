package compiler

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDirIsReadOnlyAndDetectsGeneratedDrift(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "providers.mgo")
	if err := os.WriteFile(source, []byte("package example\n@Injectable()\ntype Provider struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(dir); !errors.Is(err, ErrOutdated) {
		t.Fatalf("expected missing artifacts error, got %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "providers.mgo" {
		t.Fatalf("check wrote generated output: %v", entries)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(dir); err != nil {
		t.Fatalf("generated output should be current: %v", err)
	}
	generated := filepath.Join(dir, "zz_modforge_metadata.go")
	data, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	stale := append(append([]byte(nil), data...), []byte("\n// stale\n")...)
	if err := os.WriteFile(generated, stale, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(dir); !errors.Is(err, ErrOutdated) || !strings.Contains(err.Error(), "zz_modforge_metadata.go (outdated)") {
		t.Fatalf("expected stale generated code error, got %v", err)
	}
	got, err := os.ReadFile(generated)
	if err != nil || string(got) != string(stale) {
		t.Fatalf("check modified generated artifact: %v", err)
	}
	if err := os.Remove(generated); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(dir); !errors.Is(err, ErrOutdated) || !strings.Contains(err.Error(), "zz_modforge_metadata.go (missing)") {
		t.Fatalf("expected missing generated code error, got %v", err)
	}
	if _, err := os.Stat(generated); !os.IsNotExist(err) {
		t.Fatalf("check created the missing artifact: %v", err)
	}
}

func TestCheckDirDoesNotTouchEditableStub(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "users.mgo"), []byte("package example\n@Injectable()\ntype UsersService struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "zz_nestgo_stubs_test.go")
	edited := []byte("// developer-owned changes\npackage example\nimport \"testing\"\nfunc TestUsersService(t *testing.T) { t.Log(\"custom assertions\") }\n")
	if err := os.WriteFile(path, edited, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(dir); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(path)
	if err != nil || string(result) != string(edited) {
		t.Fatalf("check rewrote developer tests: %v", err)
	}
}

func TestCheckDirRejectsInvalidSourceWithoutOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "invalid.mgo"), []byte("package example\n@Injectable()\ntype Broken struct {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(dir); err == nil || errors.Is(err, ErrOutdated) {
		t.Fatalf("expected compiler diagnostic, got %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("invalid check created files: %v %v", entries, err)
	}
}
