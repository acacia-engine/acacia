package registry

import (
	"context"
	"fmt"
	"sync"

	"acacia/core/auth" // Import the auth package
)

// Registry defines the interface for a centralized service discovery mechanism.
type Registry interface {
	RegisterService(name string, service interface{}, moduleName string) error
	// GetService retrieves a registered service by its name, performing an access check.
	// The context must contain a Principal for the access check to be effective.
	GetService(ctx context.Context, name string) (interface{}, error)
	UnregisterService(name string)
	UnregisterServicesByModule(moduleName string)
	GetGateway(ctx context.Context, name string) (interface{}, error)
	RegisterGateway(name string, gateway interface{}) error
	UnregisterGateway(name string)
}

// DefaultRegistry is a concrete implementation of the Registry interface.
type DefaultRegistry struct {
	services         map[string]serviceEntry
	mu               sync.RWMutex
	accessController auth.AccessController   // New field
	gateways         map[string]serviceEntry // Add a map for gateways
}

type serviceEntry struct {
	service    interface{}
	moduleName string
}

// NewDefaultRegistry creates a new instance of DefaultRegistry.
// It now requires an auth.AccessController for permission checks.
func NewDefaultRegistry(ac auth.AccessController) *DefaultRegistry {
	if ac == nil {
		// Fallback to a default "allow all" controller if none is provided,
		// though in a production kernel, a proper controller should always be used.
		ac = auth.NewDefaultAccessController(nil)
	}
	return &DefaultRegistry{
		services:         make(map[string]serviceEntry),
		gateways:         make(map[string]serviceEntry), // Initialize gateways map
		accessController: ac,
	}
}

// RegisterService registers a service with a given name and the module it belongs to.
func (r *DefaultRegistry) RegisterService(name string, service interface{}, moduleName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service with name '%s' already registered", name)
	}

	// Check if the service is a kernel.Gateway. If so, register it in the gateways map.
	// This is a simplified check; a more robust solution might involve a dedicated RegisterGateway method
	// or a type assertion against a kernel.Gateway interface. For now, we'll assume
	// that if a service is intended to be a gateway, it will be retrieved via GetGateway.
	r.services[name] = serviceEntry{
		service:    service,
		moduleName: moduleName,
	}
	return nil
}

// RegisterGateway registers a gateway with a given name.
func (r *DefaultRegistry) RegisterGateway(name string, gateway interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.gateways[name]; exists {
		return fmt.Errorf("gateway with name '%s' already registered", name)
	}

	r.gateways[name] = serviceEntry{
		service:    gateway,
		moduleName: name, // For gateways, the moduleName is typically the gateway name itself
	}
	return nil
}

// UnregisterGateway unregisters a gateway by its name.
func (r *DefaultRegistry) UnregisterGateway(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.gateways, name)
}

// GetService retrieves a registered service by its name, performing an access check.
// The context must contain a Principal for the access check to be effective.
func (r *DefaultRegistry) GetService(ctx context.Context, name string) (interface{}, error) {
	r.mu.RLock()
	entry, exists := r.services[name]
	r.mu.RUnlock() // Release read lock before potentially calling access controller

	if !exists {
		return nil, fmt.Errorf("service with name '%s' not found", name)
	}

	// Extract principal from context
	p := auth.PrincipalFromContext(ctx)
	if p == nil {
		// If no principal is in context, deny access for security.
		return nil, fmt.Errorf("no principal found in context for service access check for '%s'", name)
	}

	// Define the permission required to access this service
	// Convention: "service.<module_name>.<service_name>.access"
	permission := auth.Permission(fmt.Sprintf("service.%s.%s.access", entry.moduleName, name))

	// Perform the access check
	if !r.accessController.HasPermission(p, permission) {
		return nil, fmt.Errorf("principal %s (type: %s) is not authorized to access service '%s' (missing permission: %s)", p.ID(), p.Type(), name, permission)
	}

	return entry.service, nil
}

// UnregisterService unregisters a service by its name.
func (r *DefaultRegistry) UnregisterService(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, name)
}

// UnregisterServicesByModule unregisters all services associated with a given module name.
func (r *DefaultRegistry) UnregisterServicesByModule(moduleName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, entry := range r.services {
		if entry.moduleName == moduleName {
			delete(r.services, name)
		}
	}
	for name, entry := range r.gateways {
		if entry.moduleName == moduleName {
			r.UnregisterGateway(name) // Use the new UnregisterGateway method
		}
	}
}

// GetGateway retrieves a registered gateway by its name, performing an access check.
func (r *DefaultRegistry) GetGateway(ctx context.Context, name string) (interface{}, error) {
	r.mu.RLock()
	entry, exists := r.gateways[name] // Retrieve from gateways map
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("gateway with name '%s' not found", name)
	}

	p := auth.PrincipalFromContext(ctx)
	if p == nil {
		return nil, fmt.Errorf("no principal found in context for gateway access check for '%s'", name)
	}

	// Convention: "gateway.<gateway_name>.access"
	permission := auth.Permission(fmt.Sprintf("gateway.%s.access", name))

	if !r.accessController.HasPermission(p, permission) {
		return nil, fmt.Errorf("principal %s (type: %s) is not authorized to access gateway '%s' (missing permission: %s)", p.ID(), p.Type(), name, permission)
	}

	return entry.service, nil
}
