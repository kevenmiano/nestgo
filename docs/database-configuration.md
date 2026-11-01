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

## Generated repositories and runtime injection (experimental)

For `@Repository(User, "users")` on a `.mgo` type with supported `@Query` methods, `CompileDir` emits a typed constructor and `zz_modforge_repositories.go`. After opening the named pools, construct the providers and pass their instances to runtime:

```go
repositories, err := NewGeneratedRepositories(connections)
if err != nil { return err }
handler, err := runtime.NewWithOptions(
    runtime.Options{Instances: append([]any{settings, connections}, repositories.Instances()...)},
    generatedMetadata,
)
```

The compiler still does **not** automatically wire these instances from `@Module({imports:[...]})`. The generated registry is a bootstrap bridge rather than completed declarative DI. Static SQL support remains limited to SELECT, named binding and fields matching the entity's scan order. These limitations are tracked in #87, #88 and #89.
