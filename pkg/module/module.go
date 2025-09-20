package module

import (
	"reflect"
)

// Module interface defines the contract for all modules
type Module interface {
	GetModuleName() string
	GetControllers() []interface{}
	GetServices() []interface{}
	GetImports() []Module
}

// BaseModule provides base functionality for all modules
type BaseModule struct {
	_ string `module:"true"`
}

// NewModule creates a new module and automatically registers it
func NewModule[T any](moduleStruct T) T {
	AutoDetectModuleCreation(moduleStruct)
	return moduleStruct
}

// GetModuleName returns the name of the module using reflection
func (bm *BaseModule) GetModuleName() string {
	structType := reflect.TypeOf(bm)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Name() == "BaseModule" {
		return "BaseModule"
	}

	return structType.Name()
}
