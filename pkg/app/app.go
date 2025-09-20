package app

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/pkg/container"
	controllerPkg "github.com/kevenmiano/nestgo/pkg/controller"
	"github.com/kevenmiano/nestgo/pkg/decorators"
	"github.com/kevenmiano/nestgo/pkg/logger"
	"github.com/kevenmiano/nestgo/pkg/module"
	"github.com/kevenmiano/nestgo/pkg/router"
)

// App represents the main application
type App struct {
	diContainer *container.Container
	router      *router.Router
}

// NewApp creates a new application instance
func NewApp() *App {
	return &App{
		diContainer: container.NewContainer(),
		router:      router.NewRouter(),
	}
}

// RegisterModule registers a module in the application
func (app *App) RegisterModule(moduleInstance module.Module) {
	module.GetGlobalRegistry().RegisterModule(moduleInstance)

	// Auto-register controllers and services
	controllers := moduleInstance.GetControllers()
	services := moduleInstance.GetServices()

	// Register services in DI container
	for _, service := range services {
		app.diContainer.AutoRegister(service)
	}

	// Register controllers and their routes
	for _, controller := range controllers {
		app.registerControllerRoutes(controller)
	}
}

// AutoDiscoverModules automatically discovers and registers modules
func (app *App) AutoDiscoverModules(modules ...interface{}) {
	for _, module := range modules {
		app.autoRegisterModule(module)
	}
}

// RegisterAutoDiscoveredModules registers all modules that were auto-discovered
func (app *App) RegisterAutoDiscoveredModules() {
	// Get all modules from the global registry
	globalRegistry := module.GetGlobalRegistry()
	modules := globalRegistry.GetAllModules()

	for _, module := range modules {
		// Register controllers and services
		controllers := module.GetControllers()
		services := module.GetServices()

		// Register services in DI container
		for _, service := range services {
			app.diContainer.AutoRegister(service)
		}

		// Register controllers and their routes
		for _, controller := range controllers {
			app.registerControllerRoutes(controller)
		}
	}
}

// RegisterModuleAndStart registers a module and starts the application
func (app *App) RegisterModuleAndStart(moduleStruct interface{}, port string) error {
	// Auto-register the module
	module.AutoRegisterModule(module.ExtractModuleFromStruct(moduleStruct))

	// Register all auto-discovered modules
	app.RegisterAutoDiscoveredModules()

	if err := app.InjectDependencies(); err != nil {
		logger.Error("FATAL: Application startup failed due to dependency injection errors")
		return err
	}

	// Start the application
	return app.Start(port)
}

// autoRegisterModule automatically registers a module using reflection
func (app *App) autoRegisterModule(moduleInstance interface{}) {
	moduleType := reflect.TypeOf(moduleInstance)
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	moduleName := moduleType.Name()
	logger.Info("Auto-discovering module", "name", moduleName)

	// Extract module configuration using reflection
	config := app.extractModuleConfig(moduleInstance)

	// Register the module in global registry
	if moduleWrapper, ok := moduleInstance.(module.Module); ok {
		module.GetGlobalRegistry().RegisterModule(moduleWrapper)
	}

	// Register providers in DI container
	for _, provider := range config.Providers {
		app.diContainer.AutoRegister(provider)
	}

	// Register controllers and their routes
	for _, controller := range config.Controllers {
		app.registerControllerRoutes(controller)
	}
}

// extractModuleConfig extracts module configuration using reflection
func (app *App) extractModuleConfig(module interface{}) decorators.ModuleConfig {
	moduleType := reflect.TypeOf(module)
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	moduleValue := reflect.ValueOf(module)
	if moduleValue.Kind() == reflect.Ptr {
		moduleValue = moduleValue.Elem()
	}

	config := decorators.ModuleConfig{
		Controllers: make([]interface{}, 0),
		Providers:   make([]interface{}, 0),
		Imports:     make([]interface{}, 0),
	}

	// Look for fields that represent controllers and services
	for i := 0; i < moduleType.NumField(); i++ {
		field := moduleType.Field(i)
		fieldValue := moduleValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Check if field has a tag indicating it's a controller or service
		if field.Tag.Get("controller") != "" || field.Tag.Get("service") != "" {
			if fieldValue.CanInterface() {
				instance := fieldValue.Interface()

				// Check if it's a controller
				controllerExtractor := controllerPkg.NewMetaExtractor()
				if controllerExtractor.IsController(instance) {
					config.Controllers = append(config.Controllers, instance)
				} else {
					// Assume it's a service/provider
					config.Providers = append(config.Providers, instance)
				}
			}
		}
	}

	return config
}

// registerControllerRoutes registers routes for a controller
func (app *App) registerControllerRoutes(controller interface{}) {
	// Check if it's a controller
	controllerExtractor := controllerPkg.NewMetaExtractor()
	if !controllerExtractor.IsController(controller) {
		return
	}

	// Get base URL
	baseURL := controllerExtractor.GetControllerBaseURL(controller)
	if baseURL == "" {
		logger.Warn("Controller has no baseUrl", "controller", controllerExtractor.GetControllerName(controller))
		return
	}

	// Register routes
	if err := app.router.RegisterController(controller, baseURL); err != nil {
		logger.Error("Error registering controller routes", "error", err)
		return
	}

	logger.Info("Controller registered",
		"name", controllerExtractor.GetControllerName(controller),
		"baseUrl", baseURL)
}

// InjectDependencies injects dependencies into all registered controllers and services
func (app *App) InjectDependencies() error {
	modules := module.GetGlobalRegistry().GetAllModules()
	var injectionErrors []string

	for _, module := range modules {
		controllers := module.GetControllers()
		services := module.GetServices()

		// Inject dependencies into services first
		for _, service := range services {
			serviceType := reflect.TypeOf(service)
			if serviceType.Kind() == reflect.Ptr {
				serviceType = serviceType.Elem()
			}
			serviceName := serviceType.Name()

			if err := app.diContainer.Inject(service); err != nil {
				errorMsg := fmt.Sprintf("Service %s: %v", serviceName, err)
				logger.Error("DI Error for service", "service", serviceName, "error", err)
				injectionErrors = append(injectionErrors, errorMsg)
			} else {
				logger.Info("Dependencies injected successfully",
					"service", serviceName)
			}
		}

		// Then inject dependencies into controllers
		for _, controller := range controllers {
			controllerExtractor := controllerPkg.NewMetaExtractor()
			controllerName := controllerExtractor.GetControllerName(controller)

			if err := app.diContainer.Inject(controller); err != nil {
				errorMsg := fmt.Sprintf("Controller %s: %v", controllerName, err)
				logger.Error("DI Error for controller", "controller", controllerName, "error", err)
				injectionErrors = append(injectionErrors, errorMsg)
			} else {
				logger.Info("Dependencies injected successfully",
					"controller", controllerName)
			}
		}
	}

	// Se há erros de injeção, falha a aplicação
	if len(injectionErrors) > 0 {
		fatalError := fmt.Sprintf("CRITICAL: Dependency injection failed for %d components:\n%s",
			len(injectionErrors), strings.Join(injectionErrors, "\n"))
		logger.Error(fatalError)
		return fmt.Errorf("%s", fatalError)
	}

	logger.Info("All dependencies injected successfully")
	return nil
}

// Start starts the application
func (app *App) Start(port string) error {
	logger.Info("Starting NestGo application")

	// Print modules
	module.GetGlobalRegistry().PrintModules()

	// Print services
	logger.Info("DI Container Services")
	app.diContainer.PrintServices()

	// Print routes
	app.router.PrintRoutes()

	// Start server
	return app.router.StartServer(port)
}

// TestRoute tests a specific route
func (app *App) TestRoute(method, path string) {
	logger.Info("Testing route", "method", method, "path", path)
	if err := app.router.HandleRequest(method, path); err != nil {
		logger.Error("Route test failed", "error", err)
	}
}

// GetContainer returns the DI container
func (app *App) GetContainer() *container.Container {
	return app.diContainer
}

// RegisterController registers a single controller
func (app *App) RegisterController(controller interface{}) {
	app.registerControllerRoutes(controller)
}
