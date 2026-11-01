package compiler

import (
	"crypto/sha256"
)

// cachedUnit owns one immutable parsed source and its local generated Go code.
// Package-wide metadata, SQL registrations, DTO codec code and editable stubs
// are deliberately revalidated/regenerated each run: these may depend on other
// files, dialects or mutable non-.mgo Go sources.
type cachedUnit struct {
	digest    [sha256.Size]byte
	file      *File
	generated []byte
}

// incrementalCache is owned by one watcher; it has no global state, persistent
// artifacts, cross-directory reuse or impact on standalone compile/check.
type incrementalCache struct {
	units map[string]cachedUnit
	// counters make cache behavior testable without timing-based assertions.
	parsed   int
	reused   int
	rendered int
}

func (c *incrementalCache) parse(path string, raw []byte, next map[string]cachedUnit) (*File, error) {
	digest := sha256.Sum256(raw)
	if cached, ok := c.units[path]; ok && cached.digest == digest {
		c.reused++
		next[path] = cached
		return cached.file, nil
	}
	file, err := Parse(path, raw)
	if err != nil {
		return nil, err
	}
	c.parsed++
	next[path] = cachedUnit{digest: digest, file: file}
	return file, nil
}

func (c *incrementalCache) render(file *File, next map[string]cachedUnit) ([]byte, error) {
	unit := next[file.Path]
	if unit.generated != nil {
		return unit.generated, nil
	}
	rendered, err := generatedSource(file)
	if err != nil {
		return nil, err
	}
	c.rendered++
	unit.generated = rendered
	next[file.Path] = unit
	return rendered, nil
}

// Commit only after the entire compile succeeds. Errors in any source or in
// package-wide validation cannot poison the last known-good snapshot. Files
// deleted from the directory disappear from the committed cache automatically.
func (c *incrementalCache) commit(next map[string]cachedUnit) { c.units = next }
