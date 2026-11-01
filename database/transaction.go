package database

import (
 "context"
 "database/sql"
 "errors"
)

// WithinTransaction runs fn with an explicit transactional Store and guarantees
// rollback on failures or panics. Commit errors are returned to the caller.
func WithinTransaction(ctx context.Context, db *sql.DB, opts *sql.TxOptions, fn func(*Store) error) (err error) {
 if fn == nil { return errors.New("database: nil transaction callback") }
 tx,err:=Begin(ctx,db,opts)
 if err!=nil {return err}
 defer tx.Rollback() // safe after Commit; also runs during panic unwinding
 store,err:=tx.Store()
 if err!=nil {return err}
 if err=fn(store);err!=nil {return err}
 return tx.Commit()
}
