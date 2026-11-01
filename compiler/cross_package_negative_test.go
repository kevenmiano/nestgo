package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrossPackageInterfaceNegativeDiagnostics(t *testing.T) {
	cases := []struct {
		name      string
		providers string
		field     string
		want      string
	}{
		{
			name:      "ambiguous external interface",
			providers: "@Injectable()\ntype First struct{}\nfunc (*First) Greet() string { return \"one\" }\n@Injectable()\ntype Second struct{}\nfunc (*Second) Greet() string { return \"two\" }\n",
			field:     "Greeter external.Greeter",
			want:      "ambiguous interface dependency",
		},
		{
			name:      "missing external interface",
			providers: "",
			field:     "Greeter external.Greeter",
			want:      "missing interface dependency",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			repo, err := filepath.Abs("..")
			if err != nil {
				t.Fatal(err)
			}
			module := "module example.com/nestgo-di-negative\n\ngo 1.23.0\n\nrequire github.com/kevenmiano/nestgo v0.0.0\nreplace github.com/kevenmiano/nestgo => " + filepath.ToSlash(repo) + "\n"
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(module), 0600); err != nil {
				t.Fatal(err)
			}
			external := filepath.Join(root, "external")
			app := filepath.Join(root, "app")
			for _, dir := range []string{external, app} {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(external, "types.go"), []byte("package external\ntype Greeter interface { Greet() string }\n"), 0600); err != nil {
				t.Fatal(err)
			}
			providers := ""
			if strings.Contains(tc.providers, "First") {
				providers = "First,Second"
			}
			source := "package app\nimport \"example.com/nestgo-di-negative/external\"\n" + tc.providers +
				"@Controller(\"/test\")\ntype Consumer struct { " + tc.field + " }\n" +
				"@Get(\"/\")\nfunc (*Consumer) Read() string { return \"ok\" }\n" +
				"@Module({controllers:[Consumer]" + func() string {
					if providers != "" {
						return ",providers:[" + providers + "]"
					}
					return ""
				}() + "})\ntype AppModule struct{}\n"
			if err := os.WriteFile(filepath.Join(app, "module.mgo"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			err = CompileDir(app)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q diagnostic, got %v", tc.want, err)
			}
			for _, generated := range []string{"module_generated.go", "zz_modforge_metadata.go"} {
				if _, statErr := os.Stat(filepath.Join(app, generated)); !os.IsNotExist(statErr) {
					t.Fatalf("failed compilation left %s: %v", generated, statErr)
				}
			}
		})
	}
}
