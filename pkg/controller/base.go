package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/kevenmiano/nestgo/pkg/logger"
)

const (
	// Tag keys for struct fields
	TagRequired = "required"
	TagDesc     = "desc"
	TagValidate = "validate"
	TagJSON     = "json"
	TagBaseURL  = "baseUrl"
	TagHTTP     = "http"

	// Tag values
	TagValueTrue = "true"

	// Controller naming
	ControllerSuffix = "Controller"

	// Error messages
	ErrUnknownController = "UnknownController"
)

// BaseController provides base functionality for all controllers
type BaseController struct {
	_ string `controller:"true"`

	// HTTP context (will be injected by the framework)
	ResponseWriter http.ResponseWriter
	Request        *http.Request
}

// Controller interface defines the contract for all controllers
type Controller interface {
	GetControllerName() string
	GetControllerDescription() string
	IsController() bool
}

// RouteController interface defines methods that represent routes
type RouteController interface {
	Controller
	GetRoutes() map[string]string
}

// GetControllerName returns the name of the controller using reflection
func (bc *BaseController) GetControllerName() string {
	structType := reflect.TypeOf(bc)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Name() == "BaseController" {
		return "BaseController"
	}

	return structType.Name()
}

// GetControllerDescription generates a description based on the controller name
func (bc *BaseController) GetControllerDescription() string {
	name := bc.GetControllerName()

	if strings.HasSuffix(name, ControllerSuffix) {
		baseName := strings.TrimSuffix(name, ControllerSuffix)
		return fmt.Sprintf("%sController manages %s operations", name, strings.ToLower(baseName))
	}

	return fmt.Sprintf("%s manages system operations", name)
}

// IsController returns true indicating this is a controller
func (bc *BaseController) IsController() bool {
	return true
}

// JSON sends a JSON response
func (bc *BaseController) JSON(data interface{}) {
	if bc.ResponseWriter == nil {
		logger.Warn("ResponseWriter is nil in BaseController.JSON()")
		return
	}

	logger.Info("BaseController.JSON() called", "data", data)
	bc.ResponseWriter.Header().Set("Content-Type", "application/json")

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Error("Failed to marshal JSON", "error", err)
		http.Error(bc.ResponseWriter, `{"error": "Failed to serialize response"}`, http.StatusInternalServerError)
		return
	}

	logger.Info("Writing JSON response", "jsonData", string(jsonData))
	bc.ResponseWriter.Write(jsonData)
}

// JSONWithStatus sends a JSON response with custom status code
func (bc *BaseController) JSONWithStatus(statusCode int, data interface{}) {
	if bc.ResponseWriter == nil {
		return
	}

	bc.ResponseWriter.Header().Set("Content-Type", "application/json")
	bc.ResponseWriter.WriteHeader(statusCode)

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(bc.ResponseWriter, `{"error": "Failed to serialize response"}`, http.StatusInternalServerError)
		return
	}

	bc.ResponseWriter.Write(jsonData)
}

// SetHTTPContext sets the HTTP context for the controller
func (bc *BaseController) SetHTTPContext(w http.ResponseWriter, r *http.Request) {
	logger.Info("BaseController.SetHTTPContext called", "responseWriter", w != nil, "request", r != nil)
	bc.ResponseWriter = w
	bc.Request = r
	logger.Info("HTTP context set successfully", "responseWriter", bc.ResponseWriter != nil)
}

// MetaExtractor extracts metadata from structs using reflection
type MetaExtractor struct{}

// NewMetaExtractor creates a new MetaExtractor instance
func NewMetaExtractor() *MetaExtractor {
	return &MetaExtractor{}
}

// IsController checks if the given value is a controller
func (me *MetaExtractor) IsController(v interface{}) bool {
	if controller, ok := v.(Controller); ok {
		return controller.IsController()
	}

	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Kind() == reflect.Struct {
		_, found := structType.FieldByName("BaseController")
		return found
	}

	return false
}

// GetControllerName returns the name of the controller using reflection
func (me *MetaExtractor) GetControllerName(v interface{}) string {
	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	return structType.Name()
}

// GetControllerDescription generates a description based on the controller name
func (me *MetaExtractor) GetControllerDescription(v interface{}) string {
	name := me.GetControllerName(v)
	if strings.HasSuffix(name, ControllerSuffix) {
		baseName := strings.TrimSuffix(name, ControllerSuffix)
		return fmt.Sprintf("%sController manages %s operations", name, strings.ToLower(baseName))
	}

	return fmt.Sprintf("%s manages system operations", name)
}

// GetControllerBaseURL returns the base URL for the controller
func (me *MetaExtractor) GetControllerBaseURL(v interface{}) string {
	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Kind() == reflect.Struct {
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.Name == "BaseController" {
				return field.Tag.Get(TagBaseURL)
			}
		}
	}

	return ""
}

// ValidateControllerBaseURL checks if the controller has a valid baseUrl
func (me *MetaExtractor) ValidateControllerBaseURL(v interface{}) error {
	baseURL := me.GetControllerBaseURL(v)
	if baseURL == "" {
		return fmt.Errorf("baseUrl is required for controller %s", me.GetControllerName(v))
	}
	return nil
}

// GetControllerRoutes extracts routes from controller fields with HTTP tags
func (me *MetaExtractor) GetControllerRoutes(v interface{}) map[string]string {
	routes := make(map[string]string)
	baseURL := me.GetControllerBaseURL(v)

	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip BaseController field
		if field.Name == "BaseController" {
			continue
		}

		// Check if field has http tag
		httpMethod := field.Tag.Get(TagHTTP)
		if httpMethod != "" {
			route := fmt.Sprintf("%s %s", httpMethod, baseURL)
			routes[field.Name] = route
		}
	}

	return routes
}

// GetFieldTag retrieves a specific tag value from a struct field
func (me *MetaExtractor) GetFieldTag(v interface{}, fieldName, tagKey string) string {
	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Kind() != reflect.Struct {
		return ""
	}

	field, found := structType.FieldByName(fieldName)
	if !found {
		return ""
	}

	return field.Tag.Get(tagKey)
}

// HasFieldTag checks if a struct field has a specific tag
func (me *MetaExtractor) HasFieldTag(v interface{}, fieldName, tagKey string) bool {
	return me.GetFieldTag(v, fieldName, tagKey) != ""
}

// GetRequiredFields returns a list of field names marked as required
func (me *MetaExtractor) GetRequiredFields(v interface{}) []string {
	var required []string

	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.Name == "BaseController" {
			continue
		}
		if field.Tag.Get(TagRequired) == TagValueTrue {
			required = append(required, field.Name)
		}
	}

	return required
}

// PrintStructInfo prints detailed information about a struct including metadata
func (me *MetaExtractor) PrintStructInfo(v interface{}) {
	structType := reflect.TypeOf(v)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	structName := structType.Name()
	fmt.Printf("=== Struct: %s ===\n", structName)

	if me.IsController(v) {
		fmt.Println("🎮 CONTROLLER")
		if desc := me.GetControllerDescription(v); desc != "" {
			fmt.Printf("Description: %s\n", desc)
		}
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		if field.Name == "BaseController" {
			continue
		}

		fmt.Printf("  %s (%s)", field.Name, field.Type)

		if desc := field.Tag.Get(TagDesc); desc != "" {
			fmt.Printf(" - %s", desc)
		}

		if required := field.Tag.Get(TagRequired); required == TagValueTrue {
			fmt.Printf(" [REQUIRED]")
		}

		if validate := field.Tag.Get(TagValidate); validate != "" {
			fmt.Printf(" [validation: %s]", validate)
		}

		fmt.Println()
	}
	fmt.Println()
}
