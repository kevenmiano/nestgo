package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/kevenmiano/nestgo/metadata"
	nestgoruntime "github.com/kevenmiano/nestgo/runtime"
)

// RunRegistered analyzes packages explicitly registered by imports in the
// current executable. The invoking program must import its NestGo app packages.
// This cannot discover dynamic Go types that were never linked or registered.
func RunRegistered(output string, samples, iterations, coldSamples int, strict, coldChild bool, commit string) error {
	registered := metadata.All()
	names := make([]string, 0, len(registered))
	for name := range registered {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return errors.New("no NestGo packages were registered; import the generated metadata package")
	}
	packages := make([]metadata.Package, 0, len(names))
	for _, name := range names {
		packages = append(packages, registered[name])
	}
	if coldChild {
		app, err := nestgoruntime.NewApp(context.Background(), nestgoruntime.WithPackages(packages...))
		if err != nil {
			return err
		}
		return app.Close()
	}
	if coldSamples < 0 || coldSamples > 50 {
		return errors.New("cold-samples must be 0..50")
	}
	graph := Analyze(packages...)
	modules := AnalyzeModules(packages...)
	startup, err := MeasureStartup("registered application packages", packages, samples, iterations, commit)
	if err != nil {
		return fmt.Errorf("cannot initialize registered NestGo packages: %w", err)
	}
	report := Report{Graph: graph, Modules: modules, Startup: startup}
	if coldSamples > 0 {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		cold, err := MeasureProcessStartup(exe, []string{"-cold-child"}, coldSamples, commit)
		if err != nil {
			return err
		}
		report.Cold = &cold
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	for _, artifact := range []struct {
		name  string
		write func() ([]byte, error)
	}{
		{"graph.json", graph.JSON},
		{"modules.json", modules.JSON},
		{"modules.dot", func() ([]byte, error) { return []byte(modules.DOT()), nil }},
		{"graph.dot", func() ([]byte, error) { return []byte(graph.DOT()), nil }},
		{"report.json", report.JSON},
		{"report.html", report.HTML},
	} {
		data, err := artifact.write()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(output, artifact.name), data, 0644); err != nil {
			return err
		}
	}
	fmt.Printf("NestGo application diagnostics: %s (%d registered packages; median %.2f µs, p95 %.2f µs)\n",
		filepath.Join(output, "report.html"), len(names), startup.MedianNS/1000, startup.P95NS/1000)
	if strict && (len(graph.Cycles) > 0 || len(graph.Findings) > 0 || len(modules.Cycles) > 0 || len(modules.Findings) > 0) {
		return errors.New("strict diagnostics found provider cycles or unresolved/ambiguous dependencies")
	}
	return nil
}
