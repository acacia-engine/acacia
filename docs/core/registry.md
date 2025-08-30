# Registry Module

The `registry` module in the Acacia Engine provides a centralized mechanism for registering and retrieving services within the application. This module is crucial for managing dependencies and enabling a clean, explicit approach to dependency injection, aligning with Go's idiomatic practices.

## Key Components

### `Registry` Interface

The `Registry` interface defines the contract for any service registry implementation.

```go
type Registry interface {
	RegisterService(name string, service interface{}, moduleName string) error
	GetService(name string) (interface{}, error)
	UnregisterService(name string)
}
```

*   `RegisterService(name string, service interface{}, moduleName string) error`: Registers a service with a given `name` and the service `interface{}` itself. The `moduleName` parameter helps in organizing and identifying services by their originating module. Returns an error if a service with the same name is already registered.
*   `GetService(name string) (interface{}, error)`: Retrieves a registered service by its `name`. Returns the service as an `interface{}` and an error if the service is not found.
*   `UnregisterService(name string)`: Unregisters a service by its `name`.

### `DefaultRegistry` Struct

The `DefaultRegistry` is the concrete implementation of the `Registry` interface, providing a basic in-memory service registration mechanism.

```go
type DefaultRegistry struct {
	// internal map to store services
	services map[string]serviceEntry
	// mutex to ensure concurrent access safety
	mu sync.RWMutex
}
```

*   `services`: An internal map that stores `serviceEntry` structs, mapping service names to their registered instances and module names.
*   `mu`: A `sync.RWMutex` to ensure thread-safe access to the `services` map during concurrent registration and retrieval operations.

### `NewDefaultRegistry()`

This function creates and returns a new instance of `DefaultRegistry`.

```go
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		services: make(map[string]serviceEntry),
	}
}
```

### `serviceEntry` Struct

An internal helper struct used by `DefaultRegistry` to store information about each registered service.

```go
type serviceEntry struct {
	service    interface{}
	moduleName string
}
```

*   `service`: The actual service instance.
*   `moduleName`: The name of the module that registered this service.
