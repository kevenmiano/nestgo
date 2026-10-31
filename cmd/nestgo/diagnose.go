package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kevenmiano/nestgo/diagnostics"
	"github.com/kevenmiano/nestgo/metadata"
)

type sampleDatabase struct{}
type sampleService struct{ Database *sampleDatabase }
type sampleController struct{ Service *sampleService }

func (*sampleController) Ping() string { return "ok" }

func diagnose(args []string) error {
	flags := flag.NewFlagSet("diagnose", flag.ContinueOnError)
	example := flags.Bool("example", false, "run a built-in example fixture (not the user's application)")
	out := flags.String("out", "nestgo-diagnostics", "output directory for self-contained report")
	samples := flags.Int("samples", 15, "independent measurement batches")
	iterations := flags.Int("iterations", 100, "runtime.New calls per batch")
	strict := flags.Bool("strict", false, "return failure for provider cycles or injection findings")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*example {
		return errors.New("diagnose currently requires -example; real applications use diagnostics.Analyze and diagnostics.MeasureStartup programmatically with their metadata.Package values")
	}
	pkg := metadata.Package{Controllers: []metadata.Controller{{Type: reflect.TypeOf((*sampleController)(nil)), Prefix: "/health", Routes: []metadata.Route{{Method: "GET", Path: "/ping", Handler: "Ping"}}}}, Providers: []metadata.Provider{
		{Type: reflect.TypeOf((*sampleDatabase)(nil)), Scope: "singleton"},
		{Type: reflect.TypeOf((*sampleService)(nil)), Scope: "singleton"},
		{Type: reflect.TypeOf((*sampleController)(nil)), Scope: "singleton"},
	}}
	graph := diagnostics.Analyze(pkg)
	startup, err := diagnostics.MeasureStartup("built-in sample (3 providers, 1 route)", []metadata.Package{pkg}, *samples, *iterations, os.Getenv("GITHUB_SHA"))
	if err != nil {
		return err
	}
	report := diagnostics.Report{Graph: graph, Startup: startup}
	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	for _, format := range []struct {
		name string
		data func() ([]byte, error)
	}{
		{"report.json", report.JSON},
		{"graph.json", graph.JSON},
		{"graph.dot", func() ([]byte, error) { return []byte(graph.DOT()), nil }},
		{"report.html", report.HTML},
	} {
		data, err := format.data()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(*out, format.name), data, 0644); err != nil {
			return err
		}
	}
	fmt.Printf("NestGo diagnostics: %s (median %.2f µs; p95 %.2f µs; %d allocs/op)\n", filepath.Join(*out, "report.html"), startup.MedianNS/1000, startup.P95NS/1000, startup.AllocsPerOp)
	if *strict && (len(graph.Cycles) > 0 || len(graph.Findings) > 0) {
		return errors.New("strict diagnostics found cyclic or unresolved dependencies")
	}
	return nil
}
