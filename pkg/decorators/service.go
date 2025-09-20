package decorators

import (
	"fmt"
	"reflect"
)

// Injectable decorator for marking classes as injectable services
func Injectable() func(interface{}) {
	return func(target interface{}) {
		targetType := reflect.TypeOf(target)
		if targetType.Kind() == reflect.Ptr {
			targetType = targetType.Elem()
		}
		fmt.Printf("🔧 Injectable service: %s\n", targetType.Name())
	}
}

// Service decorator for marking classes as services
func Service() func(interface{}) {
	return func(target interface{}) {
		targetType := reflect.TypeOf(target)
		if targetType.Kind() == reflect.Ptr {
			targetType = targetType.Elem()
		}
		fmt.Printf("🔧 Service: %s\n", targetType.Name())
	}
}

// ServiceMetadata stores service information
type ServiceMetadata struct {
	Name string
	Type reflect.Type
}

// ExtractServiceMetadata extracts metadata from a service
func ExtractServiceMetadata(service interface{}) *ServiceMetadata {
	serviceType := reflect.TypeOf(service)
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}

	return &ServiceMetadata{
		Name: serviceType.Name(),
		Type: serviceType,
	}
}
