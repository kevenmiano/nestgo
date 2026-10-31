package diagnostics

import (
	"os"
	"reflect"
	"testing"
)

func TestSyntheticProvidersStableAndUnique(t *testing.T) {
	for _, count := range []int{1, 10, 100} {
		left, err := SyntheticProviders(count)
		if err != nil { t.Fatal(err) }
		right, err := SyntheticProviders(count)
		if err != nil { t.Fatal(err) }
		if len(left.Providers) != count { t.Fatalf("count=%d got %d", count, len(left.Providers)) }
		for i := range left.Providers {
			if left.Providers[i].Type != right.Providers[i].Type { t.Fatalf("unstable synthetic provider type %d", i) }
			if left.Providers[i].Type.Kind() != reflect.Ptr { t.Fatalf("non-pointer provider %d", i) }
		}
		g := Analyze(left)
		if len(g.Nodes) != count || len(g.Findings) != 0 || len(g.Cycles) != 0 {
			t.Fatalf("invalid synthetic graph count=%d nodes=%d findings=%v cycles=%v", count, len(g.Nodes), g.Findings, g.Cycles)
		}
	}
	for _, bad := range []int{0, -1, 1001} {
		if _, err := SyntheticProviders(bad); err == nil { t.Fatalf("expected error count=%d", bad) }
	}
}

func TestMeasureProcessStartup(t *testing.T) {
	// Test binary acts as the trusted child and exits successfully by running
	// a known test matcher. It measures launch-to-exit, not Go compilation.
	exe, err := os.Executable()
	if err != nil { t.Fatal(err) }
	m, err := MeasureProcessStartup(exe, []string{"-test.run=^TestProcessStartupChild$"}, 3, "test-commit")
	if err != nil { t.Fatal(err) }
	if len(m.SamplesNS) != 3 || m.MedianNS <= 0 || m.P95NS < m.MedianNS {
		t.Fatalf("invalid measurements %+v", m)
	}
	if _, err := MeasureProcessStartup(exe, []string{"-test.run=^TestProcessStartupChild$"}, 0, ""); err == nil {
		t.Fatal("expected invalid sample count error")
	}
	if _, err := MeasureProcessStartup(exe, []string{"-this-flag-is-invalid"}, 1, ""); err == nil {
		t.Fatal("failed child should be reported")
	}
}

func TestProcessStartupChild(t *testing.T) {}
