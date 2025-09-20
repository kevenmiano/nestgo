package module

import (
	"reflect"
)

// IsModule checks if a struct is a module by looking for BaseModule embedding
func IsModule(moduleStruct interface{}) bool {
	moduleType := reflect.TypeOf(moduleStruct)
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	// Check if the struct embeds BaseModule
	for i := 0; i < moduleType.NumField(); i++ {
		field := moduleType.Field(i)
		if field.Name == "BaseModule" {
			return true
		}
	}
	return false
}

// ExtractModuleFromStruct extracts module information from a struct
func ExtractModuleFromStruct(moduleStruct interface{}) Module {
	if !IsModule(moduleStruct) {
		return nil
	}

	moduleType := reflect.TypeOf(moduleStruct)
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	// Create a wrapper for the module
	return &ModuleWrapper{
		name:       moduleType.Name(),
		moduleType: moduleType,
		instance:   moduleStruct,
	}
}

// ModuleWrapper wraps a struct to implement the Module interface
type ModuleWrapper struct {
	name       string
	moduleType reflect.Type
	instance   interface{}
}

// GetModuleName returns the module name
func (mw *ModuleWrapper) GetModuleName() string {
	return mw.name
}

// GetControllers extracts controllers from the module struct
func (mw *ModuleWrapper) GetControllers() []interface{} {
	controllers := make([]interface{}, 0)

	moduleValue := reflect.ValueOf(mw.instance)
	if moduleValue.Kind() == reflect.Ptr {
		moduleValue = moduleValue.Elem()
	}

	for i := 0; i < mw.moduleType.NumField(); i++ {
		field := mw.moduleType.Field(i)
		fieldValue := moduleValue.Field(i)

		// Skip BaseModule field
		if field.Name == "BaseModule" {
			continue
		}

		// Check if field has controller tag
		if field.Tag.Get("controller") == "true" {
			if fieldValue.CanInterface() {
				controllers = append(controllers, fieldValue.Interface())
			}
		}
	}

	return controllers
}

// GetServices extracts services from the module struct
func (mw *ModuleWrapper) GetServices() []interface{} {
	services := make([]interface{}, 0)

	moduleValue := reflect.ValueOf(mw.instance)
	if moduleValue.Kind() == reflect.Ptr {
		moduleValue = moduleValue.Elem()
	}

	for i := 0; i < mw.moduleType.NumField(); i++ {
		field := mw.moduleType.Field(i)
		fieldValue := moduleValue.Field(i)

		// Skip BaseModule field
		if field.Name == "BaseModule" {
			continue
		}

		// Check if field has service tag
		if field.Tag.Get("service") == "true" {
			if fieldValue.CanInterface() {
				services = append(services, fieldValue.Interface())
			}
		}
	}

	return services
}

// GetImports returns imported modules (empty for now)
func (mw *ModuleWrapper) GetImports() []Module {
	return []Module{}
}
