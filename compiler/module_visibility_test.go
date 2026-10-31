package compiler

import (
	"strings"
	"testing"
)

func TestModuleProviderVisibility(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "imported and exported",
			src: "package demo\n@Injectable()\ntype Repo struct{}\n@Injectable()\ntype Service struct { Repo *Repo }\n@Module({providers:[Repo],exports:[Repo]})\ntype DatabaseModule struct{}\n@Module({imports:[DatabaseModule],providers:[Service]})\ntype AppModule struct{}\n",
		},
		{
			name: "missing import",
			src: "package demo\n@Injectable()\ntype Repo struct{}\n@Injectable()\ntype Service struct { Repo *Repo }\n@Module({providers:[Repo],exports:[Repo]})\ntype DatabaseModule struct{}\n@Module({providers:[Service]})\ntype AppModule struct{}\n",
			want: "inaccessible provider Repo",
		},
		{
			name: "not exported",
			src: "package demo\n@Injectable()\ntype Repo struct{}\n@Injectable()\ntype Service struct { Repo *Repo }\n@Module({providers:[Repo]})\ntype DatabaseModule struct{}\n@Module({imports:[DatabaseModule],providers:[Service]})\ntype AppModule struct{}\n",
			want: "inaccessible provider Repo",
		},
		{
			name: "same module",
			src: "package demo\n@Injectable()\ntype Repo struct{}\n@Injectable()\ntype Service struct { Repo *Repo }\n@Module({providers:[Repo,Service]})\ntype AppModule struct{}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := Parse("visibility.mgo", []byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			_, err = Generate([]*File{f})
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("expected %q, got %v", tc.want, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
