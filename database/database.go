// Package database defines a small SQL-first persistence boundary.
// It deliberately does not implement an identity map, change tracking, or an ORM.
package database

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
)

var ErrNilConnection = errors.New("database: nil connection")

// Executor is the minimum contract supported by database/sql, sqlx and transactions.
type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Store is a SQL execution boundary. Queries remain explicit and parameterized.
type Store struct {
	executor Executor
}

func New(executor Executor) (*Store, error) {
	if isNil(executor) {
		return nil, ErrNilConnection
	}
	return &Store{executor: executor}, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func (s *Store) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.executor.ExecContext(ctx, query, args...)
}

func (s *Store) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.executor.QueryContext(ctx, query, args...)
}

func (s *Store) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.executor.QueryRowContext(ctx, query, args...)
}

// Native wraps *sql.DB without changing pooling, driver, or transaction semantics.
func Native(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, ErrNilConnection
	}
	return New(db)
}

// Begin starts a caller-owned transaction. Always Commit or Rollback the returned Tx.
func Begin(ctx context.Context, db *sql.DB, opts *sql.TxOptions) (*Tx, error) {
	if db == nil {
		return nil, ErrNilConnection
	}
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

type Tx struct {
	*sql.Tx
}

func (t *Tx) Store() (*Store, error) {
	if t == nil || t.Tx == nil {
		return nil, ErrNilConnection
	}
	return New(t.Tx)
}
