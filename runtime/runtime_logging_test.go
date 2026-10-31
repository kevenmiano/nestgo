package runtime

import (
	"bytes"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/logging"
	"github.com/kevenmiano/nestgo/metadata"
)

type lifecycleA struct{}
type lifecycleB struct{}
type lifecycleController struct{}

func (*lifecycleController) List() string { return "ok" }
func lifecyclePackage() metadata.Package {
	return metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*lifecycleA)(nil))}, {Type: reflect.TypeOf((*lifecycleB)(nil))}}, Controllers: []metadata.Controller{{Type: reflect.TypeOf((*lifecycleController)(nil)), Prefix: "/test", Routes: []metadata.Route{{Handler: "List", Path: "/", Method: "GET"}}}}}
}
func TestBootstrapLifecycle(t *testing.T) {
	var b bytes.Buffer
	l := logging.New(logging.Config{Logger: logging.NewPretty(&b, false), MinLevel: logging.LevelDebug})
	if _, err := NewWithOptions(Options{Logger: l, Debug: true}, lifecyclePackage()); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, item := range []string{"NestFactory", "Provider instantiated", "Provider dependencies resolved", "RoutesResolver", "RouterExplorer", "Routes mapped", "NestApplication", "duration="} {
		if !strings.Contains(out, item) {
			t.Errorf("missing %q in %s", item, out)
		}
	}
	if strings.Index(out, "lifecycleA") > strings.Index(out, "lifecycleB") {
		t.Fatal("provider event order changed")
	}
}
func TestDuplicateProviderLogging(t *testing.T) {
	var b bytes.Buffer
	l := logging.New(logging.Config{Logger: logging.Slog{Logger: slog.New(slog.NewJSONHandler(&b, nil))}})
	p := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*lifecycleA)(nil))}, {Type: reflect.TypeOf((*lifecycleA)(nil))}}}
	_, err := NewWithOptions(Options{Logger: l}, p)
	if err == nil || !strings.Contains(err.Error(), "duplicate provider") || !strings.Contains(b.String(), "Bootstrap failed") {
		t.Fatalf("expected logged duplicate: %v %s", err, b.String())
	}
}
func TestMissingRouteHandlerLogging(t *testing.T) {
	var b bytes.Buffer
	l := logging.New(logging.Config{Logger: logging.NewPretty(&b, false)})
	p := metadata.Package{Controllers: []metadata.Controller{{Type: reflect.TypeOf((*lifecycleController)(nil)), Routes: []metadata.Route{{Handler: "Unknown", Method: "GET", Path: "/"}}}}}
	_, err := NewWithOptions(Options{Logger: l}, p)
	if err == nil || !strings.Contains(b.String(), "Bootstrap failed") {
		t.Fatalf("expected logged error: %v %s", err, b.String())
	}
}
func TestDebugDisabled(t *testing.T) {
	var b bytes.Buffer
	l := logging.New(logging.Config{Logger: logging.NewPretty(&b, false), MinLevel: logging.LevelDebug})
	if _, err := NewWithOptions(Options{Logger: l}, lifecyclePackage()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "Provider instantiated") {
		t.Fatal("unexpected debug log")
	}
}
func BenchmarkBootstrapLogging(b *testing.B) {
	for _, mode := range []struct {
		name  string
		min   logging.Level
		debug bool
	}{
		{"filtered", logging.LevelError, false},
		{"info", logging.LevelInfo, false},
		{"debug", logging.LevelDebug, true},
	} {
		b.Run(mode.name, func(b *testing.B) {
			l := logging.New(logging.Config{Logger: logging.NewPretty(&bytes.Buffer{}, false), MinLevel: mode.min})
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := NewWithOptions(Options{Logger: l, Debug: mode.debug}, lifecyclePackage()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

type missingRequired struct{ Dependency *lifecycleA }
type optionalRequired struct{ Dependency *lifecycleA `nestgo:"optional"` }
type missingInterface interface{ Missing() }
type missingInterfaceProvider struct{ Dependency missingInterface }

func TestMissingRequiredDependencyBlocksBootstrap(t *testing.T) {
	p := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*missingRequired)(nil))}}}
	if _, err := NewWithOptions(Options{}, p); err == nil || !strings.Contains(err.Error(), "missing required dependency") {
		t.Fatalf("expected missing dependency, got %v", err)
	}
}
func TestOptionalDependencyAllowed(t *testing.T) {
	p := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*optionalRequired)(nil))}}}
	if _, err := NewWithOptions(Options{}, p); err != nil {
		t.Fatal(err)
	}
}
func TestMissingInterfaceBlocksBootstrap(t *testing.T) {
	p := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*missingInterfaceProvider)(nil))}}}
	if _, err := NewWithOptions(Options{}, p); err == nil || !strings.Contains(err.Error(), "missing required dependency") {
		t.Fatalf("expected missing interface, got %v", err)
	}
}
func TestNilMetadataDoesNotPanic(t *testing.T) {
	cases := []metadata.Package{
		{Providers: []metadata.Provider{{Type: nil}}},
		{Controllers: []metadata.Controller{{Type: nil}}},
	}
	for _, p := range cases {
		if _, err := NewWithOptions(Options{}, p); err == nil {
			t.Fatal("expected nil metadata error")
		}
	}
}
