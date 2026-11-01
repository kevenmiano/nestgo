package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypedDIInterfaceResolution(t *testing.T) {
	cases := []struct {
		name, providers, want string
	}{
		{"resolved", "A", ""},
		{"missing", "", "missing interface dependency"},
		{"ambiguous", "A,B", "ambiguous interface dependency"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package demo\ntype Reader interface { Read() string }\n@Injectable()\ntype A struct{}\nfunc (*A) Read() string { return \"a\" }\n@Injectable()\ntype B struct{}\nfunc (*B) Read() string { return \"b\" }\n@Injectable()\ntype Service struct { Dep Reader }\n@Module({providers:[Service," + tc.providers + "]})\ntype App struct{}\n"
			if tc.providers == "" {
				source = strings.Replace(source, "Service,]", "Service]", 1)
			}
			f, err := Parse("interfaces.mgo", []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			_, err = Generate([]*File{f})
			if tc.want == "" && err != nil {
				t.Fatal(err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestTypedDIVisibilityAcrossThreeModules(t *testing.T) {
	source := "package demo\ntype Reader interface { Read() string }\n@Injectable()\ntype Repo struct{}\nfunc (*Repo) Read() string { return \"ok\" }\n@Injectable()\ntype Service struct { Dep Reader }\n@Module({providers:[Repo],exports:[Repo]})\ntype DatabaseModule struct{}\n@Module({imports:[DatabaseModule],exports:[]})\ntype MiddleModule struct{}\n@Module({imports:[MiddleModule],providers:[Service]})\ntype AppModule struct{}\n"
	f, err := Parse("three.mgo", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate([]*File{f})
	if err == nil || !strings.Contains(err.Error(), "missing interface dependency") {
		t.Fatalf("transitive provider not re-exported should be unavailable: %v", err)
	}
}

func TestTypedDIOptionalInterface(t *testing.T) {
	source := "package demo\ntype Reader interface { Read() string }\n@Injectable()\ntype Service struct { Dep Reader `nestgo:\"optional\"` }\n@Module({providers:[Service]})\ntype AppModule struct{}\n"
	f, err := Parse("optional.mgo", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate([]*File{f}); err != nil {
		t.Fatal(err)
	}
}

func TestTypedDIFailsClosedOnTypeErrors(t *testing.T) {
	f, err := Parse("invalid.mgo", []byte("package demo\n@Injectable()\ntype Service struct { Missing UnknownType }\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate([]*File{f})
	if err == nil || !strings.Contains(err.Error(), "type-check dependency graph") {
		t.Fatalf("expected type-check failure, got %v", err)
	}
}

func TestCompileDirDoesNotWriteOnTypeError(t *testing.T) {
	dir := t.TempDir()
	source := "package demo\n@Injectable()\ntype Service struct { Missing UnknownType }\n"
	if err := os.WriteFile(filepath.Join(dir, "invalid.mgo"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err == nil {
		t.Fatal("expected compiler failure")
	}
	for _, name := range []string{"invalid_generated.go", "zz_modforge_metadata.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected generated artifact %s: %v", name, err)
		}
	}
}

func TestGeneratedRepositoryImplementsInjectedInterface(t *testing.T) {
	source := "package demo\nimport \"context\"\n@Entity({table: \"users\"})\ntype User struct{ ID int64 }\n@Repository(User, \"users\")\ntype UsersRepository struct{}\n@Query(\"SELECT id FROM users WHERE id=:id\")\nfunc (r *UsersRepository) FindByID(ctx context.Context, id int64) (*User, error) { return nil, nil }\ntype UserReader interface {\n\tFindByID(ctx context.Context, id int64) (*User, error)\n}\n@Injectable()\ntype UsersService struct { Repo UserReader `nestgo:\"inject\"` }\n@Module({providers:[UsersService]})\ntype UsersModule struct{}\n"
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "users.mgo"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatalf("generated repository interface must resolve at compile time: %v", err)
	}
	// The compiler must reject an interface that the generated type does not implement.
	invalid := strings.Replace(source, "FindByID(ctx context.Context, id int64) (*User, error)\n}", "UnknownMethod(ctx context.Context, id int64) (*User, error)\n}", 1)
	invalidDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(invalidDir, "users.mgo"), []byte(invalid), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(invalidDir); err == nil || !strings.Contains(err.Error(), "missing interface dependency") {
		t.Fatalf("non-implemented generated repository interface incorrectly accepted: %v", err)
	}
}
