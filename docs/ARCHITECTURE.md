# Arquitetura do NestGo

## Visão Geral

O NestGo é um framework Go que implementa uma arquitetura modular inspirada no NestJS, proporcionando uma estrutura organizada e escalável para desenvolvimento de APIs.

## Componentes Principais

### 1. Application Layer
- **Localização**: `pkg/application/`
- **Responsabilidade**: Ponto de entrada da aplicação e gerenciamento do ciclo de vida
- **Componentes**:
  - `Application`: Classe principal que gerencia a aplicação
  - `Bootstrap`: Funções de inicialização
  - `TreeNode`: Estrutura de árvore para dependências

### 2. Module System
- **Localização**: `pkg/module/`
- **Responsabilidade**: Sistema de módulos e registro automático
- **Componentes**:
  - `Module`: Interface base para todos os módulos
  - `BaseModule`: Implementação base com funcionalidades comuns
  - `ModuleRegistry`: Registro global de módulos
  - `AutoRegister`: Sistema de auto-registro

### 3. Controller Layer
- **Localização**: `pkg/controller/`
- **Responsabilidade**: Gerenciamento de rotas HTTP e handlers
- **Componentes**:
  - `BaseController`: Controller base com funcionalidades comuns
  - `MetaExtractor`: Extração de metadados de controllers

### 4. Service Layer
- **Localização**: `pkg/service/`
- **Responsabilidade**: Lógica de negócio da aplicação
- **Componentes**:
  - `BaseService`: Service base com funcionalidades comuns

### 5. Server Layer
- **Localização**: `pkg/server/`
- **Responsabilidade**: Servidor HTTP e descoberta de rotas
- **Componentes**:
  - `Server`: Servidor HTTP principal
  - `RouteDiscovery`: Descoberta automática de rotas

### 6. Router Layer
- **Localização**: `pkg/router/`
- **Responsabilidade**: Roteamento HTTP e gerenciamento de rotas
- **Componentes**:
  - `Router`: Gerenciador de rotas
  - `Route`: Estrutura de rota individual

### 7. Dependency Injection
- **Localização**: `pkg/container/`
- **Responsabilidade**: Sistema de injeção de dependências
- **Componentes**:
  - `Container`: Container de dependências
  - `AutoRegister`: Registro automático de serviços

### 8. Logging
- **Localização**: `pkg/logger/`
- **Responsabilidade**: Sistema de logging estruturado
- **Componentes**:
  - `Logger`: Logger principal com suporte a JSON

## Fluxo de Execução

### 1. Inicialização
```
main() → application.StartApplication() → Bootstrap
```

### 2. Descoberta de Módulos
```
AutoRegister → ModuleRegistry → Module Discovery
```

### 3. Construção da Árvore de Dependências
```
buildDependencyTree() → addModuleNode() → addControllerNode() → addRouteNode()
```

### 4. Registro de Rotas
```
RouteDiscovery → Server.RegisterController() → Router.RegisterRoute()
```

### 5. Injeção de Dependências
```
Container.AutoRegister() → Inject() → Dependency Resolution
```

## Sistema de Árvore de Dependências

### Estrutura TreeNode
```go
type TreeNode struct {
    Name     string                 // Nome do nó
    Type     string                 // Tipo: root, module, controller, route
    Data     map[string]interface{} // Dados do nó
    Children []*TreeNode            // Nós filhos
    Module   module.Module          // Referência ao módulo (se aplicável)
}
```

### Hierarquia de Nós
```
Application (root)
├── UserModule (module)
│   ├── UserController (controller)
│   │   ├── GetUsers (route)
│   │   └── CreateUser (route)
│   └── UserService (service)
└── AuthModule (module)
    ├── AuthController (controller)
    └── AuthService (service)
```

## Sistema de Rotas

### Descoberta Automática
1. **Reflexão**: Análise de métodos de controllers
2. **Convenções**: Prefixos de método (Get, Create, Update, Delete)
3. **Decorators**: Comentários `@route` para configuração
4. **Mapeamento**: Geração automática de rotas HTTP

### Convenções de Nomenclatura
- `Get*` → GET
- `Create*` → POST
- `Update*` → PUT
- `Delete*` → DELETE

## Dependency Injection

### Sistema de Tags
```go
type UserController struct {
    UserService *UserService `inject:"UserService"`
}
```

### Resolução de Dependências
1. **Registro**: Serviços registrados no container
2. **Resolução**: Injeção automática baseada em tags
3. **Singleton**: Instâncias únicas por tipo

## Logging Estruturado

### Formato JSON
```json
{
  "time": "2025-09-19T20:12:33.005788158-03:00",
  "level": "INFO",
  "msg": "Registering route",
  "method": "GetUsers",
  "httpMethod": "GET",
  "path": "/users/"
}
```

### Níveis de Log
- `DEBUG`: Informações detalhadas para debugging
- `INFO`: Informações gerais da aplicação
- `WARN`: Avisos sobre situações anômalas
- `ERROR`: Erros que não interrompem a aplicação

## Performance e Escalabilidade

### Otimizações
- **Reflexão Cached**: Metadados de reflexão são cacheados
- **Lazy Loading**: Módulos carregados sob demanda
- **Connection Pooling**: Pool de conexões para recursos externos

### Escalabilidade
- **Modular**: Estrutura modular permite crescimento horizontal
- **Stateless**: Controllers e services são stateless
- **Concurrent**: Suporte nativo à concorrência do Go

## Segurança

### Middleware de Segurança
- **CORS**: Configuração de Cross-Origin Resource Sharing
- **Rate Limiting**: Limitação de taxa de requisições
- **Authentication**: Sistema de autenticação integrado

### Validação
- **Input Validation**: Validação automática de entrada
- **Type Safety**: Tipagem forte do Go
- **Error Handling**: Tratamento robusto de erros

## Extensibilidade

### Plugins
- **Middleware**: Sistema de middleware extensível
- **Decorators**: Decorators customizáveis
- **Hooks**: Hooks de ciclo de vida

### Integração
- **Database**: Integração com bancos de dados
- **Cache**: Sistema de cache integrado
- **Message Queue**: Suporte a filas de mensagens
