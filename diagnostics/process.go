package diagnostics

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"time"

	"github.com/kevenmiano/nestgo/metadata"
)

// ProcessStartup measures launch-to-exit of an already-built child process.
// Includes process boot, package initialization and runtime.New, but not build,
// socket listen or network readiness.
type ProcessStartup struct {
	Scenario string `json:"scenario"`
	GoVersion string `json:"goVersion"`
	GOOS string `json:"goos"`
	GOARCH string `json:"goarch"`
	CPUCount int `json:"cpuCount"`
	Commit string `json:"commit,omitempty"`
	SamplesNS []float64 `json:"samplesNs"`
	MedianNS float64 `json:"medianNs"`
	P95NS float64 `json:"p95Ns"`
}

// MeasureProcessStartup runs a trusted executable directly (never via a shell)
// and propagates all child failures rather than treating them as measurements.
func MeasureProcessStartup(binary string, args []string, samples int, commit string) (ProcessStartup, error) {
	if binary == "" || samples < 1 || samples > 50 {
		return ProcessStartup{}, fmt.Errorf("binary required and cold samples must be 1..50")
	}
	m := ProcessStartup{Scenario: "process boot, Go init, CLI and NestGo runtime.New",
		GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		CPUCount: runtime.NumCPU(), Commit: commit, SamplesNS: make([]float64, 0, samples)}
	for i := 0; i < samples; i++ {
		cmd := exec.Command(binary, args...)
		cmd.Env = os.Environ()
		start := time.Now()
		if output, err := cmd.CombinedOutput(); err != nil {
			return m, fmt.Errorf("cold startup failed for sample %d: %w: %s", i+1, err, output)
		}
		m.SamplesNS = append(m.SamplesNS, float64(time.Since(start).Nanoseconds()))
	}
	sorted := append([]float64(nil), m.SamplesNS...)
	sort.Float64s(sorted)
	m.MedianNS, m.P95NS = percentile(sorted, 0.5), percentile(sorted, 0.95)
	return m, nil
}

// ScaleStartup stores a real, separately measured warm startup fixture.
type ScaleStartup struct {
	Providers int `json:"providers"`
	Startup Startup `json:"startup"`
}

// SyntheticProviders creates a reproducible set of distinct pointer-to-struct
// types; the structs intentionally have no injected fields or HTTP routes.
func SyntheticProviders(count int) (metadata.Package, error) {
	if count < 1 || count > 1000 {
		return metadata.Package{}, fmt.Errorf("provider count must be 1..1000")
	}
	p := metadata.Package{Providers: make([]metadata.Provider, 0, count)}
	for i := 0; i < count; i++ {
		structType := reflect.StructOf([]reflect.StructField{{
			Name: fmt.Sprintf("Field%04d", i),
			Type: reflect.TypeOf(int64(0)),
		}})
		p.Providers = append(p.Providers, metadata.Provider{Type: reflect.PtrTo(structType), Scope: "singleton"})
	}
	return p, nil
}
