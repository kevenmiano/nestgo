# SQL persistence in NestGo

NestGo is SQL-first and persistence is opt-in. It uses `database/sql` and optional `sqlx`; there is no runtime entity reflection, active record or implicit migration.

## Canonical runtime API

```go
app, err := runtime.NewApp(ctx,
    runtime.WithConfig(config.WithEnvFile(".env")),
    runtime.WithDatabase(database.WithConnection("users",
        database.WithDriver("postgres"),
        database.WithDSN(usersDSN),
        database.WithMaxOpenConns(20),
    )),
    runtime.WithRegistry("users"),
)
if err != nil { return err }
defer app.Close()
server := &http.Server{Addr: ":8080", Handler: app.HTTPHandler()}
return server.ListenAndServe()
```

Import your SQL driver in the application. `NewApp` opens pools and injects `*config.Service` and `*database.Connections` into the DI container.

## Generated repositories (experimental)

Write a `.mgo` entity and repository:

```go
@Entity({table: "users"})
type User struct { ID int64; Name string }
@Repository(User, "users")
type UsersRepository struct{}
@Query("SELECT id,name FROM users WHERE id=:id")
func (r *UsersRepository) FindByID(ctx context.Context, id int64) (*User, error) { return nil, nil }
```

Run `go run ./cmd/modforge compile ./path/to/package`. The compiler emits `UsersRepositoryGenerated` and registers a DI factory. One local `@Module` owns the generated repository automatically. Multiple modules require exactly one explicit owner and appropriate imports/exports.

**Generated SQL support:** `SELECT` returns `*Entity` or `[]Entity`; `INSERT`, `UPDATE` and `DELETE` without `RETURNING` return `(int64, error)` with rows affected. PostgreSQL `RETURNING` can map back to entity results. Unsupported statement/return combinations fail at compile time. Returned columns map by name (case-insensitive field names and snake_case), not struct order. Unknown or duplicate columns fail explicitly; empty single-row results return `sql.ErrNoRows`. `@Entity` is only a local model; `@Column`, migrations, CTEs, arbitrary DTOs and autocomplete remain outstanding (#89, #91, #93).

**SQL dialects:** `CompileDir` defaults to PostgreSQL; `CompileDirWithDialect` selects one dialect for the compilation. For mixed databases, compile explicitly with `CompileDirWithDialects(dir, map[string]sqlgen.Dialect{"users": sqlgen.PostgreSQL, "billing": sqlgen.Question})`, or run `go run ./cmd/modforge compile ./path/to/package --connection=users:postgres --connection=billing:mysql`. The compiler resolves each `@Repository(Entity, "connection")` by name and refuses missing mappings. The connection/driver dialect is not yet cross-validated at runtime (#90).

## Direct native SQL

```go
db, err := sql.Open("postgres", dsn)
if err != nil { return err }
defer db.Close() // caller-owned
store, err := database.Native(db)
if err != nil { return err }
var name string
err = store.QueryRow(ctx, "SELECT name FROM users WHERE id=$1", id).Scan(&name)
```

## Optional sqlx

```go
db, err := sqlx.Open("postgres", dsn)
if err != nil { return err }
defer db.Close()
adapter, err := nestgosqlx.New(db)
if err != nil { return err }
err = adapter.Get(ctx, &user, "SELECT id,name FROM users WHERE id=$1", id)
```

Import `github.com/kevenmiano/nestgo/database/sqlx` as `nestgosqlx`; application code opts into `github.com/jmoiron/sqlx`.

## Transactions and ownership

`database.Begin(ctx, db, options)` and `database.WithinTransaction` expose explicit commit and rollback. `database.Native(db)` and `database.New(executor)` wrap caller-owned handles. **`Connections.Register(name, db)` takes ownership on success**; rejected handles remain caller-owned. `runtime.NewApp` owns and closes its opened pools via `App.Close()`, but does not shut down an externally owned HTTP server.

Quality gates: gofmt, go vet, race tests, generated-code compile, PostgreSQL/MySQL integration and benchmarks (#94).
