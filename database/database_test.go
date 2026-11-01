package database

import (
 "context"
 "database/sql"
 "errors"
 "testing"
)

func TestNilNative(t *testing.T) {
 if _, err := Native(nil); !errors.Is(err,ErrNilConnection) { t.Fatalf("expected ErrNilConnection, got %v",err) }
}
func TestNilExecutor(t *testing.T) {
 if _, err := New(nil); !errors.Is(err,ErrNilConnection) { t.Fatalf("expected ErrNilConnection, got %v",err) }
}
func TestNilBegin(t *testing.T) {
 if _, err := Begin(context.Background(),nil,nil); !errors.Is(err,ErrNilConnection) { t.Fatalf("expected ErrNilConnection, got %v",err) }
}
func TestNilTx(t *testing.T) {
 var tx *Tx
 if _, err := tx.Store(); !errors.Is(err,ErrNilConnection) { t.Fatalf("expected ErrNilConnection, got %v",err) }
}
var _ Executor = (*sql.DB)(nil)
var _ Executor = (*sql.Tx)(nil)
