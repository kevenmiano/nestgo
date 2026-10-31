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

func TestModuleCycleAcrossFilesDeterministic(t *testing.T) {
	a, err := Parse("a.mgo", []byte("package demo\n@Module({imports:[B]})\ntype A struct{}\n"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse("b.mgo", []byte("package demo\n@Module({imports:[C]})\ntype B struct{}\n"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse("c.mgo", []byte("package demo\n@Module({imports:[A]})\ntype C struct{}\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, files := range [][]*File{{a, b, c}, {c, a, b}, {b, c, a}} {
		_, err := Generate(files)
		if err == nil || !strings.Contains(err.Error(), "module import cycle: A -> B -> C -> A") {
			t.Fatalf("expected stable cycle for source order, got %v", err)
		}
	}
}

func TestModuleSelfCycleRejected(t *testing.T) {
	f, err := Parse("self.mgo", []byte("package demo\n@Module({imports:[A]})\ntype A struct{}\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Generate([]*File{f})
	if err == nil || !strings.Contains(err.Error(), "module import cycle: A -> A") {
		t.Fatalf("expected self-cycle error, got %v", err)
	}
}
