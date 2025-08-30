# Kernel Documentation

## 1. Introduction to the Kernel
The Kernel is the central coordinator for managing the lifecycle of modules and gateways within the Acacia application. It acts as a central coordinator for starting, stopping, reloading, and managing the state of various components.

## 2. Key Concepts

### 2.1. Module Interface (`kernel.Module`)
A `Module` represents a self-contained feature that can be managed by the Kernel. Implementations must be concurrency-safe with respect to their lifecycle methods. The `Name` of a module must be unique across all modules.

**Methods:**
*   `Name() string`: Returns the unique name of the module.
*   `Version() string`: Returns the semantic version of the module (e.g., "1.0.0"). This is used by the kernel for dependency resolution and compatibility checks.
*   `SetRegistry(reg registry.Registry)`: Provides the module with the kernel's service registry. This method should be called once after the module is loaded.
*   `Dependencies() map[string]string`: Returns a map where keys are module names and values are semantic version constraints (e.g., "module-a": "^1.0.0", "module-b": ">=2.1.0 <3.0.0"). The kernel uses this to determine the correct startup and shutdown order and to ensure version compatibility.
*   `SetEventBus(bus events.Bus)`: Provides the module with the kernel's event bus. This method is called once after the module is loaded.
*   `OnLoad(ctx context.Context) error`: Called once when the module is first loaded by the kernel. This is suitable for initial setup that does not require other modules to be started.
*   `Configure(cfg interface{}) error`: Provides the module with its specific configuration. This method is called after `OnLoad` and before `Start`. It is also called when the application's configuration is reloaded.
*   `Start(ctx context.Context) error`: Initializes and starts the module. It should block until the module is fully ready to accept work. This is called after all its dependencies have started.
*   `OnReady(ctx context.Context) error`: Called after the module itself and all its declared dependencies have successfully started. This is a good place for modules to register services or perform actions that rely on the full system being operational. Errors in `OnReady` are logged but do not halt kernel startup since `OnReady` may involve non-critical post-startup tasks.
*   `RegisterServices(reg registry.Registry) error`: Is called after the module has successfully started, allowing it to register its services with the kernel's registry. Errors in `RegisterServices` will halt the entire kernel startup process, as service registration is critical for inter-module communication.
*   `Stop(ctx context.Context) error`: Gracefully shuts down the module, honoring the provided context for cancellation. This is called before its dependents are stopped.
*   `OnConfigChanged(ctx context.Context, newCfg interface{}) error`: Called when the application's configuration is reloaded. Modules should re-read and apply relevant configuration changes here.
*   `ShutdownTimeout() time.Duration`: Returns the maximum duration the kernel should wait for the module to stop gracefully. If the module does not stop within this timeout, the kernel will force its termination.
*   `UnregisterServices(reg registry.Registry)`: Called when the module is stopped or removed, allowing it to unregister its services from the kernel's registry.

**Important Notes:**
*   Modules should not directly depend on Gateways; they should expose application services that Gateways can call. Dependency wiring happens in `main`.
*   The kernel ensures modules are started in dependency order and stopped in reverse dependency order.

### 2.2. Service Registry (`core.registry`)
The `core/registry` package provides a simple service registry that allows modules to register and discover services provided by other modules. This facilitates inter-module communication without creating direct dependencies between modules, adhering to the principle of loose coupling.

*   **Registration:** Modules register their services (typically interfaces or concrete types) with the registry during their `RegisterServices` lifecycle method.
*   **Discovery:** Other modules can then retrieve these registered services from the registry by name or type.

### 2.3. Module Dependency Management and Versioning
The kernel automatically manages the startup and shutdown order of modules based on their declared dependencies and semantic version constraints.
*   **Dependency Graph:** When the kernel starts, it performs a topological sort of all enabled modules using their `Dependencies()` method. This ensures that a module's dependencies are always started before the module itself.
*   **Version Constraints:** Each module declares its own semantic version (`Version()`) and specifies version constraints for its dependencies (e.g., "module-a": "^1.0.0", "module-b": ">=2.1.0 <3.0.0"). The kernel validates these constraints during startup. If a dependency's version does not satisfy the constraint, or if a circular dependency is detected, the kernel will return an `errVersion` or `circular dependency detected` error, respectively, and halt startup.
*   **Shutdown Order:** During shutdown, modules are stopped in reverse dependency order, ensuring that a module's dependents are stopped before the module itself.

### 2.3. Panic Recovery Mechanism
The kernel implements a panic recovery mechanism using a `safelyExecute` helper function for all module and gateway lifecycle calls (e.g., `OnLoad`, `Configure`, `Start`, `Stop`, `OnReady`, `OnConfigChanged`). If a panic occurs within any of these methods, the `safelyExecute` function catches the panic, logs it with detailed context (component type, name, operation, and panic value), and converts it into a standard Go error. This prevents a single module or gateway panic from crashing the entire kernel, enhancing the application's stability and resilience.

### 2.4. Graceful Shutdown with Configurable Timeouts
Both `Module` and `Gateway` interfaces now include a `ShutdownTimeout()` method. This allows each component to specify its own maximum duration for graceful shutdown.
*   During kernel shutdown, when `Stop()` is called on a module or gateway, the kernel will wait for the component to complete its shutdown within the specified `ShutdownTimeout()`.
*   If a component fails to stop within its declared timeout, the kernel will log a warning and proceed, preventing a single misbehaving component from indefinitely delaying the entire application shutdown. If `ShutdownTimeout()` returns a duration less than or equal to zero, a default timeout is applied.

### 2.5. Health Check Granularity
The kernel provides a `Health()` method that returns the aggregated health status of all registered components. Modules and gateways can optionally implement the `HealthReporter` interface to provide more detailed health information.
*   `HealthStatus` struct: Represents the health of a component with `Status` (e.g., "healthy", "degraded", "unhealthy"), an optional `Message`, and an `Error` field.
*   `HealthReporter` interface: Defines a `Health(ctx context.Context) HealthStatus` method that modules and gateways can implement to report their internal health.

### 2.6. Security and Access Control
The kernel implements comprehensive security controls to ensure that only authorized principals can perform sensitive operations. All module management operations require proper authentication and authorization.

#### Security Requirements
* **Principal Context**: All security-critical operations require a `context.Context` containing a valid `auth.Principal`
* **Permission Checks**: Each operation validates specific permissions before proceeding
* **Audit Logging**: All security decisions are logged with full context for monitoring and forensics

#### Required Permissions
* `kernel.module.add` - Required for adding new modules
* `kernel.module.remove` - Required for removing modules
* `kernel.module.enable` - Required for enabling modules
* `kernel.module.disable` - Required for disabling modules

#### Security Best Practices
* Always provide proper principal context for module operations
* Use specific permissions rather than wildcards in production
* Implement proper error handling for security violations
* Log security events for monitoring and compliance

### 2.7. Logging Consistency
The kernel ensures consistent use of `context.Context` in its logging calls, especially within `safelyExecute` and other lifecycle methods. This allows for better trace correlation and contextual logging, including security event logging.

### 2.5. Gateway Interface (`kernel.Gateway`)
A `Gateway` represents a network interface or protocol handler that the Kernel can manage. Gateways are started after modules and stopped before modules to ensure modules are available when gateways begin accepting traffic, and that traffic ceases before modules shut down. The `Name` of a gateway must be unique across all gateways.

**Methods:**
*   `Name() string`: Returns the unique name of the gateway.
*   `Start(ctx context.Context) error`: Initializes and starts the gateway. It should block until the gateway is ready.
*   `Stop(ctx context.Context) error`: Gracefully shuts down the gateway, honoring the provided context for cancellation.
*   `Configure(cfg interface{}) error`: Provides the gateway with its specific configuration. This method should be called before `Start`.
*   `ShutdownTimeout() time.Duration`: Returns the maximum duration the kernel should wait for the gateway to stop gracefully.

**Lifecycle Order:**
*   Gateways start after modules.
*   Gateways stop before modules.

## 3. Kernel API Reference (`kernel.Kernel`)

### 3.1. Instantiation
`New(cfg *config.Config, ac auth.AccessController) Kernel`
*   Creates and returns a new Kernel implementation.
*   `cfg`: Application configuration, accessible to modules.
*   `ac`: An `AccessController` for authentication and authorization. If `nil`, a default "allow all" controller (or one configured via `AuthConfig` if available) is used.

### 3.2. Module Management
*   `AddModule(ctx context.Context, m Module) error`: **SECURITY-CRITICAL** - Registers a new module with the kernel. Requires a context with a valid principal for authentication and authorization. If the kernel is already running, the module will be started immediately. Returns an error if the module is `nil`, has an empty name, a duplicate name, if security validation fails, or if its `OnLoad`, `Configure`, `Start`, `RegisterServices`, or `OnReady` methods fail.
    *   **Security Requirements**: Requires `kernel.module.add` permission
    *   **Note on Dynamic Dependency Re-evaluation**: When adding a module to a running kernel, its dependencies are not fully re-evaluated against all currently running modules. For a more robust dynamic integration, a mechanism to re-evaluate the full dependency graph or trigger a kernel reload would be required. This is a significant architectural decision and is currently out of scope.
*   `RemoveModule(ctx context.Context, name string) error`: **SECURITY-CRITICAL** - Unregisters and stops a module by its name. Requires a context with a valid principal for authentication and authorization. Returns an error if the module is not found, security validation fails, or fails to stop.
    *   **Security Requirements**: Requires `kernel.module.remove` permission
*   `GetModule(name string) (Module, bool)`: Retrieves a module by its name. Returns the module and a boolean indicating if it was found.
*   `ListModules() []string`: Returns a sorted list of names of all registered modules.
*   `ReloadModule(m Module) error`: Attempts to stop an existing module, replace it with a new instance, and then start the new instance. Includes a rollback mechanism if the new module fails to configure or start.
*   `EnableModule(ctx context.Context, name string) error`: **SECURITY-CRITICAL** - Marks a module as enabled and starts it if the kernel is running. Requires a context with a valid principal for authentication and authorization.
    *   **Security Requirements**: Requires `kernel.module.enable` permission
*   `DisableModule(ctx context.Context, name string) error`: **SECURITY-CRITICAL** - Marks a module as disabled and stops it if the kernel is running. Requires a context with a valid principal for authentication and authorization.
    *   **Security Requirements**: Requires `kernel.module.disable` permission

### 3.3. Gateway Management
*   `AddGateway(g Gateway) error`: Registers a new gateway with the kernel. If the kernel is already running, the gateway will be started immediately. Returns an error if the gateway is `nil`, has an empty name, a duplicate name, or if its `Configure` or `Start` methods fail.
*   `RemoveGateway(name string) error`: Unregisters and stops a gateway by its name. Returns an error if the gateway is not found or fails to stop.
*   `GetGateway(name string) (Gateway, bool)`: Retrieves a gateway by its name. Returns the gateway and a boolean indicating if it was found.
*   `ListGateways() []string`: Returns a sorted list of names of all registered gateways.

### 3.4. Kernel Lifecycle Management
*   `Start(ctx context.Context) error`: Initializes and starts all registered modules (only enabled ones) and then all registered gateways. Modules are started before gateways to ensure application services are ready before network traffic. Errors in `Module.RegisterServices` will now halt kernel startup, as service registration is critical for inter-module communication. Errors in `Module.OnReady` are logged, and the specific module that failed is stopped, but the kernel startup is not halted, as `OnReady` might involve non-critical post-startup tasks. Returns an error if the kernel is already running, if any module/gateway fails to start, or if dependency resolution/version checks fail.
*   `Stop(ctx context.Context) error`: Gracefully shuts down all registered gateways and then all registered modules (only enabled ones). Gateways are stopped before modules to ensure traffic ceases before application services shut down. Returns an error if the kernel is not running or if any component fails to stop (captures the first error encountered).
*   `Running() bool`: Returns `true` if the kernel is currently running, `false` otherwise.

### 3.5. Kernel Service Access
*   `GetRegistry() registry.Registry`: Returns the kernel's service registry, allowing access to registered services from modules and other components.
*   `Health(ctx context.Context) map[string]HealthStatus`: Returns the aggregated health status of all registered components, including modules and gateways that implement the `HealthReporter` interface.
    *   `HealthStatus` struct contains:
        *   `Status string`: Health status ("healthy", "degraded", "unhealthy")
        *   `Message string`: Optional descriptive message
        *   `Error string`: Error description if unhealthy

### 3.6. Development Utilities
*   `RunDev(ctx context.Context, opts DevOptions) error`: Starts the kernel if not already running, executes a controlled development/testing cycle according to `opts`, and stops the kernel if it was started by this call. The `RunDev` function's loop for handling ticks now uses a single `select` statement that checks both the context cancellation and the delay, simplifying context handling. The kernel's shutdown initiated by `RunDev` now respects the `RunDev`'s context for graceful termination.
*   `DevOptions`: Configuration for `RunDev`.
    *   `Ticks int`: Defines how many ticks to run. If `<= 0`, it runs until the context is canceled.
    *   `OnTick func(i int)`: A callback function invoked once per tick with a 1-based index.
    *   `Delay time.Duration`: An optional sleep duration between ticks.

## 4. Error Handling
The `kernel` package defines several predefined errors for common operations:
*   `errDuplicate`: Returned when attempting to add a module/gateway with a name that already exists.
*   `errNotFound`: Returned when a module/gateway with the specified name is not found.
*   `errAlreadyRunning`: Returned when attempting to start a kernel that is already running.
*   `errNotRunning`: Returned when attempting to stop a kernel that is not running.
*   `errVersion`: Returned when a module's declared version is invalid or a dependency's version does not satisfy the required constraint.
*   `circular dependency detected`: An error indicating that a cycle was found in the module dependency graph, preventing a valid startup order.

## 5. Usage Examples

### Initializing the Kernel
```go
package main

import (
	"acacia/core/config"
	"acacia/core/kernel"
	"context"
	"fmt"
	"log"
	"time" // Import time for ShutdownTimeout
)

func main() {
	cfg := &config.Config{} // Load your application configuration
	krn := kernel.New(cfg, nil) // Create a new kernel instance

	// Add modules and gateways
	// ...

	// Start the kernel
	if err := krn.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start kernel: %v", err)
	}
	fmt.Println("Kernel started successfully.")

	// Perform operations
	// ...

	// Stop the kernel
	if err := krn.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop kernel: %v", err)
	}
	fmt.Println("Kernel stopped successfully.")
}
```

### Adding and Starting a Basic Module and Gateway
```go
package main

import (
	"acacia/core/auth"
	"acacia/core/config"
	"acacia/core/kernel"
	"context"
	"fmt"
	"log"
	"time" // Import time for ShutdownTimeout
)

// Example Module implementation
type MyModule struct {
	name string
	version string
}

func (m *MyModule) Name() string { return m.name }
func (m *MyModule) Version() string { return m.version }
func (m *MyModule) Dependencies() map[string]string { return map[string]string{} } // No dependencies for this example
func (m *MyModule) OnLoad(ctx context.Context) error {
	fmt.Printf("Module %s: OnLoad called.\n", m.name)
	return nil
}
func (m *MyModule) Configure(cfg interface{}) error {
	fmt.Printf("Module %s: Configure called.\n", m.name)
	return nil
}
func (m *MyModule) Start(ctx context.Context) error {
	fmt.Printf("Module %s: Start called.\n", m.name)
	return nil
}
func (m *MyModule) OnReady(ctx context.Context) error {
	fmt.Printf("Module %s: OnReady called.\n", m.name)
	return nil
}
func (m *MyModule) Stop(ctx context.Context) error {
	fmt.Printf("Module %s: Stop called.\n", m.name)
	return nil
}
func (m *MyModule) OnConfigChanged(ctx context.Context, newCfg interface{}) error {
	fmt.Printf("Module %s: OnConfigChanged called.\n", m.name)
	return nil
}
func (m *MyModule) ShutdownTimeout() time.Duration {
	return 5 * time.Second // Example: allow 5 seconds for graceful shutdown
}

// Example Gateway implementation
type MyGateway struct {
	name string
}

func (g *MyGateway) Name() string { return g.name }
func (g *MyGateway) Start(ctx context.Context) error {
	fmt.Printf("Gateway %s started.\n", g.name)
	return nil
}
func (g *MyGateway) Stop(ctx context.Context) error {
	fmt.Printf("Gateway %s stopped.\n", g.name)
	return nil
}
func (g *MyGateway) Configure(cfg interface{}) error { return nil }
func (g *MyGateway) ShutdownTimeout() time.Duration {
	return 3 * time.Second // Example: allow 3 seconds for graceful shutdown
}

func main() {
	cfg := &config.Config{}
	krn := kernel.New(cfg, nil)

	// Create a principal with necessary permissions for module management
	principal := auth.NewDefaultPrincipal("system-admin", "system", []string{"kernel.module.*"})
	ctx := auth.ContextWithPrincipal(context.Background(), principal)

	module1 := &MyModule{name: "ModuleA", version: "1.0.0"}
	gateway1 := &MyGateway{name: "GatewayX"}

	// Add module with proper security context
	if err := krn.AddModule(ctx, module1); err != nil {
		log.Fatalf("Failed to add module: %v", err)
	}
	if err := krn.AddGateway(gateway1); err != nil {
		log.Fatalf("Failed to add gateway: %v", err)
	}

	if err := krn.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start kernel: %v", err)
	}
	// Output will show ModuleA started, then GatewayX started.

	// Remove module with proper security context
	if err := krn.RemoveModule(ctx, "ModuleA"); err != nil {
		log.Fatalf("Failed to remove module: %v", err)
	}

	if err := krn.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop kernel: %v", err)
	}
	// Output will show GatewayX stopped, then ModuleA stopped.
}
```

### Reloading a Module
```go
package main

import (
	"acacia/core/auth"
	"acacia/core/config"
	"acacia/core/kernel"
	"context"
	"fmt"
	"log"
	"time"
)

type ReloadableModule struct {
	name    string
	version string
}

func (m *ReloadableModule) Name() string { return m.name }
func (m *ReloadableModule) Version() string { return m.version }
func (m *ReloadableModule) Dependencies() map[string]string { return map[string]string{} }
func (m *ReloadableModule) OnLoad(ctx context.Context) error { return nil }
func (m *ReloadableModule) Configure(cfg interface{}) error { return nil }
func (m *ReloadableModule) Start(ctx context.Context) error {
	fmt.Printf("Module %s (Version %s) started.\n", m.name, m.version)
	return nil
}
func (m *ReloadableModule) OnReady(ctx context.Context) error { return nil }
func (m *ReloadableModule) Stop(ctx context.Context) error {
	fmt.Printf("Module %s (Version %s) stopped.\n", m.name, m.version)
	return nil
}
func (m *ReloadableModule) OnConfigChanged(ctx context.Context, newCfg interface{}) error {
	fmt.Printf("Module %s (Version %s): OnConfigChanged called.\n", m.name, m.version)
	return nil
}
func (m *ReloadableModule) ShutdownTimeout() time.Duration {
	return 5 * time.Second
}

func main() {
	cfg := &config.Config{}
	krn := kernel.New(cfg, nil)

	// Initial module
	moduleV1 := &ReloadableModule{name: "MyService", version: "1.0.0"}
	if err := krn.AddModule(moduleV1); err != nil {
		log.Fatalf("Failed to add module V1: %v", err)
	}

	if err := krn.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start kernel: %v", err)
	}

	// Create a new version of the module
	moduleV2 := &ReloadableModule{name: "MyService", version: "2.0.0"}
	if err := krn.ReloadModule(moduleV2); err != nil {
		log.Fatalf("Failed to reload module: %v", err)
	}
	// Output will show Module MyService (Version 1.0.0) stopped.
	// Then Module MyService (Version 2.0.0) started.

	if err := krn.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop kernel: %v", err)
	}
}
```

### Enabling/Disabling a Module
```go
package main

import (
	"acacia/core/auth"
	"acacia/core/config"
	"acacia/core/kernel"
	"context"
	"fmt"
	"log"
	"time"
)

type ToggleModule struct {
	name    string
	started bool
}

func (m *ToggleModule) Name() string { return m.name }
func (m *ToggleModule) Version() string { return "1.0.0" } // Add a version
func (m *ToggleModule) Dependencies() map[string]string { return map[string]string{} }
func (m *ToggleModule) OnLoad(ctx context.Context) error { return nil }
func (m *ToggleModule) Configure(cfg interface{}) error { return nil }
func (m *ToggleModule) Start(ctx context.Context) error {
	m.started = true
	fmt.Printf("Module %s started.\n", m.name)
	return nil
}
### Using the Health Check System
```go
package main

import (
	"acacia/core/auth"
	"acacia/core/config"
	"acacia/core/kernel"
	"context"
	"fmt"
	"log"
)

type HealthReporterModule struct {
	name    string
	version string
	healthy bool
}

func (m *HealthReporterModule) Name() string { return m.name }
func (m *HealthReporterModule) Version() string { return m.version }
func (m *HealthReporterModule) Dependencies() map[string]string { return map[string]string{} }
func (m *HealthReporterModule) SetRegistry(registry.Registry) {}
func (m *HealthReporterModule) SetEventBus(events.Bus) {}
func (m *HealthReporterModule) OnLoad(ctx context.Context) error { return nil }
func (m *HealthReporterModule) Configure(cfg interface{}) error { return nil }
func (m *HealthReporterModule) Start(ctx context.Context) error { return nil }
func (m *HealthReporterModule) OnReady(ctx context.Context) error { return nil }
func (m *HealthReporterModule) RegisterServices(registry.Registry) error { return nil }
func (m *HealthReporterModule) Stop(ctx context.Context) error { return nil }
func (m *HealthReporterModule) OnConfigChanged(ctx context.Context, newCfg interface{}) error { return nil }
func (m *HealthReporterModule) ShutdownTimeout() time.Duration { return time.Second * 5 }
func (m *HealthReporterModule) UnregisterServices(registry.Registry) {}

// Implement HealthReporter interface
func (m *HealthReporterModule) Health(ctx context.Context) kernel.HealthStatus {
	if m.healthy {
		return kernel.HealthStatus{
			Status:  "healthy",
			Message: "All services operating normally",
		}
	}
	return kernel.HealthStatus{
		Status:  "unhealthy",
		Message: "Service is experiencing issues",
		Error:   "database connection lost",
	}
}

func main() {
	cfg := &config.Config{}
	krn := kernel.New(cfg, nil)

	principal := auth.NewDefaultPrincipal("admin", "system", []string{"kernel.module.*"})
	ctx := auth.ContextWithPrincipal(context.Background(), principal)

	// Add a module that implements HealthReporter
	healthModule := &HealthReporterModule{name: "health-monitor", version: "1.0.0", healthy: true}
	if err := krn.AddModule(ctx, healthModule); err != nil {
		log.Fatalf("Failed to add health module: %v", err)
	}

	// Start kernel
	if err := krn.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start kernel: %v", err)
	}

	// Check health status
	healthStatuses := krn.Health(context.Background())
	for component, status := range healthStatuses {
		fmt.Printf("%s: %s - %s\n", component, status.Status, status.Message)
		if status.Error != "" {
			fmt.Printf("  Error: %s\n", status.Error)
		}
	}

	if err := krn.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop kernel: %v", err)
	}
}
```

### Enabling/Disabling a Module
```go
package main

import (
	"acacia/core/auth"
	"acacia/core/config"
	"acacia/core/kernel"
	"context"
	"fmt"
	"log"
	"time"
)

type ToggleModule struct {
	name    string
	started bool
}

func (m *ToggleModule) Name() string { return m.name }
func (m *ToggleModule) Version() string { return "1.0.0" } // Add a version
func (m *ToggleModule) Dependencies() map[string]string { return map[string]string{} }
func (m *ToggleModule) OnLoad(ctx context.Context) error { return nil }
func (m *ToggleModule) Configure(cfg interface{}) error { return nil }
func (m *ToggleModule) Start(ctx context.Context) error {
	m.started = true
	fmt.Printf("Module %s started.\n", m.name)
	return nil
}
func (m *ToggleModule) OnReady(ctx context.Context) error { return nil }
func (m *ToggleModule) Stop(ctx context.Context) error {
	m.started = false
	fmt.Printf("Module %s stopped.\n", m.name)
	return nil
}
func (m *ToggleModule) OnConfigChanged(ctx context.Context, newCfg interface{}) error { return nil }
func (m *ToggleModule) ShutdownTimeout() time.Duration {
	return 5 * time.Second
}

func main() {
	cfg := &config.Config{}
	krn := kernel.New(cfg, nil)

	// Create a principal with necessary permissions for module management
	principal := auth.NewDefaultPrincipal("system-admin", "system", []string{"kernel.module.*"})
	ctx := auth.ContextWithPrincipal(context.Background(), principal)

	myModule := &ToggleModule{name: "FeatureX"}
	if err := krn.AddModule(ctx, myModule); err != nil {
		log.Fatalf("Failed to add module: %v", err)
	}

	// Start kernel, module will start
	if err := krn.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start kernel: %v", err)
	}
	fmt.Printf("Is FeatureX running after kernel start? %t\n", myModule.started) // true

	// Disable module with proper security context
	if err := krn.DisableModule(ctx, "FeatureX"); err != nil {
		log.Fatalf("Failed to disable module: %v", err)
	}
	fmt.Printf("Is FeatureX running after disable? %t\n", myModule.started) // false

	// Enable module with proper security context
	if err := krn.EnableModule(ctx, "FeatureX"); err != nil {
		log.Fatalf("Failed to enable module: %v", err)
	}
	fmt.Printf("Is FeatureX running after enable? %t\n", myModule.started) // true

	if err := krn.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop kernel: %v", err)
	}
}
