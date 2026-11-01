package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Connections owns multiple independently pooled database/sql connections.
// Registration is bootstrap-only; lookup is safe for concurrent requests.
type Connections struct {
	mu sync.RWMutex
	pools map[string]*sql.DB
	closed bool
}

func NewConnections() *Connections {
	return &Connections{pools: make(map[string]*sql.DB)}
}

// Register takes ownership of a connection. Failed registrations leave ownership
// with the caller, who must close the rejected handle.
func (c *Connections) Register(name string, db *sql.DB) error {
	if c == nil || db == nil || strings.TrimSpace(name) == "" {
		return ErrNilConnection
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed { return errors.New("database: registry closed") }
	if _, exists := c.pools[name]; exists { return fmt.Errorf("database: duplicate connection %q", name) }
	for existing, handle := range c.pools {
		if handle == db { return fmt.Errorf("database: pool already registered as %q", existing) }
	}
	c.pools[name] = db
	return nil
}

func (c *Connections) Get(name string) (*sql.DB, error) {
	if c == nil { return nil, ErrNilConnection }
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed { return nil, errors.New("database: registry closed") }
	db, ok := c.pools[name]
	if !ok { return nil, fmt.Errorf("database: unknown connection %q", name) }
	return db, nil
}

func (c *Connections) Open(ctx context.Context, options ConnectionOptions) error {
	if c == nil { return ErrNilConnection }
	if strings.TrimSpace(options.Name) == "" { return errors.New("database: connection name required") }
	c.mu.RLock()
	_, exists := c.pools[options.Name]
	closed := c.closed
	c.mu.RUnlock()
	if closed { return errors.New("database: registry closed") }
	if exists { return fmt.Errorf("database: duplicate connection %q", options.Name) }
	db, err := OpenConnection(ctx, options)
	if err != nil { return err }
	if err := c.Register(options.Name, db); err != nil {
		db.Close()
		return err
	}
	return nil
}

// Close shuts down all owned pools. It is safe to call repeatedly.
func (c *Connections) Close() error {
	if c == nil { return nil }
	c.mu.Lock()
	if c.closed { c.mu.Unlock(); return nil }
	c.closed = true
	pools := c.pools
	c.pools = nil
	c.mu.Unlock()
	var errs []error
	for name, db := range pools {
		if err := db.Close(); err != nil { errs = append(errs, fmt.Errorf("database: close %s: %w", name, err)) }
	}
	return errors.Join(errs...)
}
