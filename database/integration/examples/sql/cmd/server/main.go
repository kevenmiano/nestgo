// Command server runs the compiled NestGo SQL example against real databases.
package main

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kevenmiano/nestgo/database"
	_ "github.com/kevenmiano/nestgo/database/integrationtests/examples/sql"
	"github.com/kevenmiano/nestgo/runtime"
	_ "github.com/lib/pq"
)

func main() {
	postgresDSN := os.Getenv("NESTGO_TEST_POSTGRES_DSN")
	mysqlDSN := os.Getenv("NESTGO_TEST_MYSQL_DSN")
	if postgresDSN == "" || mysqlDSN == "" {
		log.Fatal("configure NESTGO_TEST_POSTGRES_DSN and NESTGO_TEST_MYSQL_DSN")
	}
	app, err := runtime.New(
		runtime.WithDatabase(
			database.WithConnection("users", database.WithDriver("postgres"), database.WithDSN(postgresDSN)),
			database.WithConnection("billing", database.WithDriver("mysql"), database.WithDSN(mysqlDSN)),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()
	log.Print("SQL example listening on :8080; GET /sql-users/postgres/{id} or /sql-users/mysql/{id}")
	log.Fatal(app.Listen(":8080"))
}
