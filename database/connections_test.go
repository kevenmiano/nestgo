package database

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func TestConnectionsValidationAndShutdown(t *testing.T) {
	registry := NewConnections()
	if _, err := registry.Get("missing"); err == nil {
		t.Fatal("unknown connection should fail")
	}
	if err := registry.Register("", &sql.DB{}); err == nil {
		t.Fatal("empty name should fail")
	}
	if err := registry.Register("primary", nil); err == nil {
		t.Fatal("nil pool should fail")
	}
	// We do not call sql.DB methods: these handles are only used to verify
	// ownership and lookup; real pool lifecycle is exercised in integration tests.
	db := new(sql.DB)
	if err := registry.Register("primary", db); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register("primary", new(sql.DB)); err == nil {
		t.Fatal("duplicate should fail")
	}
	if err := registry.Register("alias", db); err == nil {
		t.Fatal("same pool alias must fail")
	}
	if got, err := registry.Get("primary"); err != nil || got != db {
		t.Fatalf("lookup: %v", err)
	}
	// Avoid closing the zero-value sql.DB; it is not a valid initialized pool.
	registry.mu.Lock()
	registry.pools = map[string]*sql.DB{}
	registry.mu.Unlock()
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register("later", new(sql.DB)); err == nil {
		t.Fatal("registration after close should fail")
	}
	if _, err := registry.Get("primary"); err == nil {
		t.Fatal("get after close should fail")
	}
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestConnectionsOpenRejectsInvalidOptions(t *testing.T) {
	registry := NewConnections()
	if err := registry.Open(context.Background(), ConnectionOptions{}); err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected name validation: %v", err)
	}
}
