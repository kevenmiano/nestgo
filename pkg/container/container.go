package container

import (
	"fmt"
	"reflect"

	"github.com/kevenmiano/nestgo/pkg/logger"
)

// Container manages dependency injection
type Container struct {
	services map[string]interface{}
}

// NewContainer creates a new DI container
func NewContainer() *Container {
	return &Container{
		services: make(map[string]interface{}),
	}
}

// Register registers a service in the container
func (c *Container) Register(name string, service interface{}) {
	c.services[name] = service
}

// Get retrieves a service from the container
func (c *Container) Get(name string) (interface{}, bool) {
	service, exists := c.services[name]
	return service, exists
}

// AutoRegister automatically registers a service based on its type
func (c *Container) AutoRegister(service interface{}) {
	serviceType := reflect.TypeOf(service)
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}

	serviceName := serviceType.Name()
	c.services[serviceName] = service
	logger.Info("Service auto-registered", "name", serviceName, "type", serviceType.String())
}

// Inject injects dependencies into a target struct
func (c *Container) Inject(target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return nil
	}

	targetValue = targetValue.Elem()
	targetType := targetValue.Type()

	logger.Info("Injecting dependencies", "target", targetType.Name())
	c.DebugInjection(targetType.Name())

	var missingDependencies []string

	for i := 0; i < targetValue.NumField(); i++ {
		field := targetValue.Field(i)
		fieldType := targetType.Field(i)

		// Check if field has inject tag
		if injectTag := fieldType.Tag.Get("inject"); injectTag != "" {
			logger.Info("Found inject tag", "field", fieldType.Name, "injectTag", injectTag)
			if service, exists := c.Get(injectTag); exists {
				logger.Info("Service found for injection", "service", injectTag, "type", reflect.TypeOf(service))
				if field.CanSet() {
					field.Set(reflect.ValueOf(service))
					logger.Info("Dependency injected successfully", "field", fieldType.Name, "service", injectTag)
				} else {
					logger.Warn("Cannot set field", "field", fieldType.Name)
					missingDependencies = append(missingDependencies, fmt.Sprintf("field %s (cannot set)", fieldType.Name))
				}
			} else {
				logger.Error("Service not found for injection", "service", injectTag)
				missingDependencies = append(missingDependencies, fmt.Sprintf("service %s (not found)", injectTag))
			}
		}
	}

	// Se há dependências faltando, retorna erro fatal
	if len(missingDependencies) > 0 {
		errorMsg := fmt.Sprintf("CRITICAL: Failed to inject dependencies for %s. Missing: %v",
			targetType.Name(), missingDependencies)
		logger.Error(errorMsg)
		return fmt.Errorf("%s", errorMsg)
	}

	return nil
}

// GetAllServices returns all registered services
func (c *Container) GetAllServices() map[string]interface{} {
	return c.services
}

// PrintServices prints all registered services
func (c *Container) PrintServices() {
	logger.Info("Registered Services", "count", len(c.services))
	for name, service := range c.services {
		serviceType := reflect.TypeOf(service)
		logger.Info("Service registered", "name", name, "type", serviceType.String())
	}
}

// DebugInjection prints debug info for injection
func (c *Container) DebugInjection(targetName string) {
	logger.Info("=== DEBUG INJECTION ===", "target", targetName)
	logger.Info("Available services", "count", len(c.services))
	for name, service := range c.services {
		serviceType := reflect.TypeOf(service)
		logger.Info("Available service", "name", name, "type", serviceType.String())
	}
}
