package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompileCommandSupportsWatchFlagInEitherPosition(t *testing.T) {
	for _, args := range [][]string{
		{"--watch"},
		{"-watch"},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "module.mgo"), []byte("package example\n@Injectable()\ntype Provider struct{}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		args = append(args, dir)
		ctx, cancel := context.WithCancel(context.Background())
		output := new(bytes.Buffer)
		errors := new(bytes.Buffer)
		done := make(chan error, 1)
		go func() { done <- compileCommand(ctx, args, output, errors) }()
		deadline := time.After(3 * time.Second)
		for {
			if _, err := os.Stat(filepath.Join(dir, "zz_nestgo_wiring_generated_test.go")); err == nil {
				break
			}
			select {
			case <-deadline:
				cancel()
				t.Fatal("watch flag did not trigger initial compilation")
			case <-time.After(10 * time.Millisecond):
			}
		}
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("compile watch did not exit on cancellation")
		}
	}
}

func TestCompileCommandRejectsBadArguments(t *testing.T) {
	for _, args := range [][]string{
		nil, {"--watch"}, {"--oops", "./x"}, {"a", "b"},
	} {
		if err := compileCommand(context.Background(), args, new(bytes.Buffer), new(bytes.Buffer)); err == nil {
			t.Fatalf("expected error for args %v", args)
		}
	}
}

func TestCompileCommandSingleShot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dto.mgo")
	if err := os.WriteFile(path, []byte("package example\n@Dto()\ntype ExampleDTO struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	output := new(bytes.Buffer)
	if err := compileCommand(context.Background(), []string{path}, output, new(bytes.Buffer)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "compiled") {
		t.Fatalf("missing compilation output: %s", output.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "zz_nestgo_stubs_test.go")); err != nil {
		t.Fatal(err)
	}
}
