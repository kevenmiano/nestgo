# NestGo - Exemplo Simples

Este é um exemplo simplificado do framework NestGo que demonstra os conceitos principais.

## Como Funciona

### 1. **Model** - Estrutura de dados
```go
type Product struct {
    ID    int     `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}
```

### 2. **Service** - Lógica de negócio
```go
type ProductService struct {
    service.BaseService
    products map[int]*Product
    nextID   int
}
```

### 3. **Controller** - Rotas HTTP
```go
type ProductController struct {
    controller.BaseController `baseUrl:"/products"`

    // Injeção de dependência
    ProductService *ProductService `inject:"ProductService"`

    // Rotas definidas com tags
    GetProducts   func() `route:"GET /"`
    GetProduct    func() `route:"GET /:id"`
    CreateProduct func() `route:"POST /"`
}
```

### 4. **Module** - Configuração do módulo
```go
var _ = module.New(module.ModuleConfig{
    Controllers: []interface{}{NewProductController()},
    Providers: []interface{}{
        NewProductService(),
    },
})(&ProductModule{})
```

## Conceitos Principais

### 🔧 **Injeção de Dependência**
- Use a tag `inject:"NomeDoService"` para injetar dependências
- O framework resolve automaticamente as dependências

### 🛣️ **Rotas**
- Use a tag `route:"METHOD /path"` para definir rotas
- Suporte a parâmetros: `/:id` vira `/{id}` no Gorilla Mux
- Extraia parâmetros com `mux.Vars(c.Request)`

### 📦 **Módulos**
- Agrupe controllers e services em módulos
- Use o decorator `module.New()` para configurar

### 🎯 **BaseController**
- Fornece métodos como `c.JSON()` e `c.Request`
- Acesso fácil ao request e response

## Como Executar

1. **Execute o exemplo simples:**
```bash
go run example_simple.go
```

2. **Teste as rotas:**
```bash
# Listar produtos
curl http://localhost:3001/products/

# Buscar produto por ID
curl http://localhost:3001/products/1

# Criar produto
curl -X POST http://localhost:3001/products/ \
  -H "Content-Type: application/json" \
  -d '{"name":"Teclado","price":89.90}'
```

## Estrutura do Framework

```
NestGo Framework
├── 🏗️  Module System (Agrupamento)
├── 🔧  Dependency Injection (Injeção automática)
├── 🛣️  Route Discovery (Descoberta de rotas)
├── 🎯  BaseController (Funcionalidades base)
└── 📝  Decorators (Tags para configuração)
```

## Vantagens

- ✅ **Simples**: Pouco código para muita funcionalidade
- ✅ **Automático**: Descoberta automática de rotas e dependências
- ✅ **Familiar**: Sintaxe similar ao NestJS
- ✅ **Flexível**: Fácil de estender e customizar
