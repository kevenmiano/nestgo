package compiler

import (
	"strings"
	"testing"
)

func TestModuleImportCycleRejected(t *testing.T) {
	source := []byte("package demo\n@Module({imports:[B]})\ntype A struct{}\n@Module({imports:[A]})\ntype B struct{}\n")
	f, err := Parse("cycle.mgo", source)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate([]*File{f})
	if err == nil || !strings.Contains(err.Error(), "module import cycle: A -> B -> A") {
		t.Fatalf("expected deterministic module cycle error, got %v", err)
	}
}

func TestAcyclicModuleImportsAllowed(t *testing.T) {
	source := []byte("package demo\n@Module({imports:[B]})\ntype A struct{}\n@Module()\ntype B struct{}\n")
	f, err := Parse("acyclic.mgo", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate([]*File{f}); err != nil {
		t.Fatal(err)
	}
}
