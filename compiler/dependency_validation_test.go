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
