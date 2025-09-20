package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kevenmiano/nestgo/pkg/application"
	"github.com/kevenmiano/nestgo/pkg/controller"
	"github.com/kevenmiano/nestgo/pkg/logger"
	"github.com/kevenmiano/nestgo/pkg/module"
	"github.com/kevenmiano/nestgo/pkg/service"
)

// User model
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age,omitempty"`
}

// FakeDatabase - Database em memória
type FakeDatabase struct {
	users  map[int]*User
	nextID int
	mutex  sync.RWMutex
}

// NewFakeDatabase cria uma nova instância do database fake
func NewFakeDatabase() *FakeDatabase {
	db := &FakeDatabase{
		users:  make(map[int]*User),
		nextID: 1,
	}

	// Adiciona alguns usuários de exemplo
	db.users[1] = &User{ID: 1, Name: "João Silva", Email: "joao@example.com", Age: 30}
	db.users[2] = &User{ID: 2, Name: "Maria Santos", Email: "maria@example.com", Age: 25}
	db.users[3] = &User{ID: 3, Name: "Pedro Costa", Email: "pedro@example.com", Age: 35}
	db.nextID = 4

	logger.Info("FakeDatabase created with sample data", "users", len(db.users))
	return db
}

// CreateUser cria um novo usuário no database
func (db *FakeDatabase) CreateUser(name, email string, age int) *User {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	user := &User{
		ID:    db.nextID,
		Name:  name,
		Email: email,
		Age:   age,
	}

	db.users[db.nextID] = user
	db.nextID++

	logger.Info("User created in fake database", "user", user)
	return user
}

// GetAllUsers retorna todos os usuários
func (db *FakeDatabase) GetAllUsers() []*User {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	users := make([]*User, 0, len(db.users))
	for _, user := range db.users {
		users = append(users, user)
	}

	return users
}

// GetUserByID retorna um usuário por ID
func (db *FakeDatabase) GetUserByID(id int) *User {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	return db.users[id]
}

// UpdateUser atualiza um usuário existente
func (db *FakeDatabase) UpdateUser(id int, name, email string, age int) *User {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if user, exists := db.users[id]; exists {
		user.Name = name
		user.Email = email
		user.Age = age
		logger.Info("User updated in fake database", "user", user)
		return user
	}

	return nil
}

// DeleteUser remove um usuário
func (db *FakeDatabase) DeleteUser(id int) bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if _, exists := db.users[id]; exists {
		delete(db.users, id)
		logger.Info("User deleted from fake database", "id", id)
		return true
	}

	return false
}

// UserService com database fake
type UserService struct {
	service.BaseService
	Database *FakeDatabase `inject:"FakeDatabase"`
}

func NewUserService() *UserService {
	return &UserService{
		BaseService: service.BaseService{},
	}
}

type UserController struct {
	controller.BaseController `baseUrl:"/users"`

	// DI fields
	UserService *UserService `inject:"UserService"`

	// Route fields with explicit HTTP method and path tags
	GetUsers     func() `route:"GET /"`
	CreateUser   func() `route:"POST /"`
	GetUser      func() `route:"GET /:id"`
	UpdateUser   func() `route:"PUT /:id"`
	DeleteUser   func() `route:"DELETE /:id"`
	PatchUser    func() `route:"PATCH /:id"`
	HeadUsers    func() `route:"HEAD /"`
	OptionsUsers func() `route:"OPTIONS /"`
}

// UserService methods
func (s *UserService) GetAllUsers() []*User {
	if s.Database == nil {
		logger.Error("Database is nil in UserService")
		return []*User{}
	}

	users := s.Database.GetAllUsers()
	logger.Info("Retrieved users from database", "count", len(users))
	return users
}

func (s *UserService) CreateUser(name, email string, age int) *User {
	logger.Info("UserService.CreateUser called", "database", s.Database != nil)
	if s.Database == nil {
		logger.Error("Database is nil in UserService")
		return nil
	}

	user := s.Database.CreateUser(name, email, age)
	logger.Info("Created user via service", "user", user)
	return user
}

func (s *UserService) GetUserByID(id int) *User {
	if s.Database == nil {
		logger.Error("Database is nil in UserService")
		return nil
	}

	return s.Database.GetUserByID(id)
}

func (s *UserService) UpdateUser(id int, name, email string, age int) *User {
	if s.Database == nil {
		logger.Error("Database is nil in UserService")
		return nil
	}

	return s.Database.UpdateUser(id, name, email, age)
}

func (s *UserService) DeleteUser(id int) bool {
	if s.Database == nil {
		logger.Error("Database is nil in UserService")
		return false
	}

	return s.Database.DeleteUser(id)
}

// HTTP method implementations
func (c *UserController) getUsersHandler() {
	users := c.UserService.GetAllUsers()
	logger.Info("GET /users", "count", len(users))

	c.JSON(map[string]interface{}{
		"data":  users,
		"count": len(users),
	})
}

func (c *UserController) createUserHandler() {
	// Parse request body
	var requestData struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	if c.Request != nil && c.Request.Body != nil {
		if err := json.NewDecoder(c.Request.Body).Decode(&requestData); err != nil {
			logger.Error("Failed to parse request body", "error", err)
			c.JSON(map[string]interface{}{
				"error": "Invalid request body",
			})
			return
		}
	} else {
		// Default values for testing
		requestData.Name = "Test User"
		requestData.Email = "test@example.com"
		requestData.Age = 25
	}

	user := c.UserService.CreateUser(requestData.Name, requestData.Email, requestData.Age)
	if user == nil {
		logger.Error("Failed to create user")
		c.JSON(map[string]interface{}{
			"error": "Failed to create user",
		})
		return
	}

	logger.Info("POST /users", "user", user)
	c.JSON(map[string]interface{}{
		"message": "User created successfully",
		"data":    user,
		"status":  "success",
	})
}

func (c *UserController) getUserHandler() {
	if c.UserService == nil {
		logger.Error("UserService is nil - dependency injection failed")
		c.JSON(map[string]interface{}{
			"error": "Internal server error - service not available",
		})
		return
	}

	// Extract ID from URL path using Gorilla Mux
	var userID int
	if c.Request != nil {
		vars := mux.Vars(c.Request)
		if idStr, exists := vars["id"]; exists {
			if id, err := strconv.Atoi(idStr); err == nil {
				userID = id
			}
		}
	}

	if userID == 0 {
		c.JSON(map[string]interface{}{
			"error": "Invalid user ID",
		})
		return
	}

	user := c.UserService.GetUserByID(userID)
	if user == nil {
		c.JSON(map[string]interface{}{
			"error": "User not found",
		})
		return
	}

	logger.Info("GET /users/:id", "user", user)
	c.JSON(map[string]interface{}{
		"data": user,
	})
}

func (c *UserController) updateUserHandler() {
	if c.UserService == nil {
		logger.Error("UserService is nil - dependency injection failed")
		c.JSON(map[string]interface{}{
			"error": "Internal server error - service not available",
		})
		return
	}

	// Extract ID from URL path
	var userID int
	if c.Request != nil {
		path := c.Request.URL.Path
		if len(path) > 7 && path[:7] == "/users/" {
			if id, err := strconv.Atoi(path[7:]); err == nil {
				userID = id
			}
		}
	}

	if userID == 0 {
		c.JSON(map[string]interface{}{
			"error": "Invalid user ID",
		})
		return
	}

	// Parse request body
	var requestData struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	if c.Request != nil && c.Request.Body != nil {
		if err := json.NewDecoder(c.Request.Body).Decode(&requestData); err != nil {
			logger.Error("Failed to parse request body", "error", err)
			c.JSON(map[string]interface{}{
				"error": "Invalid request body",
			})
			return
		}
	}

	user := c.UserService.UpdateUser(userID, requestData.Name, requestData.Email, requestData.Age)
	if user == nil {
		c.JSON(map[string]interface{}{
			"error": "User not found or update failed",
		})
		return
	}

	logger.Info("PUT /users/:id", "user", user)
	c.JSON(map[string]interface{}{
		"message": "User updated successfully",
		"data":    user,
		"status":  "success",
	})
}

func (c *UserController) deleteUserHandler() {
	if c.UserService == nil {
		logger.Error("UserService is nil - dependency injection failed")
		c.JSON(map[string]interface{}{
			"error": "Internal server error - service not available",
		})
		return
	}

	// Extract ID from URL path
	var userID int
	if c.Request != nil {
		path := c.Request.URL.Path
		if len(path) > 7 && path[:7] == "/users/" {
			if id, err := strconv.Atoi(path[7:]); err == nil {
				userID = id
			}
		}
	}

	if userID == 0 {
		c.JSON(map[string]interface{}{
			"error": "Invalid user ID",
		})
		return
	}

	success := c.UserService.DeleteUser(userID)
	if !success {
		c.JSON(map[string]interface{}{
			"error": "User not found or delete failed",
		})
		return
	}

	logger.Info("DELETE /users/:id", "id", userID)
	c.JSON(map[string]interface{}{
		"message": "User deleted successfully",
		"status":  "success",
	})
}

func (c *UserController) patchUserHandler() {
	if c.UserService == nil {
		logger.Error("UserService is nil - dependency injection failed")
		c.JSON(map[string]interface{}{
			"error": "Internal server error - service not available",
		})
		return
	}

	// Extract ID from URL path
	var userID int
	if c.Request != nil {
		path := c.Request.URL.Path
		if len(path) > 7 && path[:7] == "/users/" {
			if id, err := strconv.Atoi(path[7:]); err == nil {
				userID = id
			}
		}
	}

	if userID == 0 {
		c.JSON(map[string]interface{}{
			"error": "Invalid user ID",
		})
		return
	}

	// Get existing user
	existingUser := c.UserService.GetUserByID(userID)
	if existingUser == nil {
		c.JSON(map[string]interface{}{
			"error": "User not found",
		})
		return
	}

	// Parse request body for partial update
	var requestData struct {
		Name  *string `json:"name,omitempty"`
		Email *string `json:"email,omitempty"`
		Age   *int    `json:"age,omitempty"`
	}

	if c.Request != nil && c.Request.Body != nil {
		if err := json.NewDecoder(c.Request.Body).Decode(&requestData); err != nil {
			logger.Error("Failed to parse request body", "error", err)
			c.JSON(map[string]interface{}{
				"error": "Invalid request body",
			})
			return
		}
	}

	// Update only provided fields
	name := existingUser.Name
	email := existingUser.Email
	age := existingUser.Age

	if requestData.Name != nil {
		name = *requestData.Name
	}
	if requestData.Email != nil {
		email = *requestData.Email
	}
	if requestData.Age != nil {
		age = *requestData.Age
	}

	user := c.UserService.UpdateUser(userID, name, email, age)
	if user == nil {
		c.JSON(map[string]interface{}{
			"error": "User not found or update failed",
		})
		return
	}

	logger.Info("PATCH /users/:id", "user", user)
	c.JSON(map[string]interface{}{
		"message": "User patched successfully",
		"data":    user,
		"status":  "success",
	})
}

func (c *UserController) headUsersHandler() {
	users := c.UserService.GetAllUsers()

	// Set headers for HEAD request
	if c.ResponseWriter != nil {
		c.ResponseWriter.Header().Set("Content-Type", "application/json")
		c.ResponseWriter.Header().Set("X-Total-Count", strconv.Itoa(len(users)))
		c.ResponseWriter.WriteHeader(http.StatusOK)
	}

	logger.Info("HEAD /users", "count", len(users))
}

func (c *UserController) optionsUsersHandler() {
	if c.ResponseWriter != nil {
		c.ResponseWriter.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.ResponseWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.ResponseWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.ResponseWriter.WriteHeader(http.StatusOK)
	}

	logger.Info("OPTIONS /users")
}

// UserModule follows NestJS pattern with decorators
type UserModule struct{}

// Create controller instance with initialized route handlers
func NewUserController() *UserController {
	controller := &UserController{}

	// Initialize route handlers
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

// Define the module with decorator (like NestJS @Module)
var _ = module.New(module.ModuleConfig{
	Controllers: []interface{}{NewUserController()},
	Providers: []interface{}{
		NewFakeDatabase(),
		NewUserService(), // Service registrado depois para receber a injeção
	},
})(&UserModule{})

func main() {
	logger.Info("🚀 Starting NestGo")

	// Start application (graceful shutdown is handled internally)
	application.StartApplication(":3000")
}
