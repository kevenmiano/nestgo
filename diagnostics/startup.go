package diagnostics

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/metadata"
	nestgoruntime "github.com/kevenmiano/nestgo/runtime"
)

// Startup stores measured warm application construction costs, not OS process boot
// time. A fixed fixture and identical environment improve comparability, but
// elapsed time is inherently nondeterministic.
type Startup struct {
	Scenario            string    `json:"scenario"`
	GoVersion           string    `json:"goVersion"`
	GOOS                string    `json:"goos"`
	GOARCH              string    `json:"goarch"`
	CPUCount            int       `json:"cpuCount"`
	Commit              string    `json:"commit,omitempty"`
	IterationsPerSample int       `json:"iterationsPerSample"`
	SamplesNS           []float64 `json:"samplesNsPerOp"`
	MedianNS            float64   `json:"medianNsPerOp"`
	P95NS               float64   `json:"p95NsPerOp"`
	BytesPerOp          int64     `json:"bytesPerOp"`
	AllocsPerOp         int64     `json:"allocsPerOp"`
}

func percentile(sorted []float64, fraction float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1)*fraction + 0.999999999)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// MeasureStartup measures runtime.New with prebuilt metadata. It does not include
// Go compilation, OS launch, imported package init(), or network listening.
// The workload excludes graph analysis, avoiding measurement contamination.
func MeasureStartup(scenario string, packages []metadata.Package, samples, iterations int, commit string) (Startup, error) {
	if samples < 1 || samples > 1000 || iterations < 1 || iterations > 100000 {
		return Startup{}, fmt.Errorf("samples and iterations must be between 1 and their limits (1000 / 100000)")
	}
	m := Startup{Scenario: scenario, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUCount: runtime.NumCPU(), Commit: commit, IterationsPerSample: iterations, SamplesNS: make([]float64, 0, samples)}
	if _, err := nestgoruntime.New(nestgoruntime.WithPackages(packages...)); err != nil {
		return m, fmt.Errorf("startup fixture invalid: %w", err)
	}
	for i := 0; i < samples; i++ {
		start := time.Now()
		for j := 0; j < iterations; j++ {
			if _, err := nestgoruntime.New(nestgoruntime.WithPackages(packages...)); err != nil {
				return m, err
			}
		}
		m.SamplesNS = append(m.SamplesNS, float64(time.Since(start).Nanoseconds())/float64(iterations))
	}
	values := append([]float64(nil), m.SamplesNS...)
	sort.Float64s(values)
	m.MedianNS = percentile(values, 0.5)
	m.P95NS = percentile(values, 0.95)
	b := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = nestgoruntime.New(nestgoruntime.WithPackages(packages...))
		}
	})
	m.BytesPerOp = b.AllocedBytesPerOp()
	m.AllocsPerOp = b.AllocsPerOp()
	return m, nil
}
