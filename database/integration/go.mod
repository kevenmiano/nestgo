module github.com/kevenmiano/nestgo/database/integrationtests

go 1.23.0

require (
 github.com/kevenmiano/nestgo v0.0.0
 github.com/go-sql-driver/mysql v1.9.3
 github.com/jmoiron/sqlx v1.4.0
 github.com/lib/pq v1.10.9
)

replace github.com/kevenmiano/nestgo => ../..
