package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// diagnoseApplication compiles a temporary, standalone diagnostic runner that
// imports the user's compiled package and thereby executes its generated
// metadata.Register init calls. It never edits user project files.
func diagnoseApplication(target, output string, samples, iterations, coldSamples int, strict bool) error {
	if target == "" {
		return fmt.Errorf("-app requires a Go package path or directory")
	}
	getPackage := func(format string) (string, error) {
		command := exec.Command("go", "list", "-f", format, target)
		raw, err := command.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("go list %s: %w: %s", target, err, strings.TrimSpace(string(raw)))
		}
		return strings.TrimSpace(string(raw)), nil
	}
	importPath, err := getPackage("{{.ImportPath}}")
	if err != nil {
		return err
	}
	name, err := getPackage("{{.Name}}")
	if err != nil {
		return err
	}
	if name == "main" {
		return fmt.Errorf("%s is package main and cannot be imported; move generated metadata into an importable package", target)
	}
	if strings.HasPrefix(importPath, "command-line-arguments") || importPath == "" {
		return fmt.Errorf("cannot determine an importable Go package for %s", target)
	}
	dir, err := os.MkdirTemp("", "nestgo-diagnose-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	source := fmt.Sprintf(`package main

import (
	"flag"
	"fmt"
	"os"

	_ %q
	"github.com/kevenmiano/nestgo/diagnostics"
)

func main() {
	out := flag.String("out", "nestgo-diagnostics", "output directory")
	samples := flag.Int("samples", 15, "warm samples")
	iterations := flag.Int("iterations", 100, "warm iterations")
	cold := flag.Int("cold-samples", 3, "cold process launches")
	strict := flag.Bool("strict", false, "fail for dependency findings")
	child := flag.Bool("cold-child", false, "internal child flag")
	flag.Parse()
	if err := diagnostics.RunRegistered(*out, *samples, *iterations, *cold, *strict, *child, os.Getenv("GITHUB_SHA")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`, importPath)
	sourceFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(sourceFile, []byte(source), 0600); err != nil {
		return err
	}
	binary := filepath.Join(dir, "nestgo-diagnose-runner")
	build := exec.Command("go", "build", "-o", binary, sourceFile)
	if output, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("compile diagnostics for %s: %w: %s", importPath, err, strings.TrimSpace(string(output)))
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	args := []string{
		"-out", output,
		"-samples", strconv.Itoa(samples),
		"-iterations", strconv.Itoa(iterations),
		"-cold-samples", strconv.Itoa(coldSamples),
	}
	if strict {
		args = append(args, "-strict")
	}
	run := exec.Command(binary, args...)
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if err := run.Run(); err != nil {
		return fmt.Errorf("analyze application %s: %w", importPath, err)
	}
	return nil
}
