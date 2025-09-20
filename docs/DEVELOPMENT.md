# Guia de Desenvolvimento - NestGo

## Configuração do Ambiente

### Pré-requisitos
- Go 1.21 ou superior
- Git
- Editor de código (VS Code, GoLand, Vim, etc.)

### Instalação
```bash
# Clone o repositório
git clone https://github.com/kevenmiano/nestgo.git
cd nestgo

# Instale as dependências
go mod tidy

# Execute os testes
go test ./...

# Execute a aplicação
go run main.go
```

## Estrutura de um Projeto NestGo

### Layout Recomendado
```
meu-projeto/
├── main.go                    # Ponto de entrada
├── go.mod                     # Dependências
├── go.sum                     # Checksums
├── examples.http              # Testes de API
├── README.md                  # Documentação
├── pkg/                       # Código da aplicação
│   ├── controllers/           # Controllers
│   │   ├── user_controller.go
│   │   └── auth_controller.go
│   ├── services/              # Services
│   │   ├── user_service.go
│   │   └── auth_service.go
│   ├── models/                # Modelos de dados
│   │   ├── user.go
│   │   └── auth.go
│   ├── modules/               # Módulos
│   │   ├── user_module.go
│   │   └── auth_module.go
│   └── middleware/            # Middleware customizado
│       └── auth_middleware.go
└── docs/                      # Documentação
    ├── API.md
    └── ARCHITECTURE.md
```

## Criando um Controller

### Estrutura Básica
```go
package controllers

import (
    "github.com/kevenmiano/nestgo/pkg/controller"
)

type UserController struct {
    controller.BaseController `baseUrl:"/users"`

    // Injeção de dependências
    UserService *services.UserService `inject:"UserService"`
}

// @route GET /
func (c *UserController) GetUsers() {
    users := c.UserService.GetAllUsers()
    // Lógica do endpoint
}

// @route POST /
func (c *UserController) CreateUser() {
    user := c.UserService.CreateUser("New User")
    // Lógica do endpoint
}
```

### Convenções de Nomenclatura
- **Controllers**: `*Controller` (ex: `UserController`)
- **Métodos**: Prefixos baseados no HTTP method
  - `Get*` para GET
  - `Create*` para POST
  - `Update*` para PUT
  - `Delete*` para DELETE

### Decorators de Rota
```go
// @route GET /
func (c *UserController) GetUsers() { }

// @route POST /
func (c *UserController) CreateUser() { }

// @route PUT /:id
func (c *UserController) UpdateUser() { }

// @route DELETE /:id
func (c *UserController) DeleteUser() { }
```

## Criando um Service

### Estrutura Básica
```go
package services

import (
    "github.com/kevenmiano/nestgo/pkg/service"
)

type UserService struct {
    service.BaseService
    // Dependências
    Database *database.Database `inject:"Database"`
}

func (s *UserService) GetAllUsers() []models.User {
    // Lógica de negócio
    return s.Database.FindAllUsers()
}

func (s *UserService) CreateUser(name string) models.User {
    user := models.User{Name: name}
    return s.Database.SaveUser(user)
}
```

### Princípios de Design
- **Single Responsibility**: Um service por domínio
- **Dependency Injection**: Use tags `inject` para dependências
- **Interface Segregation**: Interfaces pequenas e específicas

## Criando um Módulo

### Estrutura Básica
```go
package modules

import (
    "github.com/kevenmiano/nestgo/pkg/module"
    "meu-projeto/pkg/controllers"
    "meu-projeto/pkg/services"
)

type UserModule struct{}

// Configuração do módulo
var _ = module.ModuleDecorator(module.ModuleConfig{
    Controllers: []interface{}{
        &controllers.UserController{},
    },
    Providers: []interface{}{
        &services.UserService{},
    },
    Imports: []interface{}{
        // Outros módulos se necessário
    },
})(&UserModule{})
```

### Organização de Módulos
- **Feature-based**: Um módulo por feature
- **Domain-driven**: Baseado em domínios de negócio
- **Loose coupling**: Baixo acoplamento entre módulos

## Dependency Injection

### Tags de Injeção
```go
type UserController struct {
    controller.BaseController `baseUrl:"/users"`

    // Injeção por nome
    UserService *UserService `inject:"UserService"`

    // Injeção por tipo (automática)
    Database *Database `inject:""`
}
```

### Registro de Dependências
```go
// No módulo
var _ = module.ModuleDecorator(module.ModuleConfig{
    Providers: []interface{}{
        &services.UserService{},
        &database.Database{},
    },
})(&UserModule{})
```

## Middleware Customizado

### Criando Middleware
```go
package middleware

import (
    "net/http"
    "github.com/kevenmiano/nestgo/pkg/logger"
)

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")

        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Validação do token
        if !validateToken(token) {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        logger.Info("User authenticated", "token", token)
        next.ServeHTTP(w, r)
    })
}

func validateToken(token string) bool {
    // Lógica de validação
    return true
}
```

### Aplicando Middleware
```go
// No controller
type UserController struct {
    controller.BaseController `baseUrl:"/users" middleware:"AuthMiddleware"`
    UserService *UserService `inject:"UserService"`
}
```

## Logging

### Uso Básico
```go
import "github.com/kevenmiano/nestgo/pkg/logger"

// Logs de diferentes níveis
logger.Debug("Debug information", "key", "value")
logger.Info("User created", "userId", user.ID)
logger.Warn("Deprecated method used", "method", "oldMethod")
logger.Error("Database error", "error", err)
```

### Logging Estruturado
```go
// Com contexto
logger.Info("Request processed",
    "method", r.Method,
    "path", r.URL.Path,
    "status", 200,
    "duration", time.Since(start))
```

## Testes

### Testes de Controller
```go
package controllers

import (
    "testing"
    "net/http/httptest"
    "github.com/stretchr/testify/assert"
)

func TestUserController_GetUsers(t *testing.T) {
    // Setup
    controller := &UserController{
        UserService: &MockUserService{},
    }

    // Test
    req := httptest.NewRequest("GET", "/users/", nil)
    w := httptest.NewRecorder()

    controller.GetUsers(w, req)

    // Assertions
    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "user1")
}
```

### Testes de Service
```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestUserService_CreateUser(t *testing.T) {
    // Setup
    service := &UserService{
        Database: &MockDatabase{},
    }

    // Test
    user := service.CreateUser("Test User")

    // Assertions
    assert.Equal(t, "Test User", user.Name)
    assert.NotEmpty(t, user.ID)
}
```

## Debugging

### Logs de Debug
```go
// Ative logs de debug
logger.Debug("Processing request", "requestId", requestID)
```

### Visualização da Árvore
```go
// A árvore de dependências é exibida automaticamente na inicialização
// Para acessar programaticamente:
app := application.NewApplication()
tree := app.GetTree()
```

## Performance

### Otimizações Recomendadas
1. **Use interfaces**: Para melhor testabilidade
2. **Cache reflexão**: Metadados são cacheados automaticamente
3. **Connection pooling**: Para recursos externos
4. **Lazy loading**: Módulos carregados sob demanda

### Profiling
```bash
# CPU profiling
go run main.go -cpuprofile=cpu.prof

# Memory profiling
go run main.go -memprofile=mem.prof

# Analisar profiles
go tool pprof cpu.prof
```

## Deploy

### Build para Produção
```bash
# Build otimizado
go build -ldflags="-s -w" -o app main.go

# Executar
./app
```

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o app main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/app .
CMD ["./app"]
```

## Boas Práticas

### Código
- **Nomes descritivos**: Use nomes que expliquem a intenção
- **Funções pequenas**: Máximo 20-30 linhas
- **Error handling**: Sempre trate erros adequadamente
- **Documentação**: Comente código complexo

### Arquitetura
- **Separation of concerns**: Separe responsabilidades
- **Dependency inversion**: Dependa de abstrações
- **Single responsibility**: Uma responsabilidade por classe
- **Open/closed principle**: Aberto para extensão, fechado para modificação

### Performance
- **Avoid allocations**: Reutilize objetos quando possível
- **Use sync.Pool**: Para objetos frequentemente alocados
- **Profile regularly**: Monitore performance regularmente
- **Cache wisely**: Cache dados que são caros de calcular
