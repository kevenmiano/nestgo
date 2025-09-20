package server

import (
	"reflect"

	"github.com/kevenmiano/nestgo/pkg/logger"
	"github.com/kevenmiano/nestgo/pkg/module"
)

// RouteDiscovery discovers and registers routes from modules
type RouteDiscovery struct {
	server *Server
}

// NewRouteDiscovery creates a new route discovery instance
func NewRouteDiscovery(server *Server) *RouteDiscovery {
	return &RouteDiscovery{
		server: server,
	}
}

// DiscoverAndRegisterRoutes discovers routes from all registered modules
func (rd *RouteDiscovery) DiscoverAndRegisterRoutes() {
	registry := module.GetGlobalRegistry()
	modules := registry.GetAllModules()

	logger.Info("Discovering routes from modules", "moduleCount", len(modules))

	routeCount := 0
	for moduleName, moduleInstance := range modules {
		moduleRouteCount := rd.registerModuleRoutes(moduleName, moduleInstance)
		routeCount += moduleRouteCount
	}

	logger.Info("Route discovery completed", "totalRoutes", routeCount)
}

// registerModuleRoutes registers routes from a specific module
func (rd *RouteDiscovery) registerModuleRoutes(moduleName string, moduleInstance module.Module) int {
	controllers := moduleInstance.GetControllers()

	logger.Info("Registering routes for module", "module", moduleName, "controllerCount", len(controllers))

	routeCount := 0
	for _, controller := range controllers {
		controllerRouteCount := rd.registerControllerRoutes(moduleName, controller)
		routeCount += controllerRouteCount
	}

	logger.Info("Module routes registered", "module", moduleName, "routeCount", routeCount)
	return routeCount
}

// registerControllerRoutes registers routes from a controller
func (rd *RouteDiscovery) registerControllerRoutes(moduleName string, controller interface{}) int {
	controllerType := reflect.TypeOf(controller)
	controllerValue := reflect.ValueOf(controller)

	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
		controllerValue = controllerValue.Elem()
	}

	// Extract base URL from controller struct tags
	baseURL := rd.extractBaseURL(controllerType)
	if baseURL == "" {
		logger.Warn("Controller has no base URL", "controller", controllerType.Name())
		return 0
	}

	logger.Info("Registering controller routes", "controller", controllerType.Name(), "baseURL", baseURL)

	// Count methods that will be registered as routes
	routeCount := 0
	ptrType := reflect.PtrTo(controllerType)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if method.IsExported() {
			// All exported methods are potential routes (they need @route comment)
			routeCount++
		}
	}

	// Register all methods as routes
	rd.server.RegisterController(moduleName, controller, baseURL)

	logger.Info("Controller routes registered", "controller", controllerType.Name(), "routeCount", routeCount)
	return routeCount
}

// extractBaseURL extracts the base URL from controller struct tags
func (rd *RouteDiscovery) extractBaseURL(controllerType reflect.Type) string {
	// Look for BaseController field with baseUrl tag
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)

		// Check if it's a BaseController field
		if field.Type.Name() == "BaseController" {
			// Extract baseUrl from tag
			if baseURL := field.Tag.Get("baseUrl"); baseURL != "" {
				return baseURL
			}
		}
	}

	return ""
}
