# Configuração e conexões de banco no NestGo

A API usa construtores Go e Functional Options. Não existem `forRoot`, `forRootAsync` ou `registerAsync`.

## Construção

`database.New(executor)` continua sendo a API compatível para encapsular um executor `database/sql` existente. O construtor de um **módulo de múltiplos bancos** é `database.NewModule(options...)` para evitar colisão de nomes na linguagem Go.

```go
module, err := database.NewModule(
    database.WithConnection("users",
        database.WithDriver("postgres"),
        database.WithDSN(usersDSN),
        database.WithMaxOpenConns(20),
        database.WithMaxIdleConns(5),
    ),
    database.WithConnection("billing",
        database.WithDriver("mysql"),
        database.WithDSN(billingDSN),
    ),
)
if err != nil {
    return err
}
connections, err := module.Open(ctx)
if err != nil {
    return err
}
defer connections.Close()
```

`NewModule` constrói o plano sem I/O. `Open(ctx)` inicializa pools, testa conectividade e desfaz a inicialização anterior caso uma conexão falhe. `Close()` encerra os pools de propriedade do registry.

## Configuração

```go
settings, err := config.New(config.WithEnvFile(".env"))
if err != nil {
    return err
}
var cfg struct {
    DSN string `env:"DSN,required"`
    Port int `env:"PORT,default=5432,min=1,max=65535"`
}
if err := settings.Decode("DB", &cfg); err != nil {
    return err
}
```

O Viper está disponível no módulo **opcional** `config/viper`. O carregamento via Viper é explicitamente selecionado; não é o padrão automático do pacote `config`.

## Contratos e compatibilidade

- `New` constrói o tipo quando o nome está livre; `NewModule` distingue um módulo do `Store` já existente.
- `WithX` configura valores sem abrir conexões.
- `Open(ctx)` executa I/O e `Close()` libera recursos.
- Conexões são nomeadas e possuem pools independentes.
- Erros de validação não devem incluir DSNs ou segredos.
- `config.Load`, `database.Bootstrap`, `database.BootstrapFromConfig` e `database.New(executor)` continuam disponíveis.
- `@Repository` com seleção de conexão, módulos dinâmicos declarativos e DI gerado ainda não estão implementados. Não confundir o registry Go com a integração completa ao compilador.

## Generated repository injection (experimental)

The standard `CompileDir` command now generates a typed repository constructor and registers a provider factory in the generated `metadata.Package`. The runtime executes the factory once during bootstrap and injects the resulting repository into compatible provider/controller fields.

```go
// Once the named database pools have been opened:
handler, err := runtime.NewWithOptions(
    runtime.Options{Instances: []any{settings, connections}},
    generatedMetadata,
)
```

No manual `NewGeneratedRepositories` call is needed when the module is compiled with the new factory metadata. `@Repository(User, "users")` selects the named pool at startup; query handlers reuse the selected executor without a registry lookup on every request.

For a single local `@Module`, compiled repository providers are assigned automatically. With multiple local modules, specify the generated provider type in the owning module's `providers` declaration, for example `@Module({providers:[UsersRepositoryGenerated]})`. A repository owned by zero or multiple modules is a compile error.

**Limits:** database pools still need explicit initialization and must be provided to `runtime.Options.Instances`; decorators do not currently instantiate `DatabaseModule.NewModule(...)` themselves. Only the supported repository `SELECT` subset is emitted. The `@Query` editor autocomplete and full schema/migrations remain separate outstanding issues (#89–#93).

## Unified runtime bootstrap

The runtime also offers `runtime.Bootstrap(ctx, runtime.BootstrapOptions{...}, packages...)` as an opt-in lifecycle owner. It loads validated configuration, creates named database pools, and registers `*config.Service` plus `*database.Connections` in DI before starting the HTTP handler. Repository factories defined by compiled metadata can resolve the named registry during startup.

```go
app, err := runtime.Bootstrap(ctx, runtime.BootstrapOptions{
    ConfigOptions: []config.Option{config.WithEnvFile(".env")},
    DatabaseOptions: []database.Option{
        database.WithConnection("users", database.WithDriver("postgres"), database.WithDSN(usersDSN)),
    },
}, generatedMetadata)
if err != nil { return err }
defer app.Close()
handler := app.Handler
_ = handler
```

On an initialization error, opened pools are closed. On success, the caller must invoke `Close()` during shutdown. This API does not start an HTTP listener or parse dynamic constructors inside `@Module({imports:[...]})`; the executable Go bootstrap remains explicit. Neither the repository metadata nor `@Query` offer SQL editor autocomplete yet.
