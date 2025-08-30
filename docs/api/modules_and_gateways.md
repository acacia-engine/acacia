# Acacia API for Module Designers

This document provides a comprehensive guide for designing and implementing modules and gateways within the Acacia framework. It focuses on the interfaces and lifecycle management provided by the Kernel, allowing module designers to build robust and integrated components without needing deep knowledge of the Kernel's internal workings.

## 1. Introduction to Acacia Modules and Gateways

Acacia is built around a modular architecture, where core functionalities are encapsulated within "Modules" and external interactions are handled by "Gateways." The "Kernel" acts as the central orchestrator, managing the lifecycle and intercommunication of these components.

### What are Modules?
Modules represent self-contained features or core business logic within the Acacia application. They are designed to be independent, allowing for clear separation of concerns and easier development, testing, and deployment. Examples include user management, data processing, or specific application features.

### What are Gateways?
Gateways are responsible for handling external interactions, such as network interfaces, protocol handlers, or integrations with third-party services. They act as the bridge between the internal modules and the outside world, ensuring that modules can focus purely on business logic. Examples include HTTP API servers, message queue consumers, or database connectors.

### The Role of the Kernel
The Kernel is the heart of the Acacia application. It is responsible for:
*   **Lifecycle Management:** Loading, configuring, starting, stopping, and reloading modules and gateways.
*   **Dependency Resolution:** Ensuring modules are started in the correct order based on their declared dependencies.
*   **Configuration Distribution:** Providing specific configurations to modules and gateways.
*   **Service Discovery:** Facilitating communication between modules via a central registry.

## 2. Module Interface (`kernel.Module`)

The `kernel.Module` interface defines the contract that all Acacia modules must adhere to. By implementing this interface, your module can be seamlessly integrated and managed by the Kernel.

```go
type Module interface {
	Name() string
	Version() string
	Dependencies() map[string]string
	OnLoad(ctx context.Context) error
	Configure(cfg interface{}) error
	Start(ctx context.Context) error
	OnReady(ctx context.Context) error
	RegisterServices(reg registry.Registry) error
	Stop(ctx context.Context) error
	OnConfigChanged(ctx context.Context, newCfg interface{}) error
	ShutdownTimeout() time.Duration
}
```

### Key Methods:

*   **`Name() string`**:
    *   Returns the unique name of the module. This name is used by the Kernel for identification, dependency resolution, and configuration lookup.
*   **`Version() string`**:
    *   Returns the semantic version (e.g., "1.0.0") of the module. Used for dependency version checking.
*   **`Dependencies() map[string]string`**:
    *   Returns a map where keys are module names and values are semantic version constraints (e.g., `{"auth": "^1.0.0"}`). The Kernel uses this to determine the correct startup order and ensure compatibility.
*   **`OnLoad(ctx context.Context) error`**:
    *   Called once when the module is first loaded by the Kernel. This is suitable for initial setup that does not require other modules to be started or configured.
*   **`Configure(cfg interface{}) error`**:
    *   Provides the module with its specific configuration. This method is called before `Start`. The `cfg` parameter will be a decoded configuration structure specific to your module.
*   **`Start(ctx context.Context) error`**:
    *   Initializes and starts the module's core functionality. This method should block until the module is fully ready to accept work.
*   **`OnReady(ctx context.Context) error`**:
    *   Called after the module and all its declared dependencies have successfully started. This is the ideal place for logic that relies on other modules being fully operational.
*   **`RegisterServices(reg registry.Registry) error`**:
    *   Called after the module has successfully started, allowing it to register its services with the Kernel's central `registry.Registry`. This enables other modules to discover and interact with your module's exposed functionalities.
*   **`Stop(ctx context.Context) error`**:
    *   Gracefully shuts down the module. Implementations should honor the provided `context.Context` for cancellation signals.
*   **`OnConfigChanged(ctx context.Context, newCfg interface{}) error`**:
    *   Called when the application's configuration is reloaded. Modules can use this to dynamically update their internal state based on new configuration values without requiring a full restart.
*   **`ShutdownTimeout() time.Duration`**:
    *   Returns the maximum duration the Kernel should wait for the module to stop gracefully during shutdown.

### Module Lifecycle Flow:

The Kernel manages modules through a defined lifecycle:

1.  **Loading & `OnLoad`**: Module binary is loaded, `NewModule()` is called, and `OnLoad()` is invoked for initial setup.
2.  **Configuration & `Configure`**: Module receives its specific configuration via `Configure()`.
3.  **Startup & `Start`**: Modules are started in dependency order. `Start()` is called and blocks until the module is ready.
4.  **Service Registration & `RegisterServices`**: After starting, modules register their services with the registry.
5.  **Readiness & `OnReady`**: Once all dependencies are started, `OnReady()` is called.
6.  **Runtime**: The module operates normally.
7.  **Config Change & `OnConfigChanged`**: If configuration changes, `OnConfigChanged()` is called.
8.  **Shutdown & `Stop`**: During application shutdown, `Stop()` is called for graceful termination (in reverse dependency order).

### Error Handling:
All module methods that return an `error` should return meaningful errors to the Kernel. The Kernel will log these errors and, in some cases (e.g., during `Start`), may halt the application startup or trigger rollbacks.

## 3. Gateway Interface (`kernel.Gateway`)

The `kernel.Gateway` interface defines the contract for components that handle external communication.

```go
type Gateway interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Configure(cfg interface{}) error
	ShutdownTimeout() time.Duration
}
```

### Key Methods:

*   **`Name() string`**:
    *   Returns the unique name of the gateway.
*   **`Start(ctx context.Context) error`**:
    *   Initializes and starts the gateway, typically binding to network ports or connecting to external systems. It should block until the gateway is ready to accept traffic.
*   **`Stop(ctx context.Context) error`**:
    *   Gracefully shuts down the gateway, typically unbinding from ports or disconnecting from external systems.
*   **`Configure(cfg interface{}) error`**:
    *   Provides the gateway with its specific configuration. This method is called before `Start` and also when configuration changes.
*   **`ShutdownTimeout() time.Duration`**:
    *   Returns the maximum duration the Kernel should wait for the gateway to stop gracefully.

### Gateway Lifecycle Flow:

Gateways have a simpler lifecycle compared to modules, but their startup and shutdown are coordinated with modules:

1.  **Configuration & `Configure`**: Gateway receives its specific configuration.
2.  **Startup & `Start`**: Gateways are started *after* all modules have started. This ensures that application services are ready before external traffic is accepted.
3.  **Runtime**: The gateway operates normally.
4.  **Config Change & `Configure`**: If configuration changes, `Configure()` is called again.
5.  **Shutdown & `Stop`**: During application shutdown, `Stop()` is called for graceful termination *before* modules are stopped. This ensures traffic ceases before modules shut down.

## 4. Configuration Management

Acacia uses a centralized configuration system. Modules and Gateways receive their specific configuration via the `Configure(cfg interface{})` method. The `cfg` parameter is typically a generic `interface{}`, which you should decode into a module-specific struct using a library like `mapstructure`.

Example (`modules/noop/noop.go`):
```go
type NoopConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Message string `mapstructure:"message"`
}

func (m *NoopModule) Configure(cfg interface{}) error {
	if cfg == nil {
		return nil // No configuration provided, use defaults.
	}
	var noopCfg NoopConfig
	if err := mapstructure.Decode(cfg, &noopCfg); err != nil {
		return fmt.Errorf("failed to decode NoopModule config: %w", err)
	}
	m.config = noopCfg
	return nil
}
```

The `OnConfigChanged(ctx context.Context, newCfg interface{}) error` method allows modules to react to runtime configuration updates. Gateways handle config changes via their `Configure` method.

## 5. Service Registry (`registry.Registry`)

The `registry.Registry` provides a mechanism for inter-module communication. Modules can register services they provide and discover services provided by other modules.

*   **Registering Services**: In your module's `RegisterServices` method, you can add your module's public interfaces or functions to the registry.
*   **Discovering Services**: Other modules can then retrieve these registered services from the registry to interact with them.

This promotes loose coupling between modules, as they interact through well-defined interfaces rather than direct imports.

## 6. Dependency Management

Modules declare their dependencies using the `Dependencies() map[string]string` method. The Kernel uses this information to:

*   **Topological Sort**: Determine the correct order in which modules must be started to satisfy all dependencies.
*   **Version Compatibility**: Enforce semantic versioning constraints. If a dependency's version does not meet the declared constraint, the Kernel will prevent startup.

Example:
```go
func (m *MyModule) Dependencies() map[string]string {
	return map[string]string{
		"auth":    "^1.2.0", // Depends on auth module, version 1.2.x or higher (but less than 2.0.0)
		"storage": "~0.5.0", // Depends on storage module, version 0.5.x
	}
}
```

## 7. Best Practices for Module/Gateway Development

*   **Concurrency Safety**: All methods of `Module` and `Gateway` interfaces must be concurrency-safe, as the Kernel may call them from different goroutines. Use mutexes (`sync.Mutex`, `sync.RWMutex`) to protect shared state.
*   **Context Handling**: Always honor the `context.Context` passed to lifecycle methods. Use `ctx.Done()` to listen for cancellation signals and `context.WithTimeout` for operations that should not exceed a certain duration.
*   **Logging and Metrics**: Utilize the `acacia/core/logger` and `acacia/core/metrics` packages for consistent logging and performance monitoring. The Kernel automatically integrates with these.
*   **Avoid Direct Gateway Dependencies in Modules**: Modules should expose application services that Gateways can call. Modules should not directly depend on Gateways; dependency wiring happens at the application's main entry point.
*   **Graceful Shutdown**: Implement `Stop()` methods to release resources cleanly (e.g., close database connections, stop goroutines, unbind network listeners).

## 8. Example: The Noop Module

The `modules/noop/noop.go` file provides a minimal example of a module implementation. It demonstrates:

*   Implementing the `kernel.Module` interface.
*   Defining a module-specific configuration struct (`NoopConfig`).
*   Handling configuration via `Configure()`.
*   Basic `Start()` and `Stop()` implementations.
*   Returning an empty dependency map.
*   Using `fmt.Printf` for simple output during lifecycle events (in a real module, you would use the `acacia/core/logger`).

This module serves as a good starting point for understanding the basic structure and interaction points of an Acacia module.
