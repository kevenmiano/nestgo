package module

import (
	"fmt"
	"sync"

	"github.com/kevenmiano/nestgo/pkg/logger"
)

// ModuleRegistry manages all registered modules
type ModuleRegistry struct {
	modules map[string]Module
	mutex   sync.RWMutex
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]Module),
	}
}

// RegisterModule registers a module in the registry
func (mr *ModuleRegistry) RegisterModule(module Module) {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	moduleName := module.GetModuleName()
	mr.modules[moduleName] = module
	logger.Info("Module registered", "name", moduleName)
}

// GetModule retrieves a module by name
func (mr *ModuleRegistry) GetModule(name string) (Module, error) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	module, exists := mr.modules[name]
	if !exists {
		return nil, fmt.Errorf("module %s not found", name)
	}
	return module, nil
}

// GetAllModules returns all registered modules
func (mr *ModuleRegistry) GetAllModules() map[string]Module {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	// Return a copy to avoid race conditions
	modules := make(map[string]Module)
	for name, module := range mr.modules {
		modules[name] = module
	}
	return modules
}

// PrintModules prints all registered modules
func (mr *ModuleRegistry) PrintModules() {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	logger.Info("Registered modules", "count", len(mr.modules))
	for name, module := range mr.modules {
		controllers := module.GetControllers()
		services := module.GetServices()
		imports := module.GetImports()

		logger.Info("Module details",
			"name", name,
			"controllers", len(controllers),
			"services", len(services),
			"imports", len(imports))
	}
}

// IsModuleRegistered checks if a module is already registered
func (mr *ModuleRegistry) IsModuleRegistered(moduleName string) bool {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	_, exists := mr.modules[moduleName]
	return exists
}

// Global registry instance
var globalRegistry *ModuleRegistry
var initOnce sync.Once

// GetGlobalRegistry returns the global module registry (auto-initializes)
func GetGlobalRegistry() *ModuleRegistry {
	initOnce.Do(func() {
		globalRegistry = NewModuleRegistry()
	})
	return globalRegistry
}
