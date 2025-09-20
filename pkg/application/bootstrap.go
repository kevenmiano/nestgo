package application

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kevenmiano/nestgo/pkg/app"
	"github.com/kevenmiano/nestgo/pkg/logger"
	"github.com/kevenmiano/nestgo/pkg/module"
)

// Bootstrap creates and auto-registers a module
func Bootstrap(moduleStruct interface{}) *Application {
	// Auto-register the module
	module.AutoRegisterOnCreate(moduleStruct)

	// Create application
	application := NewApplication()

	// Register the module in the application
	application.RegisterModule(moduleStruct)

	return application
}

// StartApplication starts the application with auto-discovered modules and graceful shutdown
func StartApplication(port string) {
	logger.Info("Starting NestGo application with auto-discovery", "port", port)
	logger.Info("DEBUG: StartApplication called")

	// Setup graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start graceful shutdown handler in goroutine
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received, gracefully shutting down...")
		cancel()
	}()

	// Get all auto-registered modules
	registry := module.GetGlobalRegistry()
	modules := registry.GetAllModules()

	if len(modules) == 0 {
		logger.Warn("No modules found. Make sure to create modules that inherit from BaseModule.")
		return
	}

	// Create application
	app := app.NewApp()

	// Register all auto-discovered modules
	for _, module := range modules {
		// Register controllers and services
		controllers := module.GetControllers()
		services := module.GetServices()

		// Register services in DI container
		for _, service := range services {
			app.GetContainer().AutoRegister(service)
		}

		// Register controllers and their routes
		for _, controller := range controllers {
			app.RegisterController(controller)
		}
	}

	logger.Info("DEBUG: About to inject dependencies")
	if err := app.InjectDependencies(); err != nil {
		logger.Error("FATAL: Application startup failed due to dependency injection errors", "error", err)
		return
	}
	logger.Info("DEBUG: Dependencies injected successfully")

	// Start the application
	if err := app.Start(port); err != nil {
		logger.Error("Failed to start application", "error", err)
		return
	}

	// Give some time for cleanup
	time.Sleep(2 * time.Second)
	logger.Info("Application shutdown complete")
}

// CreateModule creates a module that auto-registers itself
func CreateModule(moduleStruct interface{}) interface{} {
	// Auto-register the module
	module.AutoRegisterOnCreate(moduleStruct)
	return moduleStruct
}
