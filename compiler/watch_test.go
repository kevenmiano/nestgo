package compiler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSourceFingerprintDetectsSameSizeEditsAndIgnoresGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "dto.mgo")
	if err := os.WriteFile(source, []byte("package sample\n"), 0644); err != nil {
		t.Fatal(err)
	}
	first, err := sourceFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dto_generated.go"), []byte("generated"), 0644); err != nil {
		t.Fatal(err)
	}
	next, err := sourceFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if next != first {
		t.Fatal("generated Go file changed the source fingerprint")
	}
	if err := os.WriteFile(source, []byte("package SAMPLE\n"), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := sourceFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("same-length source change went unnoticed")
	}
}

func TestWatchDirRecompilesOnChangesAndRecoversFromInvalidSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "users.mgo")
	// Replace sources atomically: editors commonly save via rename and a
	// truncate-and-write sequence may expose transient, incomplete Go syntax.
	// The watcher is tested against complete, stable source revisions here.
	writeSource := func(contents string) {
		t.Helper()
		staged := source + ".tmp"
		if err := os.WriteFile(staged, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(staged, source); err != nil {
			t.Fatal(err)
		}
	}
	good := "package sample\n@Injectable()\ntype Repository struct{}\n"
	writeSource(good)
	ctx, cancel := context.WithCancel(context.Background())
	reports := make(chan error, 16)
	done := make(chan struct{})
	var watchErr error
	go func() {
		defer close(done)
		watchErr = WatchDir(ctx, dir, 15*time.Millisecond, func(err error) {
			select {
			case reports <- err:
			case <-ctx.Done():
			}
		})
	}()
	// t.Fatal also runs Cleanup; never remove the temp directory while
	// WatchDir may still be reading/writing generated files.
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
			if watchErr != nil {
				t.Errorf("watch exited: %v", watchErr)
			}
		case <-time.After(3 * time.Second):
			t.Error("watch did not exit during test cleanup")
		}
	})
	receive := func() error {
		t.Helper()
		select {
		case err := <-reports:
			return err
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for compile")
			return nil
		}
	}
	if err := receive(); err != nil {
		t.Fatal(err)
	}
	// Edits to compiler-owned output do not recursively retrigger the watch.
	generated := filepath.Join(dir, "users_generated.go")
	if err := os.WriteFile(generated, []byte("modified by editor"), 0644); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-reports:
		t.Fatalf("generated output triggered watch: %v", err)
	case <-time.After(70 * time.Millisecond):
	}
	writeSource(good + "\n@Injectable()\ntype Service struct{}\n")
	if err := receive(); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(generated); err != nil || !strings.Contains(string(raw), "type Service struct") {
		t.Fatalf("source did not regenerate: %v %s", err, raw)
	}
	invalid := "package sample\n@Injectable()\ntype Invalid struct {\n"
	writeSource(invalid)
	if err := receive(); err == nil {
		t.Fatal("expected diagnostic for invalid source")
	}
	writeSource(good)
	if err := receive(); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-done:
		if watchErr != nil {
			t.Fatal(watchErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not exit on cancellation")
	}
}

func TestWatchDirRejectsInvalidInterval(t *testing.T) {
	if err := WatchDir(context.Background(), t.TempDir(), 0, nil); err == nil {
		t.Fatal("expected invalid interval error")
	}
}

func BenchmarkSourceFingerprint(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 25; i++ {
		path := filepath.Join(dir, strings.Repeat("a", i+1)+".mgo")
		if err := os.WriteFile(path, []byte("package demo\n"), 0644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sourceFingerprint(dir); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSourceFingerprintTracksHandwrittenGoDependencies(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dto.mgo"), []byte("package sample\n"), 0644); err != nil {
		t.Fatal(err)
	}
	before, err := sourceFingerprint(dir)
	if err != nil {
		t.Fatal(err)
	}
	goSource := filepath.Join(dir, "helpers.go")
	if err := os.WriteFile(goSource, []byte("package sample\ntype Helper struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := sourceFingerprint(dir)
	if err != nil || changed == before {
		t.Fatalf("new handwritten Go file was ignored: %v", err)
	}
	for _, ignored := range []string{"helpers_generated.go", "zz_modforge_metadata.go", "zz_nestgo_stubs_test.go"} {
		if err := os.WriteFile(filepath.Join(dir, ignored), []byte("generated"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	ignored, err := sourceFingerprint(dir)
	if err != nil || ignored != changed {
		t.Fatalf("generated/test output retriggered fingerprint: %v", err)
	}
	if err := os.WriteFile(goSource, []byte("package sample\ntype Helper struct{ Name string }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	updated, err := sourceFingerprint(dir)
	if err != nil || updated == changed {
		t.Fatalf("edited handwritten Go source was not detected: %v", err)
	}
}
