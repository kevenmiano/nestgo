package database

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestConnectionFactoryDeferredUntilOpen(t *testing.T) {
	calls := 0
	module, err := NewModule(WithConnectionFactory("users", func(ctx context.Context) (ConnectionOptions, error) {
		calls++
		return ConnectionOptions{Driver: "invalid-driver", DSN: "not-a-real-dsn"}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("factory executed during construction")
	}
	_, err = module.Open(context.Background())
	if calls != 1 || err == nil {
		t.Fatalf("factory calls=%d open err=%v", calls, err)
	}
}

func TestConnectionFactoryValidation(t *testing.T) {
	_, err := NewModule(WithConnection("same"), WithConnectionFactory("same", func(context.Context) (ConnectionOptions, error) {
		return ConnectionOptions{}, nil
	}))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate name diagnostic: %v", err)
	}
	_, err = NewModule(WithConnectionFactory("users", nil))
	if err == nil {
		t.Fatal("nil factory accepted")
	}
	module, err := NewModule(WithConnectionFactory("users", func(context.Context) (ConnectionOptions, error) {
		return ConnectionOptions{Name: "wrong"}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = module.Open(context.Background())
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected name mismatch: %v", err)
	}
}

func TestConnectionFactoryError(t *testing.T) {
	sentinel := errors.New("backend unavailable")
	module, err := NewModule(WithConnectionFactory("users", func(context.Context) (ConnectionOptions, error) {
		return ConnectionOptions{}, sentinel
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = module.Open(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("factory error not propagated: %v", err)
	}
}
