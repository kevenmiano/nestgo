package main

import (
	"encoding/json"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kevenmiano/nestgo/pkg/application"
	"github.com/kevenmiano/nestgo/pkg/controller"
	"github.com/kevenmiano/nestgo/pkg/logger"
	"github.com/kevenmiano/nestgo/pkg/module"
	"github.com/kevenmiano/nestgo/pkg/service"
)

// Product model
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// ProductService - Service simples
type ProductService struct {
	service.BaseService
	products map[int]*Product
	nextID   int
}

func NewProductService() *ProductService {
	service := &ProductService{
		products: make(map[int]*Product),
		nextID:   1,
	}

	// Dados de exemplo
	service.products[1] = &Product{ID: 1, Name: "Laptop", Price: 1500.00}
	service.products[2] = &Product{ID: 2, Name: "Mouse", Price: 25.50}

	logger.Info("ProductService criado com dados de exemplo")
	return service
}

func (s *ProductService) GetAllProducts() []*Product {
	products := make([]*Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}
	return products
}

func (s *ProductService) GetProductByID(id int) *Product {
	return s.products[id]
}

func (s *ProductService) CreateProduct(name string, price float64) *Product {
	product := &Product{
		ID:    s.nextID,
		Name:  name,
		Price: price,
	}
	s.products[s.nextID] = product
	s.nextID++

	logger.Info("Produto criado", "product", product)
	return product
}

// ProductController - Controller com rotas
type ProductController struct {
	controller.BaseController `baseUrl:"/products"`

	// Injeção de dependência
	ProductService *ProductService `inject:"ProductService"`

	// Rotas definidas com tags
	GetProducts   func() `route:"GET /"`
	GetProduct    func() `route:"GET /:id"`
	CreateProduct func() `route:"POST /"`
}

// Handlers dos métodos HTTP
func (c *ProductController) getProductsHandler() {
	products := c.ProductService.GetAllProducts()

	c.JSON(map[string]interface{}{
		"data":  products,
		"count": len(products),
	})
}

func (c *ProductController) getProductHandler() {
	// Extrai ID da URL usando Gorilla Mux
	var productID int
	if c.Request != nil {
		vars := mux.Vars(c.Request)
		if idStr, exists := vars["id"]; exists {
			if id, err := strconv.Atoi(idStr); err == nil {
				productID = id
			}
		}
	}

	if productID == 0 {
		c.JSON(map[string]interface{}{
			"error": "ID inválido",
		})
		return
	}

	product := c.ProductService.GetProductByID(productID)
	if product == nil {
		c.JSON(map[string]interface{}{
			"error": "Produto não encontrado",
		})
		return
	}

	c.JSON(map[string]interface{}{
		"data": product,
	})
}

func (c *ProductController) createProductHandler() {
	// Parse do body da requisição
	var requestData struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	if c.Request != nil && c.Request.Body != nil {
		if err := json.NewDecoder(c.Request.Body).Decode(&requestData); err != nil {
			c.JSON(map[string]interface{}{
				"error": "Body inválido",
			})
			return
		}
	}

	product := c.ProductService.CreateProduct(requestData.Name, requestData.Price)

	c.JSON(map[string]interface{}{
		"message": "Produto criado com sucesso",
		"data":    product,
	})
}

// Factory do controller
func NewProductController() *ProductController {
	controller := &ProductController{}

	// Inicializa os handlers
	controller.GetProducts = func() { controller.getProductsHandler() }
	controller.GetProduct = func() { controller.getProductHandler() }
	controller.CreateProduct = func() { controller.createProductHandler() }

	return controller
}

// Módulo com decorator
var _ = module.New(module.ModuleConfig{
	Controllers: []interface{}{NewProductController()},
	Providers: []interface{}{
		NewProductService(),
	},
})(&ProductModule{})

type ProductModule struct{}

func main() {
	logger.Info("🚀 Iniciando exemplo simples do NestGo")

	// Inicia a aplicação na porta 3001
	application.StartApplication(":3001")
}
