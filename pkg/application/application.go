package application

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/kevenmiano/nestgo/pkg/app"
	"github.com/kevenmiano/nestgo/pkg/logger"
	"github.com/kevenmiano/nestgo/pkg/module"
)

// TreeNode represents a node in the dependency tree
type TreeNode struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Children []*TreeNode            `json:"children,omitempty"`
	Module   module.Module          `json:"-"`
}

// Application represents the main application that auto-discovers modules
type Application struct {
	app  *app.App
	tree *TreeNode
}

// NewApplication creates a new application instance
func NewApplication() *Application {
	return &Application{
		app: app.NewApp(),
		tree: &TreeNode{
			Name:     "Application",
			Type:     "root",
			Children: make([]*TreeNode, 0),
		},
	}
}

// Start starts the application with auto-discovery
func (a *Application) Start(port string) {
	logger.Info("🔍 Auto-discovering modules...")

	// Build dependency tree from registered modules
	a.buildDependencyTree()

	// Print the tree structure
	a.printTree()

	// Start the application
	a.app.Start(port)
}

// buildDependencyTree builds the dependency tree from registered modules
func (a *Application) buildDependencyTree() {
	registry := module.GetGlobalRegistry()
	modules := registry.GetAllModules()

	logger.Info("Building dependency tree", "moduleCount", len(modules))

	for moduleName, moduleInstance := range modules {
		moduleNode := a.addModuleNode(moduleName, moduleInstance)
		a.buildModuleTree(moduleNode, moduleInstance)
	}
}

// addModuleNode adds a module node to the tree
func (a *Application) addModuleNode(moduleName string, moduleInstance module.Module) *TreeNode {
	moduleNode := &TreeNode{
		Name:     moduleName,
		Type:     "module",
		Data:     make(map[string]interface{}),
		Children: make([]*TreeNode, 0),
		Module:   moduleInstance,
	}

	// Add module data
	controllers := moduleInstance.GetControllers()
	services := moduleInstance.GetServices()
	imports := moduleInstance.GetImports()

	moduleNode.Data["controllers"] = len(controllers)
	moduleNode.Data["services"] = len(services)
	moduleNode.Data["imports"] = len(imports)

	a.tree.Children = append(a.tree.Children, moduleNode)
	return moduleNode
}

// buildModuleTree builds the tree structure for a specific module
func (a *Application) buildModuleTree(moduleNode *TreeNode, moduleInstance module.Module) {
	controllers := moduleInstance.GetControllers()

	for _, controller := range controllers {
		controllerNode := a.addControllerNode(moduleNode, controller)
		a.buildControllerTree(controllerNode, controller)
	}
}

// addControllerNode adds a controller node to the module tree
func (a *Application) addControllerNode(moduleNode *TreeNode, controller interface{}) *TreeNode {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	controllerNode := &TreeNode{
		Name:     controllerType.Name(),
		Type:     "controller",
		Data:     make(map[string]interface{}),
		Children: make([]*TreeNode, 0),
	}

	// Extract base URL from controller
	baseURL := a.extractBaseURL(controllerType)
	controllerNode.Data["baseUrl"] = baseURL

	moduleNode.Children = append(moduleNode.Children, controllerNode)
	return controllerNode
}

// buildControllerTree builds the tree structure for a specific controller
func (a *Application) buildControllerTree(controllerNode *TreeNode, controller interface{}) {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	// Get the pointer type to access methods
	ptrType := reflect.PtrTo(controllerType)

	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if method.IsExported() {
			// Check if method can be mapped to a route
			if strings.HasPrefix(method.Name, "Get") || strings.HasPrefix(method.Name, "Create") {
				routeNode := a.addRouteNode(controllerNode, method.Name, controller)
				a.buildRouteTree(routeNode, method.Name, controller)
			}
		}
	}
}

// addRouteNode adds a route node to the controller tree
func (a *Application) addRouteNode(controllerNode *TreeNode, methodName string, controller interface{}) *TreeNode {
	routeNode := &TreeNode{
		Name:     methodName,
		Type:     "route",
		Data:     make(map[string]interface{}),
		Children: make([]*TreeNode, 0),
	}

	// Extract route information
	httpMethod, path := a.extractRouteInfo(methodName, controller)
	routeNode.Data["httpMethod"] = httpMethod
	routeNode.Data["path"] = path

	controllerNode.Children = append(controllerNode.Children, routeNode)
	return routeNode
}

// buildRouteTree builds the tree structure for a specific route
func (a *Application) buildRouteTree(routeNode *TreeNode, methodName string, controller interface{}) {
	// Add route-specific data
	routeNode.Data["handler"] = methodName
	routeNode.Data["status"] = "registered"
}

// extractBaseURL extracts the base URL from controller struct tags
func (a *Application) extractBaseURL(controllerType reflect.Type) string {
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)
		if field.Type.Name() == "BaseController" {
			if baseURL := field.Tag.Get("baseUrl"); baseURL != "" {
				return baseURL
			}
		}
	}
	return ""
}

// extractRouteInfo extracts HTTP method and path from method name
func (a *Application) extractRouteInfo(methodName string, controller interface{}) (string, string) {
	// Parse method name to determine HTTP method
	var httpMethod string
	if strings.HasPrefix(methodName, "Get") {
		httpMethod = "GET"
	} else if strings.HasPrefix(methodName, "Create") {
		httpMethod = "POST"
	} else if strings.HasPrefix(methodName, "Update") {
		httpMethod = "PUT"
	} else if strings.HasPrefix(methodName, "Delete") {
		httpMethod = "DELETE"
	}

	// Get base URL from controller
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}
	baseURL := a.extractBaseURL(controllerType)

	// Construct full path
	path := baseURL + "/"
	return httpMethod, path
}

// printTree prints the dependency tree structure
func (a *Application) printTree() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🌳 APPLICATION DEPENDENCY TREE")
	fmt.Println(strings.Repeat("=", 80))
	a.printNode(a.tree, 0)
	fmt.Println(strings.Repeat("=", 80))
}

// printNode recursively prints a tree node
func (a *Application) printNode(node *TreeNode, depth int) {
	indent := strings.Repeat("  ", depth)
	icon := a.getIcon(node.Type)

	fmt.Printf("%s%s %s", indent, icon, node.Name)

	if len(node.Data) > 0 {
		fmt.Printf(" %v", node.Data)
	}

	fmt.Println()

	for _, child := range node.Children {
		a.printNode(child, depth+1)
	}
}

// getIcon returns the appropriate icon for a node type
func (a *Application) getIcon(nodeType string) string {
	switch nodeType {
	case "root":
		return "🏠"
	case "module":
		return "📦"
	case "controller":
		return "🎮"
	case "route":
		return "🛣️"
	default:
		return "📄"
	}
}

// GetTree returns the dependency tree
func (a *Application) GetTree() *TreeNode {
	return a.tree
}

// FindModuleNode finds a module node by name
func (a *Application) FindModuleNode(moduleName string) *TreeNode {
	for _, child := range a.tree.Children {
		if child.Type == "module" && child.Name == moduleName {
			return child
		}
	}
	return nil
}

// FindControllerNode finds a controller node by name within a module
func (a *Application) FindControllerNode(moduleName, controllerName string) *TreeNode {
	moduleNode := a.FindModuleNode(moduleName)
	if moduleNode == nil {
		return nil
	}

	for _, child := range moduleNode.Children {
		if child.Type == "controller" && child.Name == controllerName {
			return child
		}
	}
	return nil
}

// FindRouteNode finds a route node by name within a controller
func (a *Application) FindRouteNode(moduleName, controllerName, routeName string) *TreeNode {
	controllerNode := a.FindControllerNode(moduleName, controllerName)
	if controllerNode == nil {
		return nil
	}

	for _, child := range controllerNode.Children {
		if child.Type == "route" && child.Name == routeName {
			return child
		}
	}
	return nil
}

// AddModuleNode adds a new module node to the tree
func (a *Application) AddModuleNode(moduleName string, moduleInstance module.Module) *TreeNode {
	return a.addModuleNode(moduleName, moduleInstance)
}

// RemoveModuleNode removes a module node from the tree
func (a *Application) RemoveModuleNode(moduleName string) bool {
	for i, child := range a.tree.Children {
		if child.Type == "module" && child.Name == moduleName {
			a.tree.Children = append(a.tree.Children[:i], a.tree.Children[i+1:]...)
			return true
		}
	}
	return false
}

// UpdateModuleNode updates a module node's data
func (a *Application) UpdateModuleNode(moduleName string, data map[string]interface{}) bool {
	moduleNode := a.FindModuleNode(moduleName)
	if moduleNode == nil {
		return false
	}

	for key, value := range data {
		moduleNode.Data[key] = value
	}
	return true
}

// GetModuleDependencies returns all dependencies for a module
func (a *Application) GetModuleDependencies(moduleName string) []string {
	moduleNode := a.FindModuleNode(moduleName)
	if moduleNode == nil {
		return nil
	}

	dependencies := make([]string, 0)

	// Add controllers as dependencies
	for _, child := range moduleNode.Children {
		if child.Type == "controller" {
			dependencies = append(dependencies, child.Name)
		}
	}

	return dependencies
}

// GetRouteCount returns the total number of routes in the application
func (a *Application) GetRouteCount() int {
	count := 0
	a.countRoutes(a.tree, &count)
	return count
}

// countRoutes recursively counts routes in the tree
func (a *Application) countRoutes(node *TreeNode, count *int) {
	if node.Type == "route" {
		*count++
	}
	for _, child := range node.Children {
		a.countRoutes(child, count)
	}
}

// autoDiscoverModules automatically discovers modules in the current package
func (a *Application) autoDiscoverModules() []interface{} {
	modules := make([]interface{}, 0)

	// Get the caller's package
	_, _, _, ok := runtime.Caller(2)
	if !ok {
		return modules
	}

	// This is a simplified approach - in a real implementation you'd scan the package
	// For now, we'll use a registry approach where modules register themselves

	return modules
}

// RegisterModule registers a module for auto-discovery
func (a *Application) RegisterModule(moduleStruct interface{}) {
	// Auto-register the module
	module.AutoRegisterModule(module.ExtractModuleFromStruct(moduleStruct))
}

// GetApp returns the underlying app instance
func (a *Application) GetApp() *app.App {
	return a.app
}
