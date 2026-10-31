package compiler

import (
	"bytes"
	"strings"
	"testing"
)

func TestNestJSStyleModuleConfig(t *testing.T) {
	a, err := Parse("users.mgo", []byte("package demo\n@Controller(\"/users\")\ntype UsersController struct{}\n@Injectable()\ntype UsersService struct{}\n@Module({\n name: \"users\",\n imports: [DatabaseModule, CacheModule],\n controllers: [UsersController],\n providers: [UsersService],\n exports: [UsersService],\n})\ntype UsersModule struct{}\n"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse("deps.mgo", []byte("package demo\n@Module({name:\"db\"})\ntype DatabaseModule struct{}\n@Module({name:\"cache\"})\ntype CacheModule struct{}\n"))
	if err != nil {
		t.Fatal(err)
	}
	x, err := Generate([]*File{a, b})
	if err != nil {
		t.Fatal(err)
	}
	y, err := Generate([]*File{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(x, y) {
		t.Fatalf("nondeterministic output")
	}
	for _, expected := range []string{"Imports: []reflect.Type{reflect.TypeOf((*CacheModule)(nil)), reflect.TypeOf((*DatabaseModule)(nil))}", "Providers: []reflect.Type{reflect.TypeOf((*UsersService)(nil))}", "Controllers: []reflect.Type{reflect.TypeOf((*UsersController)(nil))}", "Exports: []reflect.Type{reflect.TypeOf((*UsersService)(nil))}"} {
		if !strings.Contains(string(x), expected) {
			t.Errorf("missing %s", expected)
		}
	}
}

func TestNestJSStyleModuleConfigRejectsStrings(t *testing.T) {
	cases := []struct{ name, decorator, want string }{
		{"string-reference", "@Module({imports:[\"DatabaseModule\"]})", "requires type identifiers"},
		{"unknown-module", "@Module({imports:[Unknown]})", "unknown local module"},
		{"duplicate-module", "@Module({imports:[DatabaseModule,DatabaseModule]})", "duplicate type"},
		{"export-not-in-providers", "@Module({providers:[],exports:[UsersService]})", "must also appear in providers"},
		{"qualified-type", "@Module({imports:[other.DatabaseModule]})", "not declared by a Go import"},
		{"unknown-property", "@Module({foo:[]})", "unknown property"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "package demo\n@Injectable()\ntype UsersService struct{}\n@Module(\"database\")\ntype DatabaseModule struct{}\n" + tc.decorator + "\ntype UsersModule struct{}\n"
			f, err := Parse("invalid.mgo", []byte(src))
			if err == nil {
				_, err = Generate([]*File{f})
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}
