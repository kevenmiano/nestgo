package compiler

import (
	"strings"
	"testing"
)

func TestMissingDependencyValidation(t *testing.T) {
	f, err := Parse("missing.mgo", []byte("package demo\ntype Repo struct{}\n@Injectable()\ntype Service struct { Repo *Repo }\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate([]*File{f})
	if err == nil || !strings.Contains(err.Error(), "missing required provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeclaredDependencyPassesValidation(t *testing.T) {
	f, err := Parse("registered.mgo", []byte("package demo\n@Injectable()\ntype Repo struct{}\n@Injectable()\ntype Service struct { Repo *Repo }\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate([]*File{f}); err != nil {
		t.Fatal(err)
	}
}

func TestMissingControllerDependencyValidation(t *testing.T) {
	f, err := Parse("controller.mgo", []byte("package demo\ntype Repo struct{}\n@Controller(\"/\")\ntype C struct { Repo *Repo }\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate([]*File{f}); err == nil || !strings.Contains(err.Error(), "missing required provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExplicitlyExcludedDependencyPassesCompile(t *testing.T) {
	f, err := Parse("ignored.mgo", []byte("package demo\ntype External struct{}\n@Injectable()\ntype Service struct { External *External `nestgo:\"-\"` }\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate([]*File{f}); err != nil {
		t.Fatalf("ignored field must not require provider: %v", err)
	}
}
