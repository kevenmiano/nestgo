# Logging — GSD iteration 1

Related: [#50](https://github.com/kevenmiano/nestgo/issues/50), [#16](https://github.com/kevenmiano/nestgo/issues/16).

The standalone `logging` package is opt-in. It does **not yet** automatically integrate with `runtime.New` or discover dependency/module/route lifecycle events; those require an instrumented bootstrap and real measurements. Do not invent startup events.

## Default: standard Go log

```go
logger := logging.New(logging.Config{})
logger.Info(context.Background(), "App", "ready")
```

This uses `log.Default()` with its original output, flags and timestamp behavior. It does not reconfigure the global Go logger.

## Pretty console (explicit opt-in)

```go
logger := logging.New(logging.Config{
    Logger: logging.NewPretty(os.Stdout, true), // ANSI enabled explicitly
    MinLevel: logging.LevelInfo,
})
logger.Info(context.Background(), "NestFactory", "Starting NestGo application...")
```

Use `false` for non-ANSI output. Pretty serializes output across goroutines.

## Structured production logging

```go
logger := logging.New(logging.Config{
    Logger: logging.Slog{Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil))},
    MinLevel: logging.LevelInfo,
})
logger.Info(ctx, "OrdersService", "created", slog.String("request_id", id))
```

`slog` is the standard library structured logging API. Custom adapters implement `logging.Logger`; keep backend dependencies outside NestGo core.

## Bootstrap contract

Call `logger.Startup(ctx, component, message, startedAt)` only on actual successful lifecycle milestones. It emits a duration attribute based on elapsed time. Runtime module graph and route-mapping integration remain follow-up work.

## Security and performance

Never log secrets or raw request bodies by default. Use level filtering in high-throughput services. Don't enable pretty formatting/ANSI in ingestion pipelines. Profile allocations and throughput before changing default runtime behavior.

## GSD gates

- Discuss/Plan: issue #50.
- Execute: logging API, adapters, examples and tests in this branch.
- Verify: local package tests completed; repository CI/race tests and real runtime instrumentation pending.
- Review/Ship: do not merge or close #50 until CI and review pass.

## Automatic runtime diagnostics

`runtime.New(...packages)` now emits lifecycle logs through the standard Go logger by default. For custom logging, use `runtime.NewWithOptions`:

```go
logger := logging.New(logging.Config{
    Logger: logging.NewPretty(os.Stdout, true),
    MinLevel: logging.LevelDebug,
})
handler, err := runtime.NewWithOptions(runtime.Options{
    Logger: logger,
    Debug: true,
}, packages...)
```

Real events: NestFactory begins bootstrap, InstanceLoader tracks package/provider registration and dependency resolution, RoutesResolver registers controllers, RouterExplorer maps validated routes, NestApplication reports handler initialization. Provider instantiation detail is DEBUG. Timings are actual `time.Since` values, not samples or estimates.

**Limits:** `runtime.New` returns an `http.Handler`; handler initialization is not HTTP listener readiness. Port bind, transport adapter lifecycle, graceful shutdown, request timing, failed bootstrap details for every validation path and complete module import/export graph need follow-up integration. `Debug` gates expensive detailed event creation; `MinLevel` still controls output. There is no promise of ordering for individual provider-resolution events from maps.

