package compiler

import (
	"strings"
	"testing"
)

func TestImportedInterfaceDependency(t *testing.T) {
	source := "package demo\nimport \"io\"\n@Injectable()\ntype Reader struct{}\nfunc (*Reader) Read(p []byte) (int, error) { return 0, io.EOF }\n@Injectable()\ntype Service struct { Source io.Reader }\n@Module({providers:[Reader,Service]})\ntype App struct{}\n"
	f, err := Parse("imported.mgo", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate([]*File{f}); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyInterfaceDependencyMissing(t *testing.T) {
	source := "package demo\ntype Reader interface { Read() string }\n@Injectable()\ntype Service struct { Source Reader }\n"
	f, err := Parse("legacy.mgo", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate([]*File{f})
	if err == nil || !strings.Contains(err.Error(), "missing interface dependency") {
		t.Fatalf("expected missing interface dependency, got %v", err)
	}
}
