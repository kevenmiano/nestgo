package router

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/pkg/logger"
	"github.com/kevenmiano/nestgo/pkg/server"
)

// Route represents a single HTTP route
type Route struct {
	Method      string
	Path        string
	Handler     interface{}
	HandlerName string
}

// Router manages HTTP routes
type Router struct {
	routes []Route
	server *server.Server
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	httpServer := server.NewServer()
	return &Router{
		routes: make([]Route, 0),
		server: httpServer,
	}
}

// RegisterController registers all routes from a controller
func (r *Router) RegisterController(controller interface{}, basePath string) error {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	// Extract routes from controller fields
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)

		// Skip BaseController field
		if field.Name == "BaseController" {
			continue
		}

		// Check if field has http tag
		httpMethod := field.Tag.Get("http")
		if httpMethod != "" {
			route := Route{
				Method:      strings.ToUpper(httpMethod),
				Path:        basePath,
				Handler:     controller,
				HandlerName: field.Name,
			}
			r.routes = append(r.routes, route)
		}
	}

	return nil
}

// GetRoutes returns all registered routes
func (r *Router) GetRoutes() []Route {
	return r.routes
}

// PrintRoutes prints all registered routes
func (r *Router) PrintRoutes() {
	logger.Info("HTTP Routes")
	for _, route := range r.routes {
		logger.Info("Route registered",
			"method", route.Method,
			"path", route.Path,
			"handler", route.HandlerName)
	}
}

// StartServer starts the HTTP server
func (r *Router) StartServer(port string) error {
	// Discover and register routes from modules
	routeDiscovery := server.NewRouteDiscovery(r.server)
	routeDiscovery.DiscoverAndRegisterRoutes()

	// Start the server
	return r.server.Start(port)
}

// Shutdown gracefully shuts down the server
func (r *Router) Shutdown(ctx context.Context) error {
	return r.server.Shutdown(ctx)
}

// HandleRequest simulates handling an HTTP request
func (r *Router) HandleRequest(method, path string) error {
	for _, route := range r.routes {
		if route.Method == strings.ToUpper(method) && route.Path == path {
			logger.Info("Handling request",
				"method", method,
				"path", path,
				"handler", route.HandlerName)

			// Simulate calling the handler
			handlerValue := reflect.ValueOf(route.Handler)
			if handlerValue.Kind() == reflect.Ptr {
				handlerValue = handlerValue.Elem()
			}

			// Try to call the method if it exists
			methodValue := handlerValue.MethodByName(route.HandlerName)
			if methodValue.IsValid() && methodValue.CanInterface() {
				// This would be the actual method call in a real implementation
				logger.Info("Executing handler", "handler", route.HandlerName)
			}

			return nil
		}
	}

	return fmt.Errorf("route %s %s not found", method, path)
}
