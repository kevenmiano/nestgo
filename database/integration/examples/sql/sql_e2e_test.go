package sqlsample

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kevenmiano/nestgo/database"
	"github.com/kevenmiano/nestgo/platform/nethttp"
	"github.com/kevenmiano/nestgo/runtime"
	_ "github.com/lib/pq"
)

// The database-integration workflow generates this package before running its
// tests. This exercises real compiled repositories, DI and both SQL engines.
func TestCompiledSQLRepositoriesEndToEnd(t *testing.T) {
	postgresDSN := os.Getenv("NESTGO_TEST_POSTGRES_DSN")
	mysqlDSN := os.Getenv("NESTGO_TEST_MYSQL_DSN")
	if postgresDSN == "" || mysqlDSN == "" {
		t.Skip("real PostgreSQL and MySQL DSNs required; mandatory in database CI")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	app, err := runtime.New(
		runtime.WithAdapter(nethttp.New()),
		runtime.WithContext(ctx),
		runtime.WithDatabase(
			database.WithConnection("users",
				database.WithDriver("postgres"),
				database.WithDSN(postgresDSN),
				database.WithMaxOpenConns(3),
			),
			database.WithConnection("billing",
				database.WithDriver("mysql"),
				database.WithDSN(mysqlDSN),
				database.WithMaxOpenConns(3),
			),
		),
	)
	if err != nil {
		t.Fatalf("bootstrap generated SQL providers and module: %v", err)
	}
	defer app.Close()

	pgDB, err := app.Connections.Get("users")
	if err != nil { t.Fatal(err) }
	myDB, err := app.Connections.Get("billing")
	if err != nil { t.Fatal(err) }
	serveDone := make(chan error, 1)
	serveStarted := false
	defer func() {
		for _, db := range []*sql.DB{pgDB, myDB} {
			if _, err := db.ExecContext(context.Background(), "DELETE FROM nestgo_sql_example_users"); err != nil {
				t.Errorf("cleanup SQL test table: %v", err)
			}
		}
		if serveStarted {
			stopCtx, cancelStop := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelStop()
			if err := app.Shutdown(stopCtx); err != nil { t.Errorf("shutdown: %v", err) }
			select {
			case <-serveDone:
			case <-time.After(3*time.Second): t.Error("HTTP server did not stop")
			}
		}
	}()
	for _, db := range []*sql.DB{pgDB, myDB} {
		if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS nestgo_sql_example_users (id BIGINT PRIMARY KEY, name VARCHAR(255) NOT NULL)"); err != nil { t.Fatal(err) }
		if _, err := db.ExecContext(ctx, "DELETE FROM nestgo_sql_example_users"); err != nil { t.Fatal(err) }
	}

	pg, err := NewPostgresUsersRepositoryGeneratedFromConnections(app.Connections)
	if err != nil { t.Fatal(err) }
	my, err := NewMySQLUsersRepositoryGeneratedFromConnections(app.Connections)
	if err != nil { t.Fatal(err) }
	const payload = "' OR 1=1; DROP TABLE nestgo_sql_example_users; --"
	if affected, err := pg.Insert(ctx, 101, payload); err != nil || affected != 1 {
		t.Fatalf("postgres INSERT affected=%d err=%v", affected, err)
	}
	if affected, err := my.Insert(ctx, 202, payload); err != nil || affected != 1 {
		t.Fatalf("mysql INSERT affected=%d err=%v", affected, err)
	}

	pgUser, err := pg.FindByID(ctx, 101)
	if err != nil || pgUser.ID != 101 || pgUser.Name != payload {
		t.Fatalf("postgres SELECT reordered projection: user=%+v err=%v", pgUser, err)
	}
	myUser, err := my.FindByID(ctx, 202)
	if err != nil || myUser.ID != 202 || myUser.Name != payload {
		t.Fatalf("mysql SELECT reordered projection: user=%+v err=%v", myUser, err)
	}
	if all, err := pg.FindAll(ctx); err != nil || len(all) != 1 { t.Fatalf("postgres list=%+v err=%v", all, err) }
	if all, err := my.FindAll(ctx); err != nil || len(all) != 1 { t.Fatalf("mysql list=%+v err=%v", all, err) }

	// A real request flows controller -> service -> generated repository -> SQL.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer listener.Close()
	serveStarted = true
	go func() { serveDone <- app.Serve(listener) }()
	client := &http.Client{Timeout: 3*time.Second}
	for _, path := range []string{"/sql-users/postgres/101", "/sql-users/mysql/202"} {
		resp, err := client.Get("http://" + listener.Addr().String() + path)
		if err != nil { t.Fatal(err) }
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil { t.Fatal(err) }
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "DROP TABLE") {
			t.Fatalf("%s: status=%d body=%s", path, resp.StatusCode, body)
		}
	}

	// Generated repositories accept a sql.Tx executor and do not bypass it.
	for _, tc := range []struct {
		name string
		db *sql.DB
		insert func(*sql.Tx) (int64, error)
		find func(int64) error
	}{
		{"postgres", pgDB, func(tx *sql.Tx) (int64, error) { return NewPostgresUsersRepositoryGenerated(tx).Insert(ctx, 303, "rolled back") }, func(id int64) error { _, err := pg.FindByID(ctx, id); return err }},
		{"mysql", myDB, func(tx *sql.Tx) (int64, error) { return NewMySQLUsersRepositoryGenerated(tx).Insert(ctx, 303, "rolled back") }, func(id int64) error { _, err := my.FindByID(ctx, id); return err }},
	} {
		t.Run(tc.name+"-rollback", func(t *testing.T) {
			tx, err := tc.db.BeginTx(ctx, nil)
			if err != nil { t.Fatal(err) }
			affected, err := tc.insert(tx)
			if err != nil || affected != 1 { t.Fatalf("INSERT in transaction affected=%d err=%v", affected, err) }
			if err := tx.Rollback(); err != nil { t.Fatal(err) }
			if err := tc.find(303); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("rolled-back row visible: %v", err)
			}
		})
	}

	if affected, err := pg.UpdateName(ctx, "postgres-updated", 101); err != nil || affected != 1 { t.Fatalf("postgres UPDATE affected=%d err=%v", affected, err) }
	if affected, err := my.UpdateName(ctx, "mysql-updated", 202); err != nil || affected != 1 { t.Fatalf("mysql UPDATE affected=%d err=%v", affected, err) }
	if user, err := pg.FindByID(ctx, 101); err != nil || user.Name != "postgres-updated" { t.Fatalf("postgres UPDATE readback: %+v %v", user, err) }
	if user, err := my.FindByID(ctx, 202); err != nil || user.Name != "mysql-updated" { t.Fatalf("mysql UPDATE readback: %+v %v", user, err) }
	if affected, err := pg.Delete(ctx, 101); err != nil || affected != 1 { t.Fatalf("postgres DELETE affected=%d err=%v", affected, err) }
	if affected, err := my.Delete(ctx, 202); err != nil || affected != 1 { t.Fatalf("mysql DELETE affected=%d err=%v", affected, err) }
	if _, err := pg.FindByID(ctx, 101); !errors.Is(err, sql.ErrNoRows) { t.Fatalf("postgres missing row: %v", err) }
	if _, err := my.FindByID(ctx, 202); !errors.Is(err, sql.ErrNoRows) { t.Fatalf("mysql missing row: %v", err) }
}
