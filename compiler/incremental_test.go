package compiler

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

func writeIncrementalFixture(t testing.TB, dir string, count int) []string {
	t.Helper()
	paths := make([]string, count)
	for i := range paths {
		paths[i] = filepath.Join(dir, fmt.Sprintf("provider_%03d.mgo", i))
		source := fmt.Sprintf("package sample\n@Injectable()\ntype Provider%03d struct{}\n", i)
		if err := os.WriteFile(paths[i], []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return paths
}

func incrementalCompile(dir string, cache *incrementalCache) error {
	return compileDirWithCache(dir, sqlgen.PostgreSQL, nil, false, cache)
}

func snapshotGenerated(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := map[string][]byte{}
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), "_generated.go") &&
			!strings.HasPrefix(entry.Name(), "zz_modforge_") &&
			entry.Name() != "zz_nestgo_wiring_generated_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		snapshot[entry.Name()] = raw
	}
	return snapshot
}

func assertSameGenerated(t *testing.T, before, after map[string][]byte) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("different artifact counts: incremental=%d full=%d", len(before), len(after))
	}
	for name, content := range before {
		if !bytes.Equal(content, after[name]) {
			t.Fatalf("incremental output differs from clean generation: %s", name)
		}
	}
}

func TestIncrementalCacheReusesOnlyUnchangedSources(t *testing.T) {
	dir := t.TempDir()
	paths := writeIncrementalFixture(t, dir, 3)
	cache := &incrementalCache{}
	if err := incrementalCompile(dir, cache); err != nil {
		t.Fatal(err)
	}
	if cache.parsed != 3 || cache.reused != 0 || cache.rendered != 3 {
		t.Fatalf("cold compilation counters: %+v", cache)
	}
	if err := incrementalCompile(dir, cache); err != nil {
		t.Fatal(err)
	}
	if cache.parsed != 3 || cache.reused != 3 || cache.rendered != 3 {
		t.Fatalf("unchanged units were recompiled: %+v", cache)
	}
	// Same-length edit must invalidate precisely one source unit.
	if err := os.WriteFile(paths[1], []byte("package sample\n@Injectable()\ntype Provider001 struct{}\n// changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := incrementalCompile(dir, cache); err != nil {
		t.Fatal(err)
	}
	if cache.parsed != 4 || cache.reused != 5 || cache.rendered != 4 {
		t.Fatalf("edited unit invalidated unrelated units: %+v", cache)
	}
	before := snapshotGenerated(t, dir)
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	assertSameGenerated(t, before, snapshotGenerated(t, dir))
	if err := CheckDir(dir); err != nil {
		t.Fatalf("incremental generated artifacts failed freshness check: %v", err)
	}
}

func TestIncrementalCacheTracksAdditionDeletionAndRecovery(t *testing.T) {
	dir := t.TempDir()
	paths := writeIncrementalFixture(t, dir, 2)
	cache := &incrementalCache{}
	if err := incrementalCompile(dir, cache); err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(dir, "new.mgo")
	good := "package sample\n@Injectable()\ntype Added struct{}\n"
	if err := os.WriteFile(added, []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	if err := incrementalCompile(dir, cache); err != nil {
		t.Fatal(err)
	}
	if cache.parsed != 3 || cache.reused != 2 {
		t.Fatalf("adding a source did not reuse existing units: %+v", cache)
	}
	// Broken edits must not replace the last successfully compiled unit.
	if err := os.WriteFile(added, []byte("package sample\n@Injectable()\ntype Broken struct {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := incrementalCompile(dir, cache); err == nil {
		t.Fatal("invalid change compiled successfully")
	}
	if len(cache.units) != 3 {
		t.Fatal("failed compile poisoned committed cache")
	}
	if err := os.WriteFile(added, []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	reuseBefore := cache.reused
	if err := incrementalCompile(dir, cache); err != nil {
		t.Fatal(err)
	}
	if cache.reused != reuseBefore+3 {
		t.Fatalf("repaired source not reused from last good snapshot: %+v", cache)
	}
	if err := os.Remove(paths[0]); err != nil {
		t.Fatal(err)
	}
	if err := incrementalCompile(dir, cache); err != nil {
		t.Fatal(err)
	}
	if len(cache.units) != 2 {
		t.Fatalf("deleted source remains cached: %d", len(cache.units))
	}
	if _, err := os.Stat(strings.TrimSuffix(paths[0], ".mgo") + "_generated.go"); !os.IsNotExist(err) {
		t.Fatalf("removed source left stale generated Go: %v", err)
	}
	before := snapshotGenerated(t, dir)
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	assertSameGenerated(t, before, snapshotGenerated(t, dir))
}

func TestIncrementalCacheDetectsRepositoryAnnotationRemoval(t *testing.T) {
	// Directly verify cleanup classification without requiring a database-backed
	// repository fixture: only files previously marked @Repository may be removed.
	previous := &File{Declarations: []Declaration{{Annotations: []Annotation{{Name: "Repository"}}}}}
	current := &File{Declarations: []Declaration{{Annotations: []Annotation{{Name: "Injectable"}}}}}
	if !isRepositoryUnit(previous) || isRepositoryUnit(current) {
		t.Fatal("repository classification changed")
	}
}

func BenchmarkCompileDirColdVsIncremental(b *testing.B) {
	for _, count := range []int{10, 50} {
		for _, incremental := range []bool{false, true} {
			label := "full"
			if incremental {
				label = "incremental"
			}
			b.Run(fmt.Sprintf("%s_%d", label, count), func(b *testing.B) {
				dir := b.TempDir()
				paths := writeIncrementalFixture(b, dir, count)
				cache := &incrementalCache{}
				if err := incrementalCompile(dir, cache); err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					edit := fmt.Sprintf("package sample\n@Injectable()\ntype Provider000 struct{}\n// revision %d\n", i%2)
					if err := os.WriteFile(paths[0], []byte(edit), 0644); err != nil {
						b.Fatal(err)
					}
					b.StartTimer()
					var err error
					if incremental {
						err = incrementalCompile(dir, cache)
					} else {
						err = CompileDir(dir)
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
