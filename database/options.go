package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Option configures a database module descriptor before connections are opened.
type Option func(*Options) error

// ConnectionOption configures one named connection.
type ConnectionOption func(*ConnectionOptions) error

// Options is an immutable-by-convention bootstrap plan after New returns.
type Options struct {
	Connections []ConnectionOptions
}

// New builds a connection descriptor without opening sockets or reading env.
// Call Open to validate connectivity and start the owned connection pools.
func New(opts ...Option) (*Module, error) {
	plan := Options{}
	for _, apply := range opts {
		if apply == nil {
			return nil, errors.New("database: nil option")
		}
		if err := apply(&plan); err != nil {
			return nil, err
		}
	}
	seen := make(map[string]bool, len(plan.Connections))
	for _, conn := range plan.Connections {
		if strings.TrimSpace(conn.Name) == "" || seen[conn.Name] {
			return nil, fmt.Errorf("database: empty or duplicate connection name %q", conn.Name)
		}
		seen[conn.Name] = true
	}
	return &Module{options: plan}, nil
}

// Module represents named connections registered at startup.
type Module struct {
	options Options
}

func (m *Module) Open(ctx context.Context) (*Connections, error) {
	if m == nil {
		return nil, errors.New("database: nil module")
	}
	return Bootstrap(ctx, append([]ConnectionOptions(nil), m.options.Connections...))
}

func WithConnection(name string, opts ...ConnectionOption) Option {
	return func(plan *Options) error {
		conn := ConnectionOptions{Name: name, MaxIdleConns: -1}
		for _, apply := range opts {
			if apply == nil {
				return errors.New("database: nil connection option")
			}
			if err := apply(&conn); err != nil {
				return err
			}
		}
		plan.Connections = append(plan.Connections, conn)
		return nil
	}
}

func WithDriver(driver string) ConnectionOption {
	return func(c *ConnectionOptions) error {
		if driver == "" {
			return errors.New("database: driver required")
		}
		c.Driver = driver
		return nil
	}
}

func WithDSN(dsn string) ConnectionOption {
	return func(c *ConnectionOptions) error {
		if dsn == "" {
			return errors.New("database: DSN required")
		}
		c.DSN = dsn
		return nil
	}
}

func WithMaxOpenConns(n int) ConnectionOption {
	return func(c *ConnectionOptions) error {
		if n < 1 {
			return errors.New("database: max open connections must be positive")
		}
		c.MaxOpenConns = n
		return nil
	}
}

func WithMaxIdleConns(n int) ConnectionOption {
	return func(c *ConnectionOptions) error {
		if n < 0 {
			return errors.New("database: max idle connections cannot be negative")
		}
		c.MaxIdleConns = n
		return nil
	}
}

func WithPingTimeout(timeout time.Duration) ConnectionOption {
	return func(c *ConnectionOptions) error {
		if timeout <= 0 {
			return errors.New("database: ping timeout must be positive")
		}
		c.PingTimeout = timeout
		return nil
	}
}
