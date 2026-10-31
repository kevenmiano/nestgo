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
