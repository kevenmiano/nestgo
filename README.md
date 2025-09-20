# NestGo 🚀

<div align="center">
  <img src="NESGO.png" alt="NestGo Logo" width="200"/>

  **Um framework Go inspirado no NestJS para desenvolvimento de APIs escaláveis e modulares**

  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)](LICENSE)
  [![Status](https://img.shields.io/badge/Status-POC-orange?style=for-the-badge)](https://github.com/kevenmiano/nestgo)
  [![Production Ready](https://img.shields.io/badge/Production%20Ready-No-red?style=for-the-badge)](https://github.com/kevenmiano/nestgo)
</div>

## 📖 Sobre o NestGo

NestGo é um framework Go moderno e poderoso inspirado no NestJS, projetado para criar aplicações server-side escaláveis e eficientes. Ele combina elementos do paradigma de programação orientada a objetos, programação funcional e programação reativa.

> ⚠️ **IMPORTANTE**: Este é um **Proof of Concept (POC)** e **NÃO está pronto para produção**. Use apenas para fins de aprendizado e experimentação.

### ✨ Características Principais

- 🏗️ **Arquitetura Modular**: Organize seu código em módulos reutilizáveis
- 🎯 **Dependency Injection**: Sistema de injeção de dependências nativo
- 🛣️ **Auto-descoberta de Rotas**: Descoberta automática de rotas baseada em convenções
- 🌳 **Árvore de Dependências**: Visualização hierárquica da estrutura da aplicação
- 🚀 **Performance**: Construído em Go para máxima performance
- 🔧 **Decorators**: Sistema de decorators para metadados
- 📦 **Modular**: Estrutura baseada em módulos como o NestJS

## ⚠️ Status do Projeto

**Este é um Proof of Concept (POC) em desenvolvimento ativo.**

### 🚧 Limitações Atuais
- ❌ Não testado em produção
- ❌ Falta de testes automatizados
- ❌ Documentação em desenvolvimento
- ❌ Middleware limitado
- ❌ Sem suporte a WebSockets
- ❌ Sem sistema de autenticação integrado

### 🎯 Objetivos do POC
- ✅ Demonstrar conceitos de DI em Go
- ✅ Implementar descoberta automática de rotas
- ✅ Criar sistema modular similar ao NestJS
- ✅ Validar viabilidade da arquitetura

### 🔮 Roadmap Futuro
- [ ] Testes automatizados
- [ ] Middleware avançado
- [ ] Sistema de autenticação
- [ ] WebSockets
- [ ] CLI para geração de código
- [ ] Documentação completa

## 🚀 Início Rápido

### Instalação

```bash
go mod init meu-projeto
go get github.com/kevenmiano/nestgo
```

### Exemplo Simples

Para entender rapidamente como o framework funciona, execute o exemplo simples:

```bash
# Execute o exemplo básico
go run examples/main.go

# Teste as rotas
curl http://localhost:3001/products/
curl http://localhost:3001/products/1
```

### Exemplo Básico

```go
package main

import (
    "github.com/kevenmiano/nestgo/pkg/application"
    "github.com/kevenmiano/nestgo/pkg/controller"
    "github.com/kevenmiano/nestgo/pkg/service"
    "github.com/kevenmiano/nestgo/pkg/module"
)

// UserService - Service para lógica de negócio
type UserService struct {
    service.BaseService
}

func (s *UserService) GetAllUsers() []string {
    return []string{"user1", "user2", "user3"}
}

func (s *UserService) CreateUser(name string) string {
    return "Created user: " + name
}

// UserController - Controller para rotas HTTP
type UserController struct {
    controller.BaseController `baseUrl:"/users"`

    // Injeção de dependência
    UserService *UserService `inject:"UserService"`
}

// @route GET /
func (c *UserController) GetUsers() {
    users := c.UserService.GetAllUsers()
    // Lógica do controller
}

// @route POST /
func (c *UserController) CreateUser() {
    result := c.UserService.CreateUser("newuser")
    // Lógica do controller
}

// UserModule - Módulo principal
type UserModule struct{}

// Configuração do módulo
var _ = module.ModuleDecorator(module.ModuleConfig{
    Controllers: []interface{}{&UserController{}},
    Providers: []interface{}{
        &UserService{},
    },
})(&UserModule{})

func main() {
    // Inicia a aplicação
    application.StartApplication(":3000")
}
```

## 🎯 Conceitos Principais

### 🔧 **Injeção de Dependência**
```go
type ProductController struct {
    controller.BaseController `baseUrl:"/products"`

    // Injeção automática por tag
    ProductService *ProductService `inject:"ProductService"`
}
```

### 🛣️ **Rotas com Tags**
```go
type ProductController struct {
    // Rotas definidas com tags
    GetProducts   func() `route:"GET /"`
    GetProduct    func() `route:"GET /:id"`
    CreateProduct func() `route:"POST /"`
}
```

### 📦 **Módulos**
```go
var _ = module.New(module.ModuleConfig{
    Controllers: []interface{}{NewProductController()},
    Providers: []interface{}{NewProductService()},
})(&ProductModule{})
```

### 🎯 **BaseController**
```go
func (c *ProductController) getProductHandler() {
    // Acesso fácil ao request e response
    vars := mux.Vars(c.Request)
    c.JSON(map[string]interface{}{
        "data": product,
    })
}
```

## 🏗️ Arquitetura

### Estrutura de Módulos

```
Application
├── UserModule
│   ├── UserController
│   │   ├── GetUsers (GET /users/)
│   │   └── CreateUser (POST /users/)
│   └── UserService
└── AuthModule
    ├── AuthController
    └── AuthService
```

### Componentes Principais

#### 🎮 Controllers
Responsáveis por lidar com requisições HTTP e retornar respostas.

```go
type UserController struct {
    controller.BaseController `baseUrl:"/users"`
    UserService *UserService `inject:"UserService"`
}

// @route GET /
func (c *UserController) GetUsers() {
    // Lógica do endpoint
}
```

#### 🔧 Services
Contêm a lógica de negócio da aplicação.

```go
type UserService struct {
    service.BaseService
}

func (s *UserService) GetAllUsers() []string {
    // Lógica de negócio
    return []string{"user1", "user2"}
}
```

#### 📦 Modules
Organizam a aplicação em módulos funcionais.

```go
type UserModule struct{}

var _ = module.ModuleDecorator(module.ModuleConfig{
    Controllers: []interface{}{&UserController{}},
    Providers: []interface{}{&UserService{}},
})(&UserModule{})
```

## 🛣️ Sistema de Rotas

### Convenções de Nomenclatura

| Prefixo do Método | HTTP Method | Exemplo |
|------------------|-------------|---------|
| `Get` | GET | `GetUsers()` → `GET /users/` |
| `Create` | POST | `CreateUser()` → `POST /users/` |
| `Update` | PUT | `UpdateUser()` → `PUT /users/` |
| `Delete` | DELETE | `DeleteUser()` → `DELETE /users/` |

### Decorators de Rota

```go
// @route GET /
func (c *UserController) GetUsers() {
    // Implementação
}

// @route POST /
func (c *UserController) CreateUser() {
    // Implementação
}
```

## 🔄 Dependency Injection

O NestGo possui um sistema de injeção de dependências integrado:

```go
type UserController struct {
    controller.BaseController `baseUrl:"/users"`

    // Injeção automática por tag
    UserService *UserService `inject:"UserService"`
    AuthService *AuthService `inject:"AuthService"`
}
```

## 🌳 Árvore de Dependências

O framework gera automaticamente uma visualização hierárquica da estrutura da aplicação:

```
================================================================================
🌳 APPLICATION DEPENDENCY TREE
================================================================================
🏠 Application
  📦 UserModule
    🎮 UserController
      🛣️ GetUsers {"httpMethod": "GET", "path": "/users/"}
      🛣️ CreateUser {"httpMethod": "POST", "path": "/users/"}
    🔧 UserService
================================================================================
```

## 📚 Exemplos de Uso

### API REST Completa

```go
package main

import (
    "github.com/kevenmiano/nestgo/pkg/application"
    "github.com/kevenmiano/nestgo/pkg/controller"
    "github.com/kevenmiano/nestgo/pkg/service"
    "github.com/kevenmiano/nestgo/pkg/module"
)

// User model
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// UserService
type UserService struct {
    service.BaseService
    users []User
}

func (s *UserService) GetAllUsers() []User {
    return s.users
}

func (s *UserService) CreateUser(name, email string) User {
    user := User{
        ID:    len(s.users) + 1,
        Name:  name,
        Email: email,
    }
    s.users = append(s.users, user)
    return user
}

// UserController
type UserController struct {
    controller.BaseController `baseUrl:"/users"`
    UserService *UserService `inject:"UserService"`
}

// @route GET /
func (c *UserController) GetUsers() {
    users := c.UserService.GetAllUsers()
    // Retorna lista de usuários
}

// @route POST /
func (c *UserController) CreateUser() {
    user := c.UserService.CreateUser("New User", "user@example.com")
    // Retorna usuário criado
}

// UserModule
type UserModule struct{}

var _ = module.ModuleDecorator(module.ModuleConfig{
    Controllers: []interface{}{&UserController{}},
    Providers: []interface{}{&UserService{}},
})(&UserModule{})

func main() {
    application.StartApplication(":3000")
}
```

## ✨ Vantagens do NestGo

- ✅ **Simples**: Pouco código para muita funcionalidade
- ✅ **Automático**: Descoberta automática de rotas e dependências
- ✅ **Familiar**: Sintaxe similar ao NestJS
- ✅ **Flexível**: Fácil de estender e customizar
- ✅ **Performance**: Construído em Go para máxima velocidade
- ✅ **Modular**: Organização clara em módulos
- ✅ **Type-Safe**: Tipagem forte do Go

## 🧪 Testando a API

Use os arquivos de exemplo incluídos para testar suas rotas:

```bash
# Exemplo simples (porta 3001)
go run examples/main.go
curl http://localhost:3001/products/

# Exemplo completo (porta 3000)
go run main.go
curl http://localhost:3000/users/
```

### Arquivos de Teste
- `examples.http` - Testes do exemplo completo
- `examples_simple.http` - Testes do exemplo simples

## 🔧 Configuração Avançada

### Middleware Personalizado

```go
func CustomMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Lógica do middleware
        next.ServeHTTP(w, r)
    })
}
```

### Logging Estruturado

```go
import "github.com/kevenmiano/nestgo/pkg/logger"

logger.Info("User created", "userId", user.ID, "name", user.Name)
logger.Error("Database error", "error", err)
```

## 📦 Estrutura do Projeto

```
nestgo/
├── main.go                    # Exemplo completo
├── examples/
│   └── main.go               # Exemplo simples
├── NESGO.png                 # Logo do framework
├── examples.http             # Testes do exemplo completo
├── examples_simple.http      # Testes do exemplo simples
├── README.md                 # Documentação principal
├── README_SIMPLE.md          # Documentação do exemplo simples
├── pkg/                      # Código do framework
│   ├── application/          # Aplicação principal
│   ├── controller/           # BaseController
│   ├── service/              # BaseService
│   ├── module/               # Sistema de módulos
│   ├── decorators/           # Decorators
│   ├── server/               # Servidor HTTP
│   └── logger/               # Sistema de logs
└── go.mod                    # Dependências Go
```

## 🤝 Contribuindo

Como este é um **POC em desenvolvimento**, contribuições são especialmente bem-vindas!

### 🎯 Como Contribuir

1. **Fork o projeto**
2. **Crie uma branch** para sua feature (`git checkout -b feature/AmazingFeature`)
3. **Commit suas mudanças** (`git commit -m 'Add some AmazingFeature'`)
4. **Push para a branch** (`git push origin feature/AmazingFeature`)
5. **Abra um Pull Request**

### 💡 Ideias para Contribuição

- 🧪 **Testes**: Adicionar testes automatizados
- 📚 **Documentação**: Melhorar a documentação
- 🛠️ **Features**: Implementar novas funcionalidades
- 🐛 **Bugs**: Reportar e corrigir bugs
- 🎨 **Exemplos**: Criar mais exemplos de uso
- 🔧 **CLI**: Desenvolver ferramentas de linha de comando

### ⚠️ Importante
- Este é um POC - seja criativo e experimental!
- Não há garantias de estabilidade da API
- Foque em demonstrar conceitos e ideias

## 📄 Licença

Este projeto está licenciado sob a Licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🙏 Agradecimentos

- Inspirado no [NestJS](https://nestjs.com/) framework
- Construído com [Go](https://golang.org/)
- Comunidade Go brasileira

---

<div align="center">
  <strong>Feito com ❤️ em Go</strong>
</div>
