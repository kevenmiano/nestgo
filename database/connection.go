package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ConnectionOptions configures a standard Go SQL driver without depending on
// a specific database, ORM, query builder or cloud provider.
type ConnectionOptions struct {
	Name string
	Driver string
	DSN string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout time.Duration
}

// OpenConnection creates a pooled connection and verifies it before exposing it
// to DI. The caller owns the returned *sql.DB and must close it at shutdown.
func OpenConnection(ctx context.Context, options ConnectionOptions) (*sql.DB, error) {
	if options.Driver == "" || options.DSN == "" {
		return nil, errors.New("database: driver and DSN required")
	}
	db, err := sql.Open(options.Driver, options.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: open connection: %w", err)
	}
	if options.MaxOpenConns > 0 {
		db.SetMaxOpenConns(options.MaxOpenConns)
	}
	if options.MaxIdleConns >= 0 {
		db.SetMaxIdleConns(options.MaxIdleConns)
	}
	if options.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(options.ConnMaxLifetime)
	}
	if options.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(options.ConnMaxIdleTime)
	}
	if options.PingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.PingTimeout)
		defer cancel()
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database: ping %q failed: %w", options.Name, err)
	}
	return db, nil
}
