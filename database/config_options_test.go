package database

import (
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/config"
)

func TestWithConnectionFromConfig(t *testing.T) {
	t.Setenv("ORDERS_DB_DRIVER", "postgres")
	t.Setenv("ORDERS_DB_DSN", "private-dsn")
	t.Setenv("ORDERS_DB_MAX_OPEN_CONNS", "8")
	t.Setenv("ORDERS_DB_MAX_IDLE_CONNS", "2")
	settings, err := config.New(config.WithIgnoreEnvFile())
	if err != nil {
		t.Fatal(err)
	}
	module, err := NewModule(WithConnectionFromConfig("orders", settings, "ORDERS_DB"))
	if err != nil {
		t.Fatal(err)
	}
	if len(module.options.Connections) != 1 {
		t.Fatal("expected one named connection")
	}
	conn := module.options.Connections[0]
	if conn.Name != "orders" || conn.Driver != "postgres" || conn.MaxOpenConns != 8 || conn.MaxIdleConns != 2 {
		t.Fatalf("invalid resolved options: name=%q driver=%q maxOpen=%d maxIdle=%d", conn.Name, conn.Driver, conn.MaxOpenConns, conn.MaxIdleConns)
	}
}

func TestWithConnectionFromConfigFailsBeforeOpening(t *testing.T) {
	t.Setenv("ORDERS_DB_DRIVER", "postgres")
	t.Setenv("ORDERS_DB_DSN", "very-private-dsn")
	t.Setenv("ORDERS_DB_MAX_OPEN_CONNS", "2")
	t.Setenv("ORDERS_DB_MAX_IDLE_CONNS", "9")
	settings, err := config.New(config.WithIgnoreEnvFile())
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewModule(WithConnectionFromConfig("orders", settings, "ORDERS_DB"))
	if err == nil || !strings.Contains(err.Error(), "idle limit") {
		t.Fatalf("expected validation failure: %v", err)
	}
	if strings.Contains(err.Error(), "very-private-dsn") {
		t.Fatal("configuration error leaked DSN")
	}
}
