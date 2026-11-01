# SQL persistence adapters (foundation)

NestGo keeps persistence opt-in. The database package is a lightweight execution boundary, not a Hibernate ORM. It supports the standard `database/sql` API and adapters satisfying the `sqlx` method set without requiring application code to depend on a particular implementation.

## Native SQL

```go
db, err := sql.Open("postgres", dsn) // register your chosen driver
if err != nil { return err }
defer db.Close()
store, err := database.Native(db)
if err != nil { return err }
row := store.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", id)
err = row.Scan(&name)
```

## sqlx

```go
db, err := sqlx.Open("postgres", dsn)
if err != nil { return err }
defer db.Close()
adapter, err := nestgosqlx.New(db)
if err != nil { return err }
err = adapter.Get(ctx, &user, "SELECT id, name FROM users WHERE id = $1", id)
```

Alias `github.com/kevenmiano/nestgo/database/sqlx` as `nestgosqlx` and import `github.com/jmoiron/sqlx` separately. Only the application needs the sqlx dependency.

## Transactions and ownership

Call `database.Begin(ctx, db, options)`, `tx.Store()`, and `tx.Commit()` or `tx.Rollback()`. Keep transaction lifetime in the service/use-case layer. NestGo never closes a caller-owned connection; the application owns pool configuration and shutdown.

## Deliberately deferred compiler features

`@Entity`, `@Column`, `@Repository`, and `@Query` syntax **is not implemented by this foundation**. Compiler validation, SQL generation, DI provider factories, generated wiring, and dialect-specific placeholder handling require separate work. Do not advertise decorator support before code-generation and compiler tests exist. No runtime SQL reflection or implicit schema migration is planned.
