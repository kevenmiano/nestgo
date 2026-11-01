package database

import (
	"context"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/config"
)

func TestBootstrapFromConfigValidatesBeforeOpening(t *testing.T) {
	t.Setenv("USERS_DB_DRIVER", "postgres")
	t.Setenv("USERS_DB_DSN", "example-secret")
	t.Setenv("USERS_DB_MAX_OPEN_CONNS", "2")
	t.Setenv("USERS_DB_MAX_IDLE_CONNS", "5")
	cfg, err := config.Load(config.Options{IgnoreEnvFile: true})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := BootstrapFromConfig(context.Background(), cfg, []ConnectionEnv{{Name: "users", Prefix: "USERS_DB"}})
	if err == nil || registry != nil || !strings.Contains(err.Error(), "idle limit") {
		t.Fatalf("expected pool validation failure, got %v", err)
	}
	if strings.Contains(err.Error(), "example-secret") {
		t.Fatal("DSN leaked in error")
	}
}

func TestBootstrapFromConfigRequiredEnv(t *testing.T) {
	cfg, err := config.Load(config.Options{IgnoreEnvFile: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = BootstrapFromConfig(context.Background(), cfg, []ConnectionEnv{{Name: "missing", Prefix: "UNLIKELY_MISSING_DB_TEST"}})
	if err == nil || !strings.Contains(err.Error(), "UNLIKELY_MISSING_DB_TEST_DRIVER") {
		t.Fatalf("expected missing environment diagnostic, got %v", err)
	}
}
