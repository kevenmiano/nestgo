package integrationtests

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	nestdb "github.com/kevenmiano/nestgo/database"
	_ "github.com/lib/pq"
)

func TestNamedModuleRealConnections(t *testing.T) {
	pg := os.Getenv("NESTGO_TEST_POSTGRES_DSN")
	my := os.Getenv("NESTGO_TEST_MYSQL_DSN")
	if pg == "" || my == "" {
		t.Skip("database DSNs are required by CI")
	}
	module, err := nestdb.NewModule(
		nestdb.WithConnection("users", nestdb.WithDriver("postgres"), nestdb.WithDSN(pg), nestdb.WithMaxOpenConns(3), nestdb.WithMaxIdleConns(2), nestdb.WithPingTimeout(5*time.Second)),
		nestdb.WithConnection("billing", nestdb.WithDriver("mysql"), nestdb.WithDSN(my), nestdb.WithMaxOpenConns(3), nestdb.WithMaxIdleConns(2), nestdb.WithPingTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	registry, err := module.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	users, err := registry.Get("users")
	if err != nil {
		t.Fatal(err)
	}
	billing, err := registry.Get("billing")
	if err != nil {
		t.Fatal(err)
	}
	if users == billing {
		t.Fatal("named connections must use different pools")
	}
	for _, db := range []*struct {
		name string
		pool interface{ PingContext(context.Context) error }
	}{{"users", users}, {"billing", billing}} {
		if err := db.pool.PingContext(ctx); err != nil {
			t.Fatalf("%s ping: %v", db.name, err)
		}
	}
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Get("users"); err == nil {
		t.Fatal("closed registry should not resolve pools")
	}
}
