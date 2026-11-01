package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kevenmiano/nestgo/diagnostics"
	"github.com/kevenmiano/nestgo/metadata"
	nestgoruntime "github.com/kevenmiano/nestgo/runtime"
)

type sampleDatabase struct{}
type sampleService struct{ Database *sampleDatabase }
type sampleController struct{ Service *sampleService }

func (*sampleController) Ping() string { return "ok" }

func diagnose(args []string) error {
	flags := flag.NewFlagSet("diagnose", flag.ContinueOnError)
	example := flags.Bool("example", false, "run a built-in example fixture")
	app := flags.String("app", "", "an importable Go application package to diagnose")
	out := flags.String("out", "nestgo-diagnostics", "output directory for self-contained report")
	samples := flags.Int("samples", 15, "independent measurement batches")
	iterations := flags.Int("iterations", 100, "runtime.NewApp calls per batch")
	coldSamples := flags.Int("cold-samples", 3, "number of fresh process launches (0 disables cold measurement)")
	withScale := flags.Bool("scale", false, "benchmark synthetic 1, 10 and 100 provider startup scenarios")
	coldChild := flags.Bool("cold-child", false, "internal process benchmark child")
	strict := flags.Bool("strict", false, "return failure for provider cycles or injection findings")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *coldChild {
		app, err := nestgoruntime.NewApp(context.Background(), nestgoruntime.WithPackages(samplePackage()))
		if err != nil { return err }
		return app.Close()
	}
	if *coldSamples < 0 || *coldSamples > 50 {
		return errors.New("cold-samples must be 0..50")
	}
	if *app != "" {
		if *example || *withScale {
			return errors.New("-app cannot be combined with -example or -scale")
		}
		return diagnoseApplication(*app, *out, *samples, *iterations, *coldSamples, *strict)
	}
	if !*example {
		return errors.New("use -app <importable Go package> or -example")
	}
	pkg := samplePackage()
	graph := diagnostics.Analyze(pkg)
	modules := diagnostics.AnalyzeModules(pkg)
	startup, err := diagnostics.MeasureStartup("built-in sample (3 providers, 1 route)", []metadata.Package{pkg}, *samples, *iterations, os.Getenv("GITHUB_SHA"))
	if err != nil {
		return err
	}
	report := diagnostics.Report{Graph: graph, Modules: modules, Startup: startup}
	if *coldSamples > 0 {
		binary, err := os.Executable()
		if err != nil {
			return err
		}
		cold, err := diagnostics.MeasureProcessStartup(binary, []string{"diagnose", "-cold-child"}, *coldSamples, os.Getenv("GITHUB_SHA"))
		if err != nil {
			return err
		}
		report.Cold = &cold
	}
	if *withScale {
		for _, count := range []int{1, 10, 100} {
			scaled, err := diagnostics.SyntheticProviders(count)
			if err != nil {
				return err
			}
			measured, err := diagnostics.MeasureStartup(fmt.Sprintf("synthetic %d providers, no routes", count), []metadata.Package{scaled}, *samples, *iterations, os.Getenv("GITHUB_SHA"))
			if err != nil {
				return err
			}
			report.Scale = append(report.Scale, diagnostics.ScaleStartup{Providers: count, Startup: measured})
		}
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	for _, format := range []struct {
		name string
		data func() ([]byte, error)
	}{
		{"report.json", report.JSON},
		{"graph.json", graph.JSON},
		{"modules.json", modules.JSON},
		{"modules.dot", func() ([]byte, error) { return []byte(modules.DOT()), nil }},
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
	fmt.Printf("NestGo diagnostics: %s (warm median %.2f µs; p95 %.2f µs; %d allocs/op)\n", filepath.Join(*out, "report.html"), startup.MedianNS/1000, startup.P95NS/1000, startup.AllocsPerOp)
	if *strict && (len(graph.Cycles) > 0 || len(graph.Findings) > 0 || len(modules.Cycles) > 0 || len(modules.Findings) > 0) {
		return errors.New("strict diagnostics found cyclic or unresolved dependencies")
	}
	return nil
}

func samplePackage() metadata.Package {
	return metadata.Package{Controllers: []metadata.Controller{{Type: reflect.TypeOf((*sampleController)(nil)), Prefix: "/health", Routes: []metadata.Route{{Method: "GET", Path: "/ping", Handler: "Ping"}}}}, Providers: []metadata.Provider{
		{Type: reflect.TypeOf((*sampleDatabase)(nil)), Scope: "singleton"},
		{Type: reflect.TypeOf((*sampleService)(nil)), Scope: "singleton"},
		{Type: reflect.TypeOf((*sampleController)(nil)), Scope: "singleton"},
	}}
}
