package sqlx

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/kevenmiano/nestgo/database"
)

type stub struct{}

func (stub) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (stub) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (stub) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func (stub) GetContext(context.Context, any, string, ...any) error {
	return nil
}

func (stub) SelectContext(context.Context, any, string, ...any) error {
	return nil
}

func (stub) Rebind(q string) string {
	return q + " rebound"
}

func TestContract(t *testing.T) {
	a, err := New(stub{})
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Rebind("?"); got != "? rebound" {
		t.Fatalf("unexpected rebind: %s", got)
	}
	if _, err := a.Store(); err != nil {
		t.Fatal(err)
	}
	if err := a.Get(context.Background(), nil, "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if err := a.Select(context.Background(), nil, "SELECT 1"); err != nil {
		t.Fatal(err)
	}
}

func TestNil(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, database.ErrNilConnection) {
		t.Fatal(err)
	}
	var a *Adapter
	if _, err := a.Store(); !errors.Is(err, database.ErrNilConnection) {
		t.Fatal(err)
	}
}
