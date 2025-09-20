package module

import (
	"github.com/kevenmiano/nestgo/pkg/logger"
)

// AutoRegisterModule automatically registers a module when it's created
func AutoRegisterModule(module Module) {
	registry := GetGlobalRegistry()

	if !registry.IsModuleRegistered(module.GetModuleName()) {
		registry.RegisterModule(module)
		logger.Info("Auto-registered module", "name", module.GetModuleName())
	}
}

// AutoRegisterModuleFromStruct automatically registers a module from a struct
func AutoRegisterModuleFromStruct(moduleStruct interface{}) {
	module := ExtractModuleFromStruct(moduleStruct)
	if module != nil {
		AutoRegisterModule(module)
	}
}

// AutoRegisterOnCreate automatically registers a module when it's created
func AutoRegisterOnCreate(moduleStruct interface{}) {
	if IsModule(moduleStruct) {
		AutoRegisterModuleFromStruct(moduleStruct)
	}
}

// AutoDetectModuleCreation automatically detects when a module is created
func AutoDetectModuleCreation(moduleStruct interface{}) {
	AutoRegisterOnCreate(moduleStruct)
}
