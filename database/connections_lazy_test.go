package database

import (
	"database/sql"
	"testing"
)

func TestConnectionsAllocatesRegistryOnlyOnFirstRegistration(t *testing.T) {
	connections := NewConnections()
	if connections.pools != nil {
		t.Fatal("HTTP-only startup should not allocate a database pool map")
	}
	if _, err := connections.Get("missing"); err == nil {
		t.Fatal("missing connection must return an error")
	}
	if err := connections.Register("", new(sql.DB)); err == nil {
		t.Fatal("empty name must be rejected")
	}
	if err := connections.Register("bad", nil); err == nil {
		t.Fatal("nil database must be rejected")
	}
	if connections.pools != nil {
		t.Fatal("rejected operations must not allocate pool map")
	}

	// The zero value is usable, and its first valid registration initializes
	// the map under the same lock used by concurrent readers and writers.
	var zero Connections
	pool := new(sql.DB)
	if err := zero.Register("primary", pool); err != nil {
		t.Fatal(err)
	}
	if zero.pools == nil {
		t.Fatal("first valid registration did not allocate pool map")
	}
	if got, err := zero.Get("primary"); err != nil || got != pool {
		t.Fatalf("lookup: got %v, error %v", got, err)
	}
	if err := zero.Register("alias", pool); err == nil {
		t.Fatal("duplicate pool registered under a second name")
	}
	// sql.DB zero values must not be closed; actual handles are tested by
	// the existing PostgreSQL and MySQL integration suites.
	zero.mu.Lock()
	zero.pools = nil
	zero.mu.Unlock()
	if err := zero.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zero.Register("after-close", new(sql.DB)); err == nil {
		t.Fatal("registration after close should fail")
	}
}

func TestEmptyConnectionRegistryCloseIsIdempotent(t *testing.T) {
	connections := NewConnections()
	if err := connections.Close(); err != nil {
		t.Fatal(err)
	}
	if err := connections.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := connections.Get("anything"); err == nil {
		t.Fatal("closed empty pool registry must reject lookup")
	}
}
