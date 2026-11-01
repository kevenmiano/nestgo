# Logging

NestGo provides the standalone `logging` package and **opt-in** bootstrap logging through the single application factory.

```go
logger := logging.New(logging.Config{
    Logger: logging.NewPretty(os.Stdout, true),
    MinLevel: logging.LevelDebug,
})
app, err := runtime.New(
    runtime.WithRegistry("shop"),
    runtime.WithLogger(logger),
    runtime.WithDebug(true),
)
if err != nil { log.Fatal(err) }
defer app.Shutdown(context.Background())
```

The application reports bootstrap start/failure, controller and route registration, provider creation at DEBUG level, and successful initialization. The `WithDebug` flag only emits detailed events when a logger is supplied. Bootstrap diagnostics do not mean the listener is already bound.

Structured production logging is supported with `logging.Slog{Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil))}`. Custom implementations of `logging.Logger` can be passed via `logging.Config`; global Go logger configuration is not changed.

Avoid logging credentials and raw request bodies. Pretty/debug output may add startup allocations; for measurements of actual bootstrap durations prefer `runtime.WithBootstrapMetrics`.

Related: [#50](https://github.com/kevenmiano/nestgo/issues/50), [runtime](RUNTIME.md).
