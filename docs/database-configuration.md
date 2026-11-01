# Configuração e múltiplas conexões SQL no NestGo

NestGo usa construtores idiomáticos com **Functional Options** (`New` / `With...`), sem `forRoot`, `forRootAsync` ou variáveis globais implícitas.

## API pública canônica

```go
app, err := runtime.NewApp(ctx,
    runtime.WithConfig(config.WithEnvFile(".env")),
    runtime.WithDatabase(
        database.WithConnection("users",
            database.WithDriver("postgres"),
            database.WithDSN(usersDSN),
            database.WithMaxOpenConns(20),
        ),
        database.WithConnection("billing",
            database.WithDriver("mysql"),
            database.WithDSN(billingDSN),
        ),
    ),
    runtime.WithRegistry("shop"),
)
if err != nil { return err }
defer app.Close()
handler := app.HTTPHandler()
_ = handler
```

O `NewApp` inicializa as configurações, abre os pools e registra `*config.Service` e `*database.Connections` no DI. Caso ocorra falha no bootstrap, os pools previamente abertos são encerrados.

## Uso sem o runtime

```go
module, err := database.NewModule(
    database.WithConnection("users", database.WithDriver("postgres"), database.WithDSN(usersDSN)),
    database.WithConnection("billing", database.WithDriver("mysql"), database.WithDSN(billingDSN)),
)
if err != nil { return err }
connections, err := module.Open(ctx)
if err != nil { return err }
defer connections.Close()
usersDB, err := connections.Get("users")
if err != nil { return err }
_ = usersDB
```

`NewModule` só constrói o plano. `Open(ctx)` valida a conectividade e executa rollback de recursos em falhas; `Close()` é idempotente. `WithConnectionFactory(name, func(context.Context) (database.ConnectionOptions, error))` resolve configurações no momento de abertura, sem convenções artificiais com `Async`.

## Configuração tipada

```go
settings, err := config.New(config.WithEnvFile(".env"))
if err != nil { return err }
var cfg struct {
    DSN string `env:"DSN,required"`
    Port int `env:"PORT,default=5432,min=1,max=65535"`
}
if err := settings.Decode("DB", &cfg); err != nil { return err }
```

O módulo `config/viper` é opcional e não é importado automaticamente. DSNs e segredos não devem aparecer nas mensagens de erro.

## Repositórios gerados e DI

`@Repository(User, "users")` usa uma conexão nomeada. `CompileDir` gera `UsersRepositoryGenerated` e uma factory registrada no metadata; não é necessário chamar `NewGeneratedRepositories` manualmente. Um único `@Module` local assume o provider automaticamente. Para múltiplos módulos, declare exatamente um proprietário e configure `imports` / `exports`.

**Limitação:** o SQL gerado ainda usa um dialeto único por compilação (PostgreSQL por padrão), **não** um dialeto por conexão. `database/sql` aceita PostgreSQL e MySQL simultaneamente, mas a geração de repositórios mistos precisa da issue #90.

Somente `SELECT` com resultados `*Entity` / `[]Entity` é gerado atualmente. Escritas, DTOs arbitrários, relações, migrations e autocomplete permanecem nas issues #89, #91 e #93. `@Column` não é suportado.

## Propriedade e desligamento

- `database.Native(db)` e `database.New(executor)` não tomam posse do executor.
- `Connections.Register(name, db)` transfere a propriedade em caso de sucesso; `Connections.Close()` fecha os pools.
- `App.Close()` fecha pools abertos por `NewApp`; o servidor HTTP externo continua responsabilidade do chamador.
- Os adaptadores HTTP ainda serão unificados na issue #103.

Veja também [guia de persistência SQL](DATABASE.md). A PR #82 deve permanecer draft até passar CI e resolver conflitos.
