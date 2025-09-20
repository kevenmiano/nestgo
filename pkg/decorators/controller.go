package decorators

import (
	"fmt"
	"reflect"
	"strings"
)

// Controller decorator for marking classes as controllers
func Controller(basePath string) func(interface{}) {
	return func(target interface{}) {
		// This would be used for metadata in a real implementation
		fmt.Printf("🎮 Controller registered with base path: %s\n", basePath)
	}
}

// Get decorator for GET routes
func Get(path string) func(interface{}) {
	return func(target interface{}) {
		fmt.Printf("  GET %s\n", path)
	}
}

// Post decorator for POST routes
func Post(path string) func(interface{}) {
	return func(target interface{}) {
		fmt.Printf("  POST %s\n", path)
	}
}

// Put decorator for PUT routes
func Put(path string) func(interface{}) {
	return func(target interface{}) {
		fmt.Printf("  PUT %s\n", path)
	}
}

// Delete decorator for DELETE routes
func Delete(path string) func(interface{}) {
	return func(target interface{}) {
		fmt.Printf("  DELETE %s\n", path)
	}
}

// Patch decorator for PATCH routes
func Patch(path string) func(interface{}) {
	return func(target interface{}) {
		fmt.Printf("  PATCH %s\n", path)
	}
}

// ControllerMetadata stores controller information
type ControllerMetadata struct {
	Name     string
	BasePath string
	Routes   []RouteMetadata
}

// RouteMetadata stores route information
type RouteMetadata struct {
	Method  string
	Path    string
	Handler string
}

// ExtractControllerMetadata extracts metadata from a controller struct
func ExtractControllerMetadata(controller interface{}) *ControllerMetadata {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	metadata := &ControllerMetadata{
		Name:   controllerType.Name(),
		Routes: make([]RouteMetadata, 0),
	}

	// Extract base path from BaseController field
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)

		if field.Name == "BaseController" {
			basePath := field.Tag.Get("baseUrl")
			if basePath != "" {
				metadata.BasePath = basePath
			}
			continue
		}

		// Extract route information from fields
		httpMethod := field.Tag.Get("http")
		if httpMethod != "" {
			route := RouteMetadata{
				Method:  strings.ToUpper(httpMethod),
				Path:    metadata.BasePath,
				Handler: field.Name,
			}
			metadata.Routes = append(metadata.Routes, route)
		}
	}

	return metadata
}
