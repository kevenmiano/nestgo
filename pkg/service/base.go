package service

import (
	"fmt"
	"reflect"
	"strings"
)

// BaseService provides base functionality for all services
type BaseService struct {
	_ string `service:"true"`
}

// Service interface defines the contract for all services
type Service interface {
	GetServiceName() string
	GetServiceDescription() string
	IsService() bool
}

// GetServiceName returns the name of the service using reflection
func (bs *BaseService) GetServiceName() string {
	structType := reflect.TypeOf(bs)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Name() == "BaseService" {
		return "BaseService"
	}

	return structType.Name()
}

// GetServiceDescription generates a description based on the service name
func (bs *BaseService) GetServiceDescription() string {
	name := bs.GetServiceName()

	if strings.HasSuffix(name, "Service") {
		baseName := strings.TrimSuffix(name, "Service")
		return fmt.Sprintf("%sService manages %s operations", name, strings.ToLower(baseName))
	}

	return fmt.Sprintf("%s manages system operations", name)
}

// IsService returns true indicating this is a service
func (bs *BaseService) IsService() bool {
	return true
}

// MetaExtractor extracts metadata from services using reflection
type MetaExtractor struct{}

// NewMetaExtractor creates a new MetaExtractor instance
func NewMetaExtractor() *MetaExtractor {
	return &MetaExtractor{}
}

// IsService checks if the given value is a service
func (me *MetaExtractor) IsService(v interface{}) bool {
	if service, ok := v.(Service); ok {
		return service.IsService()
	}

	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Kind() == reflect.Struct {
		_, found := structType.FieldByName("BaseService")
		return found
	}

	return false
}

// GetServiceName returns the name of the service using reflection
func (me *MetaExtractor) GetServiceName(v interface{}) string {
	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	return structType.Name()
}

// GetServiceDescription generates a description based on the service name
func (me *MetaExtractor) GetServiceDescription(v interface{}) string {
	name := me.GetServiceName(v)
	if strings.HasSuffix(name, "Service") {
		baseName := strings.TrimSuffix(name, "Service")
		return fmt.Sprintf("%sService manages %s operations", name, strings.ToLower(baseName))
	}

	return fmt.Sprintf("%s manages system operations", name)
}
