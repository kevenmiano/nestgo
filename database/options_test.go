package database

import (
	"strings"
	"testing"
	"time"
)

func TestNewFunctionalOptions(t *testing.T) {
	module, err := New(
		WithConnection("users", WithDriver("postgres"), WithDSN("postgres://localhost/users"), WithMaxOpenConns(20), WithMaxIdleConns(5), WithPingTimeout(3*time.Second)),
		WithConnection("billing", WithDriver("mysql"), WithDSN("billing-dsn")),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.options.Connections) != 2 {
		t.Fatalf("expected two connections, got %d", len(module.options.Connections))
	}
	first := module.options.Connections[0]
	if first.Name != "users" || first.Driver != "postgres" || first.MaxOpenConns != 20 || first.MaxIdleConns != 5 {
		t.Fatalf("unexpected configuration: %+v", first)
	}
}

func TestNewFunctionalOptionsInvalid(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
	}{
		{"nil option", []Option{nil}},
		{"duplicate", []Option{WithConnection("users"), WithConnection("users")}},
		{"missing name", []Option{WithConnection("")}},
		{"bad pool", []Option{WithConnection("users", WithMaxOpenConns(0))}},
		{"bad timeout", []Option{WithConnection("users", WithPingTimeout(0))}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.opts...); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewRejectsNilModuleBootstrap(t *testing.T) {
	var module *Module
	_, err := module.Open(nil)
	if err == nil || !strings.Contains(err.Error(), "nil module") {
		t.Fatalf("unexpected error: %v", err)
	}
}
