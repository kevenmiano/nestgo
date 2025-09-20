package decorators

import (
	"fmt"
)

// ModuleConfig represents the configuration for a module
type ModuleConfig struct {
	Controllers []interface{}
	Providers   []interface{}
	Imports     []interface{}
}

// Module creates a module configuration
func Module(config ModuleConfig) ModuleConfig {
	return config
}

// ModuleMetadata stores module information
type ModuleMetadata struct {
	Name        string
	Controllers []interface{}
	Providers   []interface{}
	Imports     []interface{}
}

// ModuleRegistry manages modules with metadata
type ModuleRegistry struct {
	modules map[string]*ModuleMetadata
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]*ModuleMetadata),
	}
}

// RegisterModule registers a module with its configuration
func (mr *ModuleRegistry) RegisterModule(moduleName string, config ModuleConfig) {
	mr.modules[moduleName] = &ModuleMetadata{
		Name:        moduleName,
		Controllers: config.Controllers,
		Providers:   config.Providers,
		Imports:     config.Imports,
	}

	fmt.Printf("📦 Module '%s' registered\n", moduleName)
	fmt.Printf("  🎮 Controllers: %d\n", len(config.Controllers))
	fmt.Printf("  🔧 Providers: %d\n", len(config.Providers))
	if len(config.Imports) > 0 {
		fmt.Printf("  📥 Imports: %d\n", len(config.Imports))
	}
}

// GetModule retrieves a module by name
func (mr *ModuleRegistry) GetModule(name string) (*ModuleMetadata, error) {
	module, exists := mr.modules[name]
	if !exists {
		return nil, fmt.Errorf("module %s not found", name)
	}
	return module, nil
}

// GetAllModules returns all registered modules
func (mr *ModuleRegistry) GetAllModules() map[string]*ModuleMetadata {
	return mr.modules
}

// PrintModules prints all registered modules
func (mr *ModuleRegistry) PrintModules() {
	fmt.Println("=== Registered Modules ===")
	for name, module := range mr.modules {
		fmt.Printf("📦 %s\n", name)
		fmt.Printf("  🎮 Controllers: %d\n", len(module.Controllers))
		fmt.Printf("  🔧 Providers: %d\n", len(module.Providers))
		if len(module.Imports) > 0 {
			fmt.Printf("  📥 Imports: %d\n", len(module.Imports))
		}
	}
}
