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
	good := "package sample\n@Injectable()\ntype Repository struct{}\n"
	if err := os.WriteFile(source, []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reports := make(chan error, 16)
	done := make(chan error, 1)
	go func() {
		done <- WatchDir(ctx, dir, 15*time.Millisecond, func(err error) { reports <- err })
	}()
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
	if err := os.WriteFile(source, []byte(good+"\n@Injectable()\ntype Service struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := receive(); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(generated); err != nil || !strings.Contains(string(raw), "type Service struct") {
		t.Fatalf("source did not regenerate: %v %s", err, raw)
	}
	invalid := "package sample\n@Injectable()\ntype Invalid struct {\n"
	if err := os.WriteFile(source, []byte(invalid), 0644); err != nil {
		t.Fatal(err)
	}
	if err := receive(); err == nil {
		t.Fatal("expected diagnostic for invalid source")
	}
	if err := os.WriteFile(source, []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	if err := receive(); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
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
