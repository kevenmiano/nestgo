package database

import (
	"context"
	"strings"
	"testing"
)

func TestBootstrapRejectsDuplicatesBeforeOpening(t *testing.T) {
	registry, err := Bootstrap(context.Background(), []ConnectionOptions{
		{Name: "users", Driver: "nonexistent"},
		{Name: "users", Driver: "nonexistent"},
	})
	if registry != nil || err == nil {
		t.Fatalf("expected failure without registry, got %v %v", registry, err)
	}
	// Driver errors must not include the DSN or credentials in diagnostics.
	if strings.Contains(err.Error(), "password=") {
		t.Fatal("credentials leaked")
	}
}

func TestBootstrapEmptyIsValid(t *testing.T) {
	registry, err := Bootstrap(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
}
