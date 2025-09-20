package decorators

import (
	"fmt"
	"reflect"
	"strings"
)

// HTTPMethod represents an HTTP method
type HTTPMethod string

const (
	GET     HTTPMethod = "GET"
	POST    HTTPMethod = "POST"
	PUT     HTTPMethod = "PUT"
	DELETE  HTTPMethod = "DELETE"
	PATCH   HTTPMethod = "PATCH"
	HEAD    HTTPMethod = "HEAD"
	OPTIONS HTTPMethod = "OPTIONS"
)

// RouteConfig represents the configuration for a route
type RouteConfig struct {
	Method HTTPMethod
	Path   string
}

// Route decorator function that marks a method as a route
func Route(method HTTPMethod, path string) func(interface{}) interface{} {
	return func(methodFunc interface{}) interface{} {
		// Store route configuration in method metadata
		// This is a placeholder - in a real implementation, you'd store this metadata
		// that can be retrieved during route discovery
		return methodFunc
	}
}

// GetRoute decorator for GET routes
func GetRoute(path string) func(interface{}) interface{} {
	return Route(GET, path)
}

// PostRoute decorator for POST routes
func PostRoute(path string) func(interface{}) interface{} {
	return Route(POST, path)
}

// PutRoute decorator for PUT routes
func PutRoute(path string) func(interface{}) interface{} {
	return Route(PUT, path)
}

// DeleteRoute decorator for DELETE routes
func DeleteRoute(path string) func(interface{}) interface{} {
	return Route(DELETE, path)
}

// PatchRoute decorator for PATCH routes
func PatchRoute(path string) func(interface{}) interface{} {
	return Route(PATCH, path)
}

// HeadRoute decorator for HEAD routes
func HeadRoute(path string) func(interface{}) interface{} {
	return Route(HEAD, path)
}

// OptionsRoute decorator for OPTIONS routes
func OptionsRoute(path string) func(interface{}) interface{} {
	return Route(OPTIONS, path)
}

// RouteInfo stores route information for methods
type RouteInfo struct {
	Method HTTPMethod
	Path   string
}

// RouteRegistry stores route metadata for methods
var routeRegistry = make(map[string]RouteInfo)

// RegisterRoute registers a route for a method
func RegisterRoute(methodName string, config RouteConfig) {
	routeRegistry[methodName] = RouteInfo{
		Method: config.Method,
		Path:   config.Path,
	}
}

// GetRouteMetadata retrieves route metadata for a method
func GetRouteMetadata(methodName string) (RouteInfo, bool) {
	metadata, exists := routeRegistry[methodName]
	return metadata, exists
}

// GetAllRoutes returns all registered routes
func GetAllRoutes() map[string]RouteInfo {
	return routeRegistry
}

// RouteExtractor extracts route information from method names and struct tags
type RouteExtractor struct{}

// NewRouteExtractor creates a new route extractor
func NewRouteExtractor() *RouteExtractor {
	return &RouteExtractor{}
}

// ExtractRouteFromMethodName extracts route information from method name convention
func (re *RouteExtractor) ExtractRouteFromMethodName(methodName string) (HTTPMethod, string, bool) {
	// Method name conventions:
	// GetUsers, GetAllUsers -> GET /
	// GetUser, GetUserByID -> GET /:id
	// CreateUser, PostUser -> POST /
	// UpdateUser, PutUser -> PUT /:id
	// DeleteUser, RemoveUser -> DELETE /:id
	// PatchUser -> PATCH /:id
	// HeadUsers -> HEAD /
	// OptionsUsers -> OPTIONS /

	methodName = strings.TrimSpace(methodName)

	// GET methods
	if strings.HasPrefix(methodName, "Get") {
		if strings.Contains(methodName, "All") || strings.HasSuffix(methodName, "s") {
			return GET, "/", true
		}
		return GET, "/:id", true
	}

	// POST methods
	if strings.HasPrefix(methodName, "Create") || strings.HasPrefix(methodName, "Post") {
		return POST, "/", true
	}

	// PUT methods
	if strings.HasPrefix(methodName, "Update") || strings.HasPrefix(methodName, "Put") {
		return PUT, "/:id", true
	}

	// DELETE methods
	if strings.HasPrefix(methodName, "Delete") || strings.HasPrefix(methodName, "Remove") {
		return DELETE, "/:id", true
	}

	// PATCH methods
	if strings.HasPrefix(methodName, "Patch") {
		return PATCH, "/:id", true
	}

	// HEAD methods
	if strings.HasPrefix(methodName, "Head") {
		return HEAD, "/", true
	}

	// OPTIONS methods
	if strings.HasPrefix(methodName, "Options") {
		return OPTIONS, "/", true
	}

	return "", "", false
}

// IsValidHTTPMethod checks if a string is a valid HTTP method
func IsValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	method = strings.ToUpper(method)
	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}
	return false
}

// ParseHTTPMethod parses a string to HTTPMethod
func ParseHTTPMethod(method string) (HTTPMethod, error) {
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		return GET, nil
	case "POST":
		return POST, nil
	case "PUT":
		return PUT, nil
	case "DELETE":
		return DELETE, nil
	case "PATCH":
		return PATCH, nil
	case "HEAD":
		return HEAD, nil
	case "OPTIONS":
		return OPTIONS, nil
	default:
		return "", fmt.Errorf("invalid HTTP method: %s", method)
	}
}

// GetMethodFromReflection extracts HTTP method from method reflection
func GetMethodFromReflection(method reflect.Method) (HTTPMethod, string, bool) {
	extractor := NewRouteExtractor()

	// First try to extract from method name convention
	if method, path, ok := extractor.ExtractRouteFromMethodName(method.Name); ok {
		return method, path, true
	}

	// If no convention match, return false
	return "", "", false
}
