package module

import (
	"reflect"
)

// ModuleConfig represents the configuration for a module
type ModuleConfig struct {
	Controllers []interface{}
	Providers   []interface{}
	Imports     []interface{}
	Exports     []interface{}
}

// Module decorator function that registers a module (like NestJS @Module)
func New(config ModuleConfig) func(interface{}) interface{} {
	return func(moduleStruct interface{}) interface{} {
		// Auto-register the module with its configuration
		AutoRegisterModuleWithConfig(moduleStruct, config)
		return moduleStruct
	}
}

// AutoRegisterModuleWithConfig automatically registers a module with its configuration
func AutoRegisterModuleWithConfig(moduleStruct interface{}, config ModuleConfig) {
	// Create a module wrapper with the configuration
	moduleType := reflect.TypeOf(moduleStruct)
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	// Create a configured module wrapper
	configuredModule := &ConfiguredModuleWrapper{
		name:       moduleType.Name(),
		moduleType: moduleType,
		instance:   moduleStruct,
		config:     config,
	}

	// Register the module
	AutoRegisterModule(configuredModule)
}

// ConfiguredModuleWrapper wraps a struct with module configuration
type ConfiguredModuleWrapper struct {
	name       string
	moduleType reflect.Type
	instance   interface{}
	config     ModuleConfig
}

// GetModuleName returns the module name
func (cmw *ConfiguredModuleWrapper) GetModuleName() string {
	return cmw.name
}

// GetControllers returns controllers from the module configuration
func (cmw *ConfiguredModuleWrapper) GetControllers() []interface{} {
	return cmw.config.Controllers
}

// GetServices returns services/providers from the module configuration
func (cmw *ConfiguredModuleWrapper) GetServices() []interface{} {
	return cmw.config.Providers
}

// GetImports returns imported modules
func (cmw *ConfiguredModuleWrapper) GetImports() []Module {
	imports := make([]Module, 0, len(cmw.config.Imports))
	for _, imp := range cmw.config.Imports {
		if module, ok := imp.(Module); ok {
			imports = append(imports, module)
		}
	}
	return imports
}
