# SQL: compiled PostgreSQL and MySQL repositories

This is a **runnable** NestGo persistence example (a separate Go integration-test module). It demonstrates:

- `@Entity`, `@Repository(Entity, "connection")` and `@Query` in `.mgo`.
- Multiple named PostgreSQL/MySQL pools, each with the correct compiled bind dialect.
- A generated repository injected into a service, then a controller, with **explicit** `nestgo:"inject"` fields.
- Generated CRUD, typed row/collection decoding, reversed column order, malicious input as bound arguments, transaction rollback and real HTTP requests.
- No runtime entity reflection, global connection pool, implicit migrations or `WithRegistry`.

## Generate and test

From the **repository root**:

```sh
go run ./cmd/nestgo compile ./database/integration/examples/sql --connection=users:postgres --connection=billing:mysql
```

Start a real local PostgreSQL/MySQL or use the `postgres:16`/`mysql:8.4` service containers configured in `.github/workflows/database-integration.yml`. Define the two **test database** connection strings:

```sh
export NESTGO_TEST_POSTGRES_DSN='postgres://nestgo:integration_only@127.0.0.1:5432/nestgo?sslmode=disable'
export NESTGO_TEST_MYSQL_DSN='nestgo:integration_only@tcp(127.0.0.1:3306)/nestgo?parseTime=true'
cd database/integration
go test -race -count=1 -v ./...
```

The test creates `nestgo_sql_example_users` in **each** disposable database, executes CRUD with the generated repositories, and cleans its rows. **Do not point this integration test at a production database**: it clears the named example table.

The same test runs automatically in GitHub Actions against two real database services. Locally, missing DSNs cause tests to skip; in CI both are mandatory.

## Run the HTTP example

After generation, with the same DSNs set:

```sh
cd database/integration
go run ./examples/sql/cmd/server
```

Insert seed rows into the table in each database, then request:

```sh
curl http://localhost:8080/sql-users/postgres/101
curl http://localhost:8080/sql-users/mysql/202
```

The `module.mgo` service/controller illustrates the full path from route binding to compiled SQL. The application imports compiled module metadata and only configures `runtime.New(runtime.WithDatabase(...))`. Drivers are imported by the application.

## Scope and limitations

`@Column`, schema migrations, CTEs, arbitrary DTO projections, automatic runtime driver/dialect cross-validation and relation mapping are not part of this example. Use a separate migration tool before serving data. See `docs/DATABASE.md` and issues #89–#94.
