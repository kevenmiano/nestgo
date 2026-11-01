package compiler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

// WatchDir compiles immediately and again when top-level .mgo files change.
// Diagnostics are reported without stopping the watcher, so fixing invalid
// source triggers a fresh compilation. Generated .go files are never watched.
func WatchDir(ctx context.Context, dir string, interval time.Duration, report func(error)) error {
	return watchDir(ctx, dir, interval, report, CompileDir)
}

// WatchDirWithDialects retains the named per-connection SQL dialect mapping on
// every recompile, instead of falling back silently to PostgreSQL.
func WatchDirWithDialects(ctx context.Context, dir string, interval time.Duration, dialects map[string]sqlgen.Dialect, report func(error)) error {
	return watchDir(ctx, dir, interval, report, func(dir string) error {
		return CompileDirWithDialects(dir, dialects)
	})
}

func watchDir(ctx context.Context, dir string, interval time.Duration, report func(error), compile func(string) error) error {
	if interval <= 0 {
		return fmt.Errorf("watch interval must be positive")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	current, err := sourceFingerprint(dir)
	if err != nil {
		return err
	}
	if report != nil {
		report(compile(dir))
	} else if err := compile(dir); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			next, err := sourceFingerprint(dir)
			if err != nil {
				if report != nil {
					report(err)
				}
				continue
			}
			if next == current {
				continue
			}
			current = next
			if err := compile(dir); report != nil {
				report(err)
			} else if err != nil {
				return err
			}
		}
	}
}

// sourceFingerprint includes file names and their contents, not modtimes:
// fast atomic saves and same-size edits must both trigger regeneration.
func sourceFingerprint(dir string) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	entries, err := os.ReadDir(dir)
	if err != nil {
		return zero, err
	}
	h := sha256.New()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mgo") {
			continue
		}
		name := entry.Name()
		if _, err := io.WriteString(h, fmt.Sprintf("%d:%s:", len(name), name)); err != nil {
			return zero, err
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return zero, err
		}
		if _, err := io.WriteString(h, fmt.Sprintf("%d:", len(data))); err != nil {
			return zero, err
		}
		if _, err := h.Write(data); err != nil {
			return zero, err
		}
	}
	return finishFingerprint(h), nil
}

func finishFingerprint(h hash.Hash) [sha256.Size]byte {
	var sum [sha256.Size]byte
	copy(sum[:], h.Sum(nil))
	return sum
}
