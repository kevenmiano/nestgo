// Package sqlx adapts sqlx-compatible handles without forcing sqlx on native users.
package sqlx

import (
 "context"
 "database/sql"
 "errors"
 "github.com/kevenmiano/nestgo/database"
)

// Handle is satisfied by *github.com/jmoiron/sqlx.DB and *sqlx.Tx.
// The adapter is structural: importing this package does not import a driver.
type Handle interface {
 database.Executor
 GetContext(context.Context, any, string, ...any) error
 SelectContext(context.Context, any, string, ...any) error
 Rebind(string) string
}

// Adapter preserves the underlying sqlx semantics, including struct mapping.
type Adapter struct { handle Handle }
func New(handle Handle) (*Adapter, error) {
 if handle == nil { return nil, database.ErrNilConnection }
 return &Adapter{handle: handle}, nil
}
func (a *Adapter) ExecContext(ctx context.Context, q string, args ...any) (sql.Result,error) {
 return a.handle.ExecContext(ctx,q,args...)
}
func (a *Adapter) QueryContext(ctx context.Context,q string,args ...any) (*sql.Rows,error) {
 return a.handle.QueryContext(ctx,q,args...)
}
func (a *Adapter) QueryRowContext(ctx context.Context,q string,args ...any) *sql.Row {
 return a.handle.QueryRowContext(ctx,q,args...)
}
func (a *Adapter) Get(ctx context.Context,dest any,q string,args ...any) error {
 return a.handle.GetContext(ctx,dest,q,args...)
}
func (a *Adapter) Select(ctx context.Context,dest any,q string,args ...any) error {
 return a.handle.SelectContext(ctx,dest,q,args...)
}
func (a *Adapter) Rebind(q string) string { return a.handle.Rebind(q) }
func (a *Adapter) Store() (*database.Store,error) {
 if a == nil || a.handle == nil { return nil,database.ErrNilConnection }
 return database.New(a)
}
var _ database.Executor = (*Adapter)(nil)
var _ = errors.Is
