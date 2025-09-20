package server

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kevenmiano/nestgo/pkg/logger"
)

// Server represents the HTTP server
type Server struct {
	router *mux.Router
	server *http.Server
}

// responseTracker tracks if a response has been written
type responseTracker struct {
	http.ResponseWriter
	written bool
}

func (rt *responseTracker) Write(data []byte) (int, error) {
	rt.written = true
	return rt.ResponseWriter.Write(data)
}

func (rt *responseTracker) WriteHeader(statusCode int) {
	rt.written = true
	rt.ResponseWriter.WriteHeader(statusCode)
}

// NewServer creates a new HTTP server
func NewServer() *Server {
	router := mux.NewRouter()

	// Test: Register a simple parameterized route directly
	router.HandleFunc("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		logger.Info("Test route hit", "id", id)
		w.Write([]byte("Test route works! ID: " + id))
	}).Methods("GET")

	return &Server{
		router: router,
	}
}

// RegisterRoute registers a route with the server
func (s *Server) RegisterRoute(method, path string, handler http.HandlerFunc) {
	// Convert :id syntax to {id} syntax for Gorilla Mux
	convertedPath := strings.ReplaceAll(path, ":id", "{id}")

	route := s.router.HandleFunc(convertedPath, handler).Methods(method)
	logger.Info("Route registered", "method", method, "originalPath", path, "convertedPath", convertedPath, "route", route)

	// Debug: Test route matching after all routes are registered
	if path == "/users/:id" && method == "PATCH" {
		logger.Info("Testing route matching for /users/:id after all routes registered")
		testReq, _ := http.NewRequest("GET", "http://localhost:3000/users/1", nil)
		match := &mux.RouteMatch{}
		if s.router.Match(testReq, match) {
			logger.Info("Route match found", "route", match.Route)
		} else {
			logger.Warn("No route match found for /users/1")
		}
	}
}

// RegisterController registers all routes from a controller
func (s *Server) RegisterController(moduleName string, controller interface{}, basePath string) {
	controllerType := reflect.TypeOf(controller)
	controllerValue := reflect.ValueOf(controller)

	// Don't dereference the pointer - we need the pointer type to get methods
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()

		controllerValue = controllerValue.Elem()
	}

	logger.Info("Processing controller fields", "controller", controllerType.Name(), "basePath", basePath, "fieldCount", controllerType.NumField())

	// First pass: register parameterized routes
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)
		fieldValue := controllerValue.Field(i)

		// Skip BaseController and non-function fields
		if field.Name == "BaseController" || field.Type.Kind() != reflect.Func {
			continue
		}

		// Check if field has route tag
		routeTag := field.Tag.Get("route")
		if routeTag == "" {
			continue
		}

		// Parse route tag: "METHOD /path"
		parts := strings.Fields(routeTag)
		if len(parts) != 2 {
			continue
		}

		httpMethod := strings.ToUpper(parts[0])
		subPath := parts[1]

		// Only register parameterized routes first
		if strings.Contains(subPath, ":") {
			// Combine basePath with subPath
			fullPath := strings.TrimSuffix(basePath, "/") + subPath

			logger.Info("Registering parameterized route", "field", field.Name, "httpMethod", httpMethod, "fullPath", fullPath)

			// Create handler function with controller instance
			handler := s.createHandlerWithField(fieldValue, controllerValue)

			// Register the route
			s.RegisterRoute(httpMethod, fullPath, handler)
		}
	}

	// Second pass: register non-parameterized routes
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)
		fieldValue := controllerValue.Field(i)

		// Skip BaseController and non-function fields
		if field.Name == "BaseController" || field.Type.Kind() != reflect.Func {
			continue
		}

		// Check if field has route tag
		routeTag := field.Tag.Get("route")
		if routeTag == "" {
			continue
		}

		// Parse route tag: "METHOD /path"
		parts := strings.Fields(routeTag)
		if len(parts) != 2 {
			continue
		}

		httpMethod := strings.ToUpper(parts[0])
		subPath := parts[1]

		// Only register non-parameterized routes
		if !strings.Contains(subPath, ":") {
			// Combine basePath with subPath
			fullPath := strings.TrimSuffix(basePath, "/") + subPath

			logger.Info("Registering non-parameterized route", "field", field.Name, "httpMethod", httpMethod, "fullPath", fullPath)

			// Create handler function with controller instance
			handler := s.createHandlerWithField(fieldValue, controllerValue)

			// Register the route
			s.RegisterRoute(httpMethod, fullPath, handler)
		}
	}
}

// serializeToJSON serializes data to JSONll
func (s *Server) serializeToJSON(data interface{}) ([]byte, error) {
	// Handle different types of data
	switch v := data.(type) {
	case []string:
		// For string slices, wrap in a response object
		response := map[string]interface{}{
			"data":  v,
			"count": len(v),
		}
		return json.Marshal(response)
	case map[string]interface{}:
		// For maps, return as is
		return json.Marshal(v)
	case string:
		// For strings, wrap in a response object
		response := map[string]interface{}{
			"message": v,
		}
		return json.Marshal(response)
	default:
		// For other types, try to marshal directly
		return json.Marshal(data)
	}
}

// createHandlerWithField creates an HTTP handler with controller field
func (s *Server) createHandlerWithField(fieldValue reflect.Value, controllerValue reflect.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Debug: Log incoming request
		logger.Info("Incoming request", "method", r.Method, "path", r.URL.Path, "rawQuery", r.URL.RawQuery)

		// Create a custom ResponseWriter to track if response was written
		responseWriter := &responseTracker{ResponseWriter: w}

		// Set HTTP context in BaseController if it exists
		logger.Info("Setting HTTP context", "controllerType", controllerValue.Type().Name())
		s.setHTTPContext(controllerValue, responseWriter, r)

		// Call the field function directly
		results := fieldValue.Call([]reflect.Value{})

		// Only write default response if no response was written by the controller
		if !responseWriter.written {
			// Handle the response
			if len(results) > 0 {
				result := results[0].Interface()
				if result != nil {
					// Serialize to JSON
					jsonData, err := s.serializeToJSON(result)
					if err != nil {
						logger.Error("Failed to serialize response", "error", err)
						http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
						return
					}

					logger.Info("Controller field executed", "result", result)
					w.Write(jsonData)
				} else {
					// No data returned
					w.Write([]byte(`{"message": "No data returned"}`))
				}
			} else {
				// No return value
				w.Write([]byte(`{"message": "Field executed successfully"}`))
			}
		}

		logger.Info("Request handled", "method", r.Method, "path", r.URL.Path)
	}
}

// setHTTPContext sets the HTTP context in the BaseController
func (s *Server) setHTTPContext(controller reflect.Value, w http.ResponseWriter, r *http.Request) {
	// Look for BaseController field
	for i := 0; i < controller.NumField(); i++ {
		field := controller.Field(i)
		fieldType := controller.Type().Field(i)

		if fieldType.Name == "BaseController" {
			// Get the address of the field to set the HTTP context
			baseControllerPtr := field.Addr()

			// Set HTTP context in BaseController
			if baseController, ok := baseControllerPtr.Interface().(interface {
				SetHTTPContext(http.ResponseWriter, *http.Request)
			}); ok {
				logger.Info("Setting HTTP context in BaseController")
				baseController.SetHTTPContext(w, r)
			} else {
				logger.Warn("Failed to set HTTP context - type assertion failed")
			}
			break
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	s.server = &http.Server{
		Addr:         port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Server starting", "port", port)

	// Print all registered routes
	s.PrintRoutes()

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		logger.Info("Shutting down server...")
		return s.server.Shutdown(ctx)
	}
	return nil
}

// PrintRoutes prints all registered routes
func (s *Server) PrintRoutes() {
	logger.Info("HTTP Routes registered")

	// Walk through all routes
	s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()

		for _, method := range methods {
			logger.Info("Available route", "method", method, "path", pathTemplate)
		}
		return nil
	})
}
