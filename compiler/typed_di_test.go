package compiler

import (
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
