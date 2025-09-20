# NestGo 🚀

<div align="center">
  <img src="docs/NESGO.png" alt="NestGo Logo" width="200"/>

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

### Exemplo Completo

Para entender como o framework funciona, execute o exemplo completo:

```bash
# Execute o exemplo completo
go run examples/main.go

# Teste as rotas
curl http://localhost:3000/users/
curl http://localhost:3000/users/1
```

### Exemplo Real - API REST Completa

O exemplo em `examples/main.go` demonstra uma API REST completa com:

#### 🎮 **Controller com Rotas HTTP**
```go
type UserController struct {
    controller.BaseController `baseUrl:"/users"`
    UserService *UserService `inject:"UserService"`

    // Rotas definidas com tags
    GetUsers     func() `route:"GET /"`
    CreateUser   func() `route:"POST /"`
    GetUser      func() `route:"GET /:id"`
    UpdateUser   func() `route:"PUT /:id"`
    DeleteUser   func() `route:"DELETE /:id"`
    PatchUser    func() `route:"PATCH /:id"`
    HeadUsers    func() `route:"HEAD /"`
    OptionsUsers func() `route:"OPTIONS /"`
}
```

#### 🔧 **Injeção de Dependência**
```go
type UserService struct {
    service.BaseService
    Database *FakeDatabase `inject:"FakeDatabase"`
}
```

#### 📦 **Módulo Configurado**
```go
var _ = module.New(module.ModuleConfig{
    Controllers: []interface{}{NewUserController()},
    Providers: []interface{}{
        NewFakeDatabase(),
        NewUserService(),
    },
})(&UserModule{})
```

## 🔗 Binding de Métodos Privados x Rotas

### Como Funciona o Sistema de Rotas

O framework NestGo usa um sistema inteligente que conecta **métodos privados** (handlers) com **rotas públicas** (tags):

#### 1️⃣ **Definição das Rotas**
```go
type UserController struct {
    // Campos de função com tags de rota
    GetUsers     func() `route:"GET /"`
    CreateUser   func() `route:"POST /"`
    GetUser      func() `route:"GET /:id"`
    UpdateUser   func() `route:"PUT /:id"`
    DeleteUser   func() `route:"DELETE /:id"`
    PatchUser    func() `route:"PATCH /:id"`
    HeadUsers    func() `route:"HEAD /"`
    OptionsUsers func() `route:"OPTIONS /"`
}
```

#### 2️⃣ **Implementação dos Handlers (Privados)**
```go
// Métodos privados que contêm a lógica real
func (c *UserController) getUsersHandler() {
    users := c.UserService.GetAllUsers()
    c.JSON(map[string]interface{}{
        "data":  users,
        "count": len(users),
    })
}

func (c *UserController) createUserHandler() {
    // Lógica para criar usuário
}

func (c *UserController) getUserHandler() {
    // Lógica para buscar usuário por ID
}
```

#### 3️⃣ **Binding Manual no Factory**
```go
func NewUserController() *UserController {
    controller := &UserController{}

    // Conecta rotas públicas com handlers privados
    controller.GetUsers = func() { controller.getUsersHandler() }
    controller.CreateUser = func() { controller.createUserHandler() }
    controller.GetUser = func() { controller.getUserHandler() }
    controller.UpdateUser = func() { controller.updateUserHandler() }
    controller.DeleteUser = func() { controller.deleteUserHandler() }
    controller.PatchUser = func() { controller.patchUserHandler() }
    controller.HeadUsers = func() { controller.headUsersHandler() }
    controller.OptionsUsers = func() { controller.optionsUsersHandler() }

    return controller
}
```

### 🎯 **Vantagens desta Abordagem**

- ✅ **Separação Clara**: Rotas públicas vs lógica privada
- ✅ **Flexibilidade**: Pode mudar implementação sem afetar rotas
- ✅ **Testabilidade**: Handlers privados são fáceis de testar
- ✅ **Convenção**: Nome da rota + "Handler" = método privado
- ✅ **Type Safety**: Go garante que as funções existem

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

## 📚 Funcionalidades Demonstradas

### ✅ **O que o Exemplo Mostra**

- **CRUD Completo**: Create, Read, Update, Delete
- **Parâmetros de Rota**: Extração de `:id` da URL
- **Parsing de JSON**: Request/Response automático
- **Injeção de Dependência**: Service → Database
- **Thread Safety**: Mutex para operações concorrentes
- **Logging Estruturado**: Logs detalhados de todas as operações
- **Tratamento de Erros**: Respostas de erro padronizadas
- **Métodos HTTP**: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS

## ✨ Vantagens do NestGo

- ✅ **Simples**: Pouco código para muita funcionalidade
- ✅ **Automático**: Descoberta automática de rotas e dependências
- ✅ **Familiar**: Sintaxe similar ao NestJS
- ✅ **Flexível**: Fácil de estender e customizar
- ✅ **Performance**: Construído em Go para máxima velocidade
- ✅ **Modular**: Organização clara em módulos
- ✅ **Type-Safe**: Tipagem forte do Go

## 🧪 Testando a API

Use o arquivo de exemplo incluído para testar todas as rotas:

```bash
# Execute o exemplo
go run examples/main.go

# Teste as rotas principais
curl http://localhost:3000/users/
curl http://localhost:3000/users/1
curl -X POST http://localhost:3000/users/ \
  -H "Content-Type: application/json" \
  -d '{"name":"João Silva","email":"joao@example.com","age":30}'
```

### Arquivo de Teste Completo
- `examples/examples.http` - Testes completos de todas as rotas (17 testes diferentes)

### Rotas Disponíveis
- `GET /users/` - Listar todos os usuários
- `POST /users/` - Criar novo usuário
- `GET /users/:id` - Buscar usuário por ID
- `PUT /users/:id` - Atualizar usuário completo
- `PATCH /users/:id` - Atualização parcial
- `DELETE /users/:id` - Deletar usuário
- `HEAD /users/` - Headers de resposta
- `OPTIONS /users/` - Métodos permitidos

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
├── examples/                 # Exemplos de uso
│   ├── main.go              # Exemplo completo com API REST
│   └── examples.http        # Testes completos (17 testes)
├── docs/
│   └── NESGO.png            # Logo do framework
├── README.md                # Documentação principal
├── pkg/                     # Código do framework
│   ├── application/         # Aplicação principal
│   ├── controller/          # BaseController
│   ├── service/             # BaseService
│   ├── module/              # Sistema de módulos
│   ├── decorators/          # Decorators
│   ├── server/              # Servidor HTTP
│   └── logger/              # Sistema de logs
└── go.mod                   # Dependências Go
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
