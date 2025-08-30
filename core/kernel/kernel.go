// Package kernel provides the core logic for managing the lifecycle of modules and gateways
// within the Acacia application. It acts as a central coordinator for starting, stopping,
// reloading, and managing the state of various components.
package kernel

import (
	"acacia/core/auth"     // Imports the auth package for AccessController interface.
	"acacia/core/config"   // Imports the config package for application configuration.
	"acacia/core/events"   // Imports the events package for the event bus.
	"acacia/core/logger"   // Imports the custom logger for logging events within the kernel.
	"acacia/core/metrics"  // Imports the metrics package for tracking operational metrics.
	"acacia/core/registry" // Imports the registry package for service discovery.

	"context" // Provides context for managing request-scoped values, cancellation signals, and deadlines.
	"errors"  // Standard library package for error handling.
	"fmt"     // Implements formatted I/O.
	"sort"    // Implements routines for sorting slices and user-defined collections.
	"sync"    // Provides basic synchronization primitives like mutexes.
	"time"    // Provides functionality for measuring and displaying time.

	"go.opentelemetry.io/otel"           // OpenTelemetry API for Go, used for distributed tracing.
	"go.opentelemetry.io/otel/attribute" // Provides attributes for OpenTelemetry spans.
	"go.opentelemetry.io/otel/codes"     // Provides status codes for OpenTelemetry spans.
	"go.opentelemetry.io/otel/trace"     // Provides tracing functionality for OpenTelemetry.
	"go.uber.org/zap"                    // A fast, structured, leveled logging library.

	"github.com/Masterminds/semver/v3" // For semantic versioning
)

// Module represents a self-contained feature that can be started and stopped by the Kernel.
// Implementations must be concurrency-safe with respect to Start/Stop being called at most once each.
// Start should return only after the module is ready to accept work.
// Stop should attempt a graceful shutdown honoring the provided context.
// Name must be unique across all modules.
//
// Note: Modules should not directly depend on Gateways; they should expose application services
// that Gateways can call. Dependency wiring happens in main.
//
// Minimal interface to keep Kernel generic.
//
//go:generate echo "no codegen"
type Module interface {
	// Name returns the unique name of the module.
	Name() string
	// Version returns the semantic version of the module.
	Version() string
	// Dependencies returns a map of module names to semantic version constraints.
	Dependencies() map[string]string
	// SetEventBus provides the module with the kernel's event bus.
	// This method should be called once after the module is loaded.
	SetEventBus(bus events.Bus)
	// SetRegistry provides the module with the kernel's service registry.
	// This method should be called once after the module is loaded.
	SetRegistry(reg registry.Registry)
	// OnLoad is called once when the module is first loaded by the kernel.
	// It's suitable for initial setup that doesn't require other modules to be started.
	OnLoad(ctx context.Context) error
	// Configure provides the module with its specific configuration.
	// This method should be called before Start.
	Configure(cfg interface{}) error
	// Start initializes and starts the module. It should block until the module is ready.
	Start(ctx context.Context) error
	// OnReady is called after the module and all its dependencies have successfully started.
	OnReady(ctx context.Context) error
	// RegisterServices is called after the module has successfully started, allowing it to register its services with the kernel's registry.
	RegisterServices(reg registry.Registry) error
	// Stop gracefully shuts down the module, honoring the provided context for cancellation.
	Stop(ctx context.Context) error
	// OnConfigChanged is called when the application's configuration is reloaded.
	OnConfigChanged(ctx context.Context, newCfg interface{}) error
	// ShutdownTimeout returns the duration to wait for the module to stop gracefully.
	ShutdownTimeout() time.Duration
	// UnregisterServices is called when the module is stopped or removed, allowing it to unregister its services from the kernel's registry.
	UnregisterServices(reg registry.Registry)
}

// Gateway represents a network interface or protocol handler that the Kernel can manage.
// Gateways are started after modules and stopped before modules to ensure modules are available
// when gateways begin accepting traffic, and that traffic ceases before modules shut down.
// Name must be unique across all gateways.
type Gateway interface {
	// Name returns the unique name of the gateway.
	Name() string
	// SetEventBus provides the gateway with the kernel's event bus.
	// This method should be called once after the gateway is loaded.
	SetEventBus(bus events.Bus)
	// Start initializes and starts the gateway. It should block until the gateway is ready.
	Start(ctx context.Context) error
	// Stop gracefully shuts down the gateway, honoring the provided context for cancellation.
	Stop(ctx context.Context) error
	// Configure provides the gateway with its specific configuration.
	// This method should be called before Start.
	Configure(cfg interface{}) error
	// ShutdownTimeout returns the duration to wait for the gateway to stop gracefully.
	ShutdownTimeout() time.Duration
}

// Kernel is the central coordinator which manages lifecycle of modules and gateways.
type Kernel interface {
	// AddModule registers a new module with the kernel. If the kernel is already running,
	// the module will be started immediately.
	// Requires context with principal for security validation.
	AddModule(ctx context.Context, m Module) error
	// RemoveModule unregisters and stops a module by its name.
	// Requires context with principal for security validation.
	RemoveModule(ctx context.Context, name string) error
	// GetModule retrieves a module by its name.
	GetModule(name string) (Module, bool)
	// ListModules returns a sorted list of names of all registered modules.
	ListModules() []string

	// AddGateway registers a new gateway with the kernel. If the kernel is already running,
	// the gateway will be started immediately.
	AddGateway(g Gateway) error
	// RemoveGateway unregisters and stops a gateway by its name.
	RemoveGateway(name string) error
	// GetGateway retrieves a gateway by its name.
	GetGateway(name string) (Gateway, bool)
	// ListGateways returns a sorted list of names of all registered gateways.
	ListGateways() []string

	// ReloadModule attempts to stop an existing module, replace it with a new instance,
	// and then start the new instance. Includes rollback on failure.
	ReloadModule(m Module) error
	// Start initializes and starts all registered modules and then all registered gateways.
	Start(ctx context.Context) error
	// Stop gracefully shuts down all registered gateways and then all registered modules.
	Stop(ctx context.Context) error
	// Running returns true if the kernel is currently running.
	Running() bool

	// EnableModule marks a module as enabled and starts it if the kernel is running.
	// Requires context with principal for security validation.
	EnableModule(ctx context.Context, name string) error
	// DisableModule marks a module as disabled and stops it if the kernel is running.
	// Requires context with principal for security validation.
	DisableModule(ctx context.Context, name string) error

	// RunDev starts the kernel, runs a controlled development/testing cycle, and then stops.
	// If opts.Ticks > 0, it runs exactly that many ticks; otherwise, it runs until ctx is canceled.
	RunDev(ctx context.Context, opts DevOptions) error

	// Health returns the aggregated health status of all registered components.
	Health(ctx context.Context) map[string]HealthStatus
	GetRegistry() registry.Registry
}

// HealthStatus represents the health of a component.
type HealthStatus struct {
	Status  string `json:"status"`  // "healthy", "degraded", "unhealthy"
	Message string `json:"message"` // Optional message
	Error   string `json:"error,omitempty"`
}

// HealthReporter is an optional interface for modules and gateways to report their health.
type HealthReporter interface {
	Health(ctx context.Context) HealthStatus
}

// DevOptions configures the development/testing cycle for RunDev.
type DevOptions struct {
	// Ticks defines how many ticks to run. If <= 0, run until ctx is canceled.
	Ticks int
	// OnTick is invoked once per tick with a 1-based index.
	OnTick func(i int)
	// Delay is an optional sleep between ticks.
	Delay time.Duration
}

// Predefined errors for common kernel operations.
var (
	errDuplicate      = errors.New("duplicate name")         // Returned when attempting to add a module/gateway with a name that already exists.
	errNotFound       = errors.New("not found")              // Returned when a module/gateway with the specified name is not found.
	errAlreadyRunning = errors.New("kernel already running") // Returned when attempting to start a kernel that is already running.
	errNotRunning     = errors.New("kernel not running")     // Returned when attempting to stop a kernel that is not running.
	errVersion        = errors.New("version conflict")       // Returned when a module's version is incompatible.
)

const (
	moduleOperationTimeout = 30 * time.Second // Default timeout for module operations
)

// Kernel event types
const (
	ModuleAddedEventType    = "module.added"
	ModuleStartedEventType  = "module.started"
	ModuleStoppedEventType  = "module.stopped"
	GatewayAddedEventType   = "gateway.added"
	GatewayStartedEventType = "gateway.started"
	GatewayStoppedEventType = "gateway.stopped"
)

// ModuleEvent is the base struct for module-related events.
type ModuleEvent struct {
	ModuleName string
}

func (e ModuleEvent) EventType() string { return "" } // Base implementation, overridden by specific events

// ModuleAddedEvent is published when a module is added to the kernel.
type ModuleAddedEvent struct {
	ModuleEvent
}

func (e ModuleAddedEvent) EventType() string { return ModuleAddedEventType }

// ModuleStartedEvent is published when a module successfully starts.
type ModuleStartedEvent struct {
	ModuleEvent
}

func (e ModuleStartedEvent) EventType() string { return ModuleStartedEventType }

// ModuleStoppedEvent is published when a module successfully stops.
type ModuleStoppedEvent struct {
	ModuleEvent
}

func (e ModuleStoppedEvent) EventType() string { return ModuleStoppedEventType }

// GatewayEvent is the base struct for gateway-related events.
type GatewayEvent struct {
	GatewayName string
}

func (e GatewayEvent) EventType() string { return "" } // Base implementation, overridden by specific events

// GatewayAddedEvent is published when a gateway is added to the kernel.
type GatewayAddedEvent struct {
	GatewayEvent
}

func (e GatewayAddedEvent) EventType() string { return GatewayAddedEventType }

// GatewayStartedEvent is published when a gateway successfully starts.
type GatewayStartedEvent struct {
	GatewayEvent
}

func (e GatewayStartedEvent) EventType() string { return GatewayStartedEventType }

// GatewayStoppedEvent is published when a gateway successfully stops.
type GatewayStoppedEvent struct {
	GatewayEvent
}

func (e GatewayStoppedEvent) EventType() string { return GatewayStoppedEventType }

// New returns a new Kernel implementation.
// It initializes the internal maps for modules and gateways and sets the application configuration.
func New(cfg *config.Config, ac auth.AccessController) Kernel {
	if ac == nil {
		// Provide a default "allow all" controller for development/unauthenticated scenarios.
		// This is useful for setups where auth is not a concern.
		ac = auth.NewDefaultAccessController(nil)
	}
	k := &kernel{
		config:           cfg,
		modules:          make(map[string]Module),
		gateways:         make(map[string]Gateway),
		moduleStates:     make(map[string]bool),
		accessController: ac,
		registry:         registry.NewDefaultRegistry(ac), // Initialize the service registry with the access controller
		eventBus:         events.New(),                    // Initialize the event bus
	}
	// Watch for config changes and notify modules
	cfg.AddConfigChangeHook(func(newCfg *config.Config) {
		configChangeTimeout := time.Duration(newCfg.Timeouts.ConfigChange) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), configChangeTimeout)
		defer cancel() // Ensure the context is cancelled to release resources.

		k.mu.RLock()
		defer k.mu.RUnlock()
		for _, m := range k.modules {
			if err := m.OnConfigChanged(ctx, newCfg); err != nil {
				logger.Error(ctx, "Module failed to handle config change", zap.String("module", m.Name()), zap.Error(err))
			}
		}
		for _, g := range k.gateways {
			// Pass the specific gateway config to the gateway's Configure method
			if gatewayCfg, ok := newCfg.Gateways[g.Name()]; ok {
				if err := g.Configure(gatewayCfg); err != nil {
					logger.Error(ctx, "Gateway failed to re-configure on config change", zap.String("gateway", g.Name()), zap.Error(err))
				}
			} else {
				logger.Warn(ctx, "No configuration found for gateway during config change", zap.String("gateway", g.Name()))
			}
		}
	})
	return k
}

// kernel is the concrete implementation of the Kernel interface.
type kernel struct {
	mu               sync.RWMutex          // Mutex to protect concurrent access to modules, gateways, and running state.
	config           *config.Config        // Application configuration, accessible to modules.
	modules          map[string]Module     // Stores registered modules.
	gateways         map[string]Gateway    // Stores registered gateways.
	moduleStates     map[string]bool       // Stores the enabled/disabled state of modules.
	running          bool                  // Indicates whether the kernel is currently running.
	accessController auth.AccessController // New field
	registry         registry.Registry     // Service registry for inter-module communication
	eventBus         events.Bus            // Event bus for system-wide events
}

// GetRegistry returns the kernel's service registry.
func (k *kernel) GetRegistry() registry.Registry {
	return k.registry
}

// safelyExecute runs a function and recovers from panics, returning an error instead.
func (k *kernel) safelyExecute(ctx context.Context, componentName string, componentType string, operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(ctx, "Panic recovered in component",
				zap.String("type", componentType),
				zap.String("component", componentName),
				zap.String("operation", operation),
				zap.Any("panic", r),
			)
			err = fmt.Errorf("panic in %s %s during %s: %v", componentType, componentName, operation, r)
		}
	}()
	return fn()
}

// ReloadModule attempts to stop an existing module, replace it with a new instance,
// and then start the new instance. It includes a rollback mechanism if the new module fails to start.
func (k *kernel) ReloadModule(m Module) error {
	if m == nil {
		return fmt.Errorf("module is nil") // Error if the provided module is nil.
	}
	name := m.Name()
	if name == "" {
		return fmt.Errorf("module name is empty") // Error if the module name is empty.
	}

	k.mu.Lock()
	oldModule, exists := k.modules[name]
	if !exists {
		k.mu.Unlock()
		logger.Warn(context.Background(), "Attempted to reload non-existent module", zap.String("module", name))
		return fmt.Errorf("module %s: %w", name, errNotFound)
	}
	k.mu.Unlock()

	logger.Info(context.Background(), "Attempting to reload module", zap.String("module", name))

	// Stop the old module
	stopTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
	stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
	defer stopCancel()
	metrics.ModuleStopCounter.WithLabelValues(name, "attempt").Inc()
	err := k.safelyExecute(stopCtx, oldModule.Name(), "module", "Stop", func() error {
		return oldModule.Stop(stopCtx)
	})
	if err != nil {
		metrics.ModuleStopCounter.WithLabelValues(name, "failed").Inc()
		logger.Error(stopCtx, "Failed to stop old module during reload", zap.String("module", name), zap.Error(err))
		return fmt.Errorf("stop old module %s: %w", name, err)
	}
	metrics.ModuleStopCounter.WithLabelValues(name, "success").Inc()
	logger.Info(stopCtx, "Old module stopped during reload", zap.String("module", name))

	// Replace with new module
	k.mu.Lock()
	k.modules[name] = m
	k.mu.Unlock()
	logger.Info(context.Background(), "Module replaced in kernel map", zap.String("module", name))

	// Configure the new module
	configureTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
	if moduleConfig, ok := k.config.Modules[name]; ok {
		configureCtx, configureCancel := context.WithTimeout(context.Background(), configureTimeout)
		defer configureCancel()
		err := k.safelyExecute(configureCtx, m.Name(), "module", "Configure", func() error {
			return m.Configure(moduleConfig)
		})
		if err != nil {
			logger.Error(configureCtx, "Failed to configure new module during reload", zap.String("module", name), zap.Error(err))
			// Rollback: try to restore and start the old module
			k.mu.Lock()
			k.modules[name] = oldModule
			k.mu.Unlock()
			rollbackStartCtx, rollbackStartCancel := context.WithTimeout(context.Background(), stopTimeout) // Use stopTimeout for rollback start
			defer rollbackStartCancel()
			if rollbackErr := oldModule.Start(rollbackStartCtx); rollbackErr != nil {
				logger.Error(rollbackStartCtx, "Failed to rollback to old module after new module config failed", zap.String("module", name), zap.Error(rollbackErr))
				return fmt.Errorf("configure new module %s: %w; rollback failed: %w", name, err, rollbackErr)
			}
			return fmt.Errorf("configure new module %s: %w", name, err)
		}
	}

	// Start the new module
	startTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
	startCtx, startCancel := context.WithTimeout(context.Background(), startTimeout)
	defer startCancel()
	metrics.ModuleStartCounter.WithLabelValues(name, "attempt").Inc()
	err = k.safelyExecute(startCtx, m.Name(), "module", "Start", func() error {
		return m.Start(startCtx)
	})
	if err != nil {
		metrics.ModuleStartCounter.WithLabelValues(name, "failed").Inc()
		logger.Error(startCtx, "Failed to start new module during reload, attempting rollback", zap.String("module", name), zap.Error(err))
		// Rollback: try to restore and start the old module
		k.mu.Lock()
		k.modules[name] = oldModule
		k.mu.Unlock()
		rollbackStartCtx, rollbackStartCancel := context.WithTimeout(context.Background(), stopTimeout) // Use stopTimeout for rollback start
		defer rollbackStartCancel()
		logger.Info(rollbackStartCtx, "Attempting to restart old module after new module failed to start", zap.String("module", name))
		if rollbackErr := oldModule.Start(rollbackStartCtx); rollbackErr != nil {
			logger.Error(rollbackStartCtx, "Failed to rollback to old module", zap.String("module", name), zap.Error(rollbackErr))
			return fmt.Errorf("start new module %s: %w; rollback failed: %w", name, err, rollbackErr)
		}
		logger.Info(rollbackStartCtx, "Rollback to old module successful", zap.String("module", name))
		return fmt.Errorf("start new module %s: %w", name, err)
	}
	metrics.ModuleStartCounter.WithLabelValues(name, "success").Inc()
	logger.Info(startCtx, "New module started successfully during reload", zap.String("module", name))

	// Call OnReady for the new module
	onReadyTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
	onReadyCtx, onReadyCancel := context.WithTimeout(context.Background(), onReadyTimeout)
	defer onReadyCancel()
	err = k.safelyExecute(onReadyCtx, m.Name(), "module", "OnReady", func() error {
		return m.OnReady(onReadyCtx)
	})
	if err != nil {
		logger.Error(onReadyCtx, "Failed to call OnReady for new module during reload", zap.String("module", name), zap.Error(err))
		// Decide if this should trigger a rollback or just be logged. For now, log and continue.
	}

	return nil // Reload successful.
}

// AddModule registers a new module with the kernel. If the kernel is already running,
// the module will be started immediately.
// Requires context with principal for security validation.
func (k *kernel) AddModule(ctx context.Context, m Module) error {
	if m == nil {
		return fmt.Errorf("module is nil") // Error if the provided module is nil.
	}
	name := m.Name()
	if name == "" {
		return fmt.Errorf("module name is empty") // Error if the module name is empty.
	}

	// Security check: Require principal in context
	principal := auth.PrincipalFromContext(ctx)
	if principal == nil {
		logger.Error(ctx, "No principal in context for AddModule", zap.String("module", name))
		return fmt.Errorf("security violation: no principal in context for AddModule %s", name)
	}

	// Check permission to add modules
	if !k.accessController.HasPermission(principal, "kernel.module.add") {
		logger.Error(ctx, "Access denied for AddModule", zap.String("module", name), zap.String("principal", principal.ID()))
		return fmt.Errorf("access denied: principal %s cannot add module %s", principal.ID(), name)
	}

	k.mu.Lock()
	if _, exists := k.modules[name]; exists {
		k.mu.Unlock()
		logger.Warn(ctx, "Attempted to add duplicate module", zap.String("module", name))
		return fmt.Errorf("module %s: %w", name, errDuplicate)
	}
	k.modules[name] = m         // Add the module to the map.
	k.moduleStates[name] = true // Mark the newly added module as enabled by default.
	running := k.running
	k.mu.Unlock()

	// Provide the event bus to the module
	m.SetEventBus(k.eventBus)

	// Provide the registry to the module if it implements SetRegistry
	if regSetter, ok := m.(interface{ SetRegistry(registry.Registry) }); ok {
		regSetter.SetRegistry(k.registry)
	}

	// Call OnLoad for the new module
	onLoadTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
	onLoadCtx, onLoadCancel := context.WithTimeout(ctx, onLoadTimeout)
	defer onLoadCancel()
	err := k.safelyExecute(onLoadCtx, m.Name(), "module", "OnLoad", func() error {
		return m.OnLoad(onLoadCtx)
	})
	if err != nil {
		k.mu.Lock() // Re-acquire lock to delete module on failure
		delete(k.modules, name)
		k.mu.Unlock()
		logger.Error(onLoadCtx, "Failed to call OnLoad for module", zap.String("module", name), zap.Error(err))
		return fmt.Errorf("module %s OnLoad: %w", name, err)
	}

	// Configure the module
	if moduleConfig, ok := k.config.Modules[name]; ok {
		configureCtx, configureCancel := context.WithTimeout(ctx, moduleOperationTimeout)
		defer configureCancel()
		err := k.safelyExecute(configureCtx, m.Name(), "module", "Configure", func() error {
			return m.Configure(moduleConfig)
		})
		if err != nil {
			logger.Error(configureCtx, "Failed to configure module", zap.String("module", name), zap.Error(err))
			return fmt.Errorf("configure module %s: %w", name, err)
		}
	}

	logger.Info(ctx, "Module added", zap.String("module", name), zap.String("principal", principal.ID()))

	// If running, start immediately (dependencies will be handled by Start method)
	if running {
		// NOTE: When adding a module to a running kernel, its dependencies are not fully re-evaluated
		// against all currently running modules. For a more robust dynamic integration, consider
		// a mechanism to re-evaluate the full dependency graph or trigger a kernel reload. This is a
		// significant architectural decision and is currently out of scope for this polishing task.
		logger.Info(ctx, "Kernel is running, attempting to start newly added module (dynamic dependency re-evaluation limited)", zap.String("module", name))
		startTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
		startCtx, startCancel := context.WithTimeout(ctx, startTimeout)
		defer startCancel()
		metrics.ModuleStartCounter.WithLabelValues(name, "attempt").Inc()
		err := k.safelyExecute(startCtx, m.Name(), "module", "Start", func() error {
			return m.Start(startCtx)
		})
		if err != nil {
			k.mu.Lock()
			delete(k.modules, name)
			k.mu.Unlock()
			logger.Error(startCtx, "Failed to start module immediately after adding", zap.String("module", name), zap.Error(err))
			metrics.ModuleStartCounter.WithLabelValues(name, "failed").Inc()
			return fmt.Errorf("start module %s: %w", name, err)
		}
		metrics.ModuleStartCounter.WithLabelValues(name, "success").Inc()
		logger.Info(startCtx, "Module started immediately after adding", zap.String("module", name))

		// Call RegisterServices for the newly added module
		registerServicesTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
		registerServicesCtx, registerServicesCancel := context.WithTimeout(ctx, registerServicesTimeout)
		defer registerServicesCancel()
		err = k.safelyExecute(registerServicesCtx, m.Name(), "module", "RegisterServices", func() error {
			return m.RegisterServices(k.registry)
		})
		if err != nil {
			logger.Error(ctx, "Failed to call RegisterServices for newly added module", zap.String("module", name), zap.Error(err))
		}

		// Call OnReady for the newly added module
		onReadyTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
		onReadyCtx, onReadyCancel := context.WithTimeout(ctx, onReadyTimeout)
		defer onReadyCancel()
		err = k.safelyExecute(onReadyCtx, m.Name(), "module", "OnReady", func() error {
			return m.OnReady(onReadyCtx)
		})
		if err != nil {
			logger.Error(onReadyCtx, "Failed to call OnReady for newly added module", zap.String("module", name), zap.Error(err))
		}
	}
	return nil
}

// GetModule retrieves a module by its name.
func (k *kernel) GetModule(name string) (Module, bool) {
	k.mu.RLock()         // Acquire a read lock.
	defer k.mu.RUnlock() // Ensure the read lock is released.
	m, ok := k.modules[name]
	return m, ok // Return the module and a boolean indicating if it was found.
}

// RemoveModule unregisters and stops a module by its name.
// Requires context with principal for security validation.
func (k *kernel) RemoveModule(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("module name is empty")
	}

	// Security check: Require principal in context
	principal := auth.PrincipalFromContext(ctx)
	if principal == nil {
		logger.Error(ctx, "No principal in context for RemoveModule", zap.String("module", name))
		return fmt.Errorf("security violation: no principal in context for RemoveModule %s", name)
	}

	// Check permission to remove modules
	if !k.accessController.HasPermission(principal, "kernel.module.remove") {
		logger.Error(ctx, "Access denied for RemoveModule", zap.String("module", name), zap.String("principal", principal.ID()))
		return fmt.Errorf("access denied: principal %s cannot remove module %s", principal.ID(), name)
	}

	k.mu.Lock()
	m, exists := k.modules[name]
	if !exists {
		k.mu.Unlock()
		logger.Warn(ctx, "Attempted to remove non-existent module", zap.String("module", name))
		return fmt.Errorf("module %s: %w", name, errNotFound)
	}
	running := k.running
	delete(k.modules, name)
	delete(k.moduleStates, name) // Also remove from moduleStates
	k.mu.Unlock()

	logger.Info(ctx, "Module removed", zap.String("module", name), zap.String("principal", principal.ID()))

	// Unregister services associated with the module
	k.registry.UnregisterServicesByModule(name)
	logger.Info(ctx, "Unregistered services for module", zap.String("module", name))

	if running {
		stopTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
		stopCtx, stopCancel := context.WithTimeout(ctx, stopTimeout)
		defer stopCancel()
		metrics.ModuleStopCounter.WithLabelValues(name, "attempt").Inc()
		if err := k.safelyExecute(stopCtx, m.Name(), "module", "Stop", func() error {
			return m.Stop(stopCtx)
		}); err != nil {
			k.mu.Lock()
			k.modules[name] = m         // Restore the module to the map if stopping fails.
			k.moduleStates[name] = true // Restore its state as enabled
			k.mu.Unlock()
			logger.Error(ctx, "Failed to stop module during removal", zap.String("module", name), zap.Error(err))
			metrics.ModuleStopCounter.WithLabelValues(name, "failed").Inc()
			return fmt.Errorf("stop module %s: %w", name, err)
		}
		metrics.ModuleStopCounter.WithLabelValues(name, "success").Inc()
		logger.Info(ctx, "Module stopped during removal", zap.String("module", name))
	}
	return nil
}

// ListModules returns a sorted list of names of all registered modules.
func (k *kernel) ListModules() []string {
	k.mu.RLock()         // Acquire a read lock.
	defer k.mu.RUnlock() // Ensure the read lock is released.
	names := make([]string, 0, len(k.modules))
	for n := range k.modules {
		names = append(names, n) // Collect all module names.
	}
	sort.Strings(names) // Sort the names alphabetically.
	return names
}

// AddGateway registers a new gateway with the kernel. If the kernel is already running,
// the gateway will be started immediately.
func (k *kernel) AddGateway(g Gateway) error {
	if g == nil {
		return fmt.Errorf("gateway is nil") // Error if the provided gateway is nil.
	}
	name := g.Name()
	if name == "" {
		return fmt.Errorf("gateway name is empty") // Error if the gateway name is empty.
	}
	k.mu.Lock()
	if _, exists := k.gateways[name]; exists {
		k.mu.Unlock()
		logger.Warn(context.Background(), "Attempted to add duplicate gateway", zap.String("gateway", name)) // Updated logger call
		return fmt.Errorf("gateway %s: %w", name, errDuplicate)                                              // Error if a gateway with the same name already exists.
	}

	// Configure the gateway before adding it.
	if gatewayConfig, ok := k.config.Gateways[name]; ok {
		if err := g.Configure(gatewayConfig); err != nil {
			k.mu.Unlock()
			logger.Error(context.Background(), "Failed to configure gateway", zap.String("gateway", name), zap.Error(err))
			return fmt.Errorf("configure gateway %s: %w", name, err)
		}
	}

	k.gateways[name] = g // Add the gateway to the map.
	running := k.running
	k.mu.Unlock()

	// Provide the event bus to the gateway
	g.SetEventBus(k.eventBus)

	// Register the gateway with the registry
	if err := k.registry.RegisterGateway(name, g); err != nil {
		logger.Error(context.Background(), "Failed to register gateway with registry", zap.String("gateway", name), zap.Error(err))
		return fmt.Errorf("register gateway %s with registry: %w", name, err)
	}

	logger.Info(context.Background(), "Gateway added and registered", zap.String("gateway", name))

	if running {
		startTimeout := time.Duration(k.config.Timeouts.GatewayOperation) * time.Second
		startCtx, startCancel := context.WithTimeout(context.Background(), startTimeout)
		defer startCancel()
		metrics.GatewayStartCounter.WithLabelValues(name, "attempt").Inc()
		if err := g.Start(startCtx); err != nil {
			k.mu.Lock()
			delete(k.gateways, name) // Remove the gateway from the map if it fails to start.
			k.mu.Unlock()
			k.registry.UnregisterGateway(name) // Unregister from registry on failure
			logger.Error(context.Background(), "Failed to start gateway immediately after adding", zap.String("gateway", name), zap.Error(err))
			metrics.GatewayStartCounter.WithLabelValues(name, "failed").Inc()
			return fmt.Errorf("start gateway %s: %w", name, err)
		}
		metrics.GatewayStartCounter.WithLabelValues(name, "success").Inc()
		logger.Info(context.Background(), "Gateway started immediately after adding", zap.String("gateway", name))
	}
	return nil // Gateway added successfully.
}

// RemoveGateway unregisters and stops a gateway by its name.
func (k *kernel) RemoveGateway(name string) error {
	if name == "" {
		return fmt.Errorf("gateway name is empty") // Error if the gateway name is empty.
	}
	k.mu.Lock()
	g, exists := k.gateways[name]
	if !exists {
		k.mu.Unlock()
		logger.Warn(context.Background(), "Attempted to remove non-existent gateway", zap.String("gateway", name)) // Updated logger call
		return fmt.Errorf("gateway %s: %w", name, errNotFound)                                                     // Error if the gateway to remove does not exist.
	}
	running := k.running
	delete(k.gateways, name) // Remove the gateway from the map.
	k.mu.Unlock()

	// Unregister the gateway from the registry
	k.registry.UnregisterGateway(name)
	logger.Info(context.Background(), "Gateway removed and unregistered", zap.String("gateway", name))

	if running {
		stopTimeout := time.Duration(k.config.Timeouts.GatewayOperation) * time.Second
		stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
		defer stopCancel()
		metrics.GatewayStopCounter.WithLabelValues(name, "attempt").Inc()
		if err := g.Stop(stopCtx); err != nil {
			k.mu.Lock()
			k.gateways[name] = g // Restore the gateway to the map if stopping fails.
			k.mu.Unlock()
			k.registry.RegisterGateway(name, g) // Re-register with registry on failure
			logger.Error(context.Background(), "Failed to stop gateway during removal", zap.String("gateway", name), zap.Error(err))
			metrics.GatewayStopCounter.WithLabelValues(name, "failed").Inc()
			return fmt.Errorf("stop gateway %s: %w", name, err)
		}
		metrics.GatewayStopCounter.WithLabelValues(name, "success").Inc()
		logger.Info(context.Background(), "Gateway stopped during removal", zap.String("gateway", name))
	}
	return nil // Gateway removed successfully.
}

// GetGateway retrieves a gateway by its name.
func (k *kernel) GetGateway(name string) (Gateway, bool) {
	k.mu.RLock()         // Acquire a read lock.
	defer k.mu.RUnlock() // Ensure the read lock is released.
	g, ok := k.gateways[name]
	return g, ok // Return the gateway and a boolean indicating if it was found.
}

// ListGateways returns a sorted list of names of all registered gateways.
func (k *kernel) ListGateways() []string {
	k.mu.RLock()         // Acquire a read lock.
	defer k.mu.RUnlock() // Ensure the read lock is released.
	names := make([]string, 0, len(k.gateways))
	for n := range k.gateways {
		names = append(names, n) // Collect all gateway names.
	}
	sort.Strings(names) // Sort the names alphabetically.
	return names
}

// Start initializes and starts all registered modules and then all registered gateways.
// Modules are started before gateways to ensure application services are ready before network traffic.
func (k *kernel) Start(ctx context.Context) error {
	k.mu.Lock() // Acquire a write lock to change the running state and access maps.
	if k.running {
		k.mu.Unlock()
		logger.Warn(ctx, "Kernel already running, cannot start again.", zap.Bool("running", k.running)) // Updated logger call
		return errAlreadyRunning                                                                        // Return error if kernel is already running.
	}
	k.running = true // Set kernel to running state.
	// Create local slices of modules and gateways to avoid holding the lock during Start calls.
	// Only include modules that are enabled.
	modulesToStart := make([]Module, 0, len(k.modules))
	for name, m := range k.modules {
		if k.moduleStates[name] { // Only start if enabled
			modulesToStart = append(modulesToStart, m)
		} else {
			logger.Info(ctx, "Skipping disabled module during start", zap.String("module", name))
		}
	}
	gatewaysToStart := make([]Gateway, 0, len(k.gateways))
	for _, g := range k.gateways {
		gatewaysToStart = append(gatewaysToStart, g)
	}
	k.mu.Unlock() // Release the lock before starting components.

	tracer := otel.Tracer("acacia-kernel")         // Get OpenTelemetry tracer.
	ctx, span := tracer.Start(ctx, "Kernel.Start") // Start a new span for the Kernel.Start operation.
	defer span.End()                               // Ensure the span is ended when the function exits.

	logger.Info(ctx, "Starting kernel...")

	// Build dependency graph and get startup order
	orderedModules, err := k.getModuleStartupOrder(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(ctx, "Failed to determine module startup order", zap.Error(err))
		k.mu.Lock()
		k.running = false
		k.mu.Unlock()
		return fmt.Errorf("module startup order: %w", err)
	}

	// Start modules in dependency order
	for _, m := range orderedModules {
		moduleCtx, moduleSpan := tracer.Start(ctx, fmt.Sprintf("Module.Start: %s", m.Name()), trace.WithAttributes(attribute.String("module.name", m.Name())))
		metrics.ModuleStartCounter.WithLabelValues(m.Name(), "attempt").Inc()
		err := k.safelyExecute(moduleCtx, m.Name(), "module", "Start", func() error {
			return m.Start(moduleCtx)
		})
		if err != nil {
			moduleSpan.RecordError(err)
			moduleSpan.SetStatus(codes.Error, err.Error())
			metrics.ModuleStartCounter.WithLabelValues(m.Name(), "failed").Inc()
			logger.Error(ctx, "Failed to start module", zap.String("module", m.Name()), zap.Error(err))

			k.mu.Lock()
			k.running = false
			k.mu.Unlock()
			// best-effort stop already-started modules in reverse order
			for i := len(orderedModules) - 1; i >= 0; i-- {
				if orderedModules[i].Name() == m.Name() {
					break
				}
				_ = orderedModules[i].Stop(context.Background())
			}
			moduleSpan.End()
			return fmt.Errorf("start module %s: %w", m.Name(), err)
		}
		metrics.ModuleStartCounter.WithLabelValues(m.Name(), "success").Inc()
		logger.Info(ctx, "Module started", zap.String("module", m.Name()))
		moduleSpan.End()
	}

	// Call RegisterServices for all started modules
	for _, m := range orderedModules {
		registerServicesCtx, registerServicesSpan := tracer.Start(ctx, fmt.Sprintf("Module.RegisterServices: %s", m.Name()), trace.WithAttributes(attribute.String("module.name", m.Name())))
		err := k.safelyExecute(registerServicesCtx, m.Name(), "module", "RegisterServices", func() error {
			return m.RegisterServices(k.registry)
		})
		if err != nil {
			registerServicesSpan.RecordError(err)
			registerServicesSpan.SetStatus(codes.Error, err.Error())
			logger.Error(ctx, "Failed to call RegisterServices for module, halting kernel startup.", zap.String("module", m.Name()), zap.Error(err))
			// Stop all modules that have already started, in reverse order.
			for i := len(orderedModules) - 1; i >= 0; i-- {
				_ = orderedModules[i].Stop(context.Background())
			}
			registerServicesSpan.End()
			return fmt.Errorf("module %s RegisterServices: %w", m.Name(), err)
		}
		registerServicesSpan.End()
	}

	// Call OnReady for all started modules
	for _, m := range orderedModules {
		onReadyCtx, onReadySpan := tracer.Start(ctx, fmt.Sprintf("Module.OnReady: %s", m.Name()), trace.WithAttributes(attribute.String("module.name", m.Name())))
		err := k.safelyExecute(onReadyCtx, m.Name(), "module", "OnReady", func() error {
			return m.OnReady(onReadyCtx)
		})
		if err != nil {
			onReadySpan.RecordError(err)
			onReadySpan.SetStatus(codes.Error, err.Error())
			logger.Error(ctx, "Failed to call OnReady for module. Attempting to stop module due to OnReady failure.", zap.String("module", m.Name()), zap.Error(err))
			// Stop the module if OnReady fails to ensure stability.
			stopCtx, stopCancel := context.WithTimeout(context.Background(), m.ShutdownTimeout())
			defer stopCancel()
			if stopErr := m.Stop(stopCtx); stopErr != nil {
				logger.Error(ctx, "Failed to stop module after OnReady failure", zap.String("module", m.Name()), zap.Error(stopErr))
			}
			return fmt.Errorf("module %s OnReady: %w", m.Name(), err) // Propagate the error
		}
		onReadySpan.End()
	}

	// Then gateways
	gatewayStartTimeout := time.Duration(k.config.Timeouts.GatewayOperation) * time.Second
	for _, g := range gatewaysToStart {
		gatewayCtx, gatewaySpan := tracer.Start(ctx, fmt.Sprintf("Gateway.Start: %s", g.Name()), trace.WithAttributes(attribute.String("gateway.name", g.Name())))
		metrics.GatewayStartCounter.WithLabelValues(g.Name(), "attempt").Inc()
		err := k.safelyExecute(gatewayCtx, g.Name(), "gateway", "Start", func() error {
			// Use a context with timeout for the individual gateway start
			startCtx, startCancel := context.WithTimeout(gatewayCtx, gatewayStartTimeout)
			defer startCancel()
			return g.Start(startCtx)
		})
		if err != nil {
			gatewaySpan.RecordError(err)
			gatewaySpan.SetStatus(codes.Error, err.Error())
			metrics.GatewayStartCounter.WithLabelValues(g.Name(), "failed").Inc()
			logger.Error(ctx, "Failed to start gateway", zap.String("gateway", g.Name()), zap.Error(err))

			k.mu.Lock()
			k.running = false
			k.mu.Unlock()
			// best-effort stop previously started gateways
			for _, started := range gatewaysToStart {
				if started == g {
					break
				}
				_ = started.Stop(context.Background())
			}
			// and stop all modules in reverse order
			for i := len(orderedModules) - 1; i >= 0; i-- {
				_ = orderedModules[i].Stop(context.Background())
			}
			gatewaySpan.End()
			return fmt.Errorf("start gateway %s: %w", g.Name(), err)
		}
		metrics.GatewayStartCounter.WithLabelValues(g.Name(), "success").Inc()
		logger.Info(ctx, "Gateway started", zap.String("gateway", g.Name()))
		k.eventBus.Publish(ctx, GatewayStartedEventType, GatewayStartedEvent{GatewayEvent: GatewayEvent{GatewayName: g.Name()}})
		gatewaySpan.End()
	}
	logger.Info(ctx, "Kernel started successfully.")
	return nil
}

// Stop gracefully shuts down all registered gateways and then all registered modules.
// Gateways are stopped before modules to ensure traffic ceases before application services shut down.
func (k *kernel) Stop(ctx context.Context) error {
	k.mu.Lock() // Acquire a write lock to change the running state and access maps.
	if !k.running {
		k.mu.Unlock()
		logger.Warn(ctx, "Kernel not running, cannot stop.", zap.Bool("running", k.running)) // Updated logger call
		return errNotRunning                                                                 // Return error if kernel is not running.
	}
	k.running = false // Set kernel to not running state.

	tracer := otel.Tracer("acacia-kernel")
	ctx, span := tracer.Start(ctx, "Kernel.Stop")
	defer span.End()

	// Create local slices of modules and gateways to avoid holding the lock during Stop calls.
	// Only include modules that are enabled.
	// Get modules in reverse dependency order for graceful shutdown
	orderedModulesToStop, err := k.getModuleShutdownOrder(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error(ctx, "Failed to determine module shutdown order", zap.Error(err))
		// Proceed with stopping what we can, but log the error
	}

	gatewaysToStop := make([]Gateway, 0, len(k.gateways))
	for _, g := range k.gateways {
		gatewaysToStop = append(gatewaysToStop, g)
	}
	k.mu.Unlock() // Release the lock before stopping components.

	logger.Info(ctx, "Stopping kernel...")

	// Stop gateways first (reverse order is not necessary since we don't track order, but we stop all)
	var firstErr error // To capture the first error encountered during stopping.
	gatewayStopTimeout := time.Duration(k.config.Timeouts.GatewayOperation) * time.Second
	for _, g := range gatewaysToStop {
		timeout := g.ShutdownTimeout()
		if timeout <= 0 {
			timeout = gatewayStopTimeout
		}
		stopCtx, stopCancel := context.WithTimeout(ctx, timeout)

		gatewayCtx, gatewaySpan := tracer.Start(stopCtx, fmt.Sprintf("Gateway.Stop: %s", g.Name()), trace.WithAttributes(attribute.String("gateway.name", g.Name())))
		metrics.GatewayStopCounter.WithLabelValues(g.Name(), "attempt").Inc()
		err := k.safelyExecute(gatewayCtx, g.Name(), "gateway", "Stop", func() error {
			return g.Stop(gatewayCtx)
		})
		stopCancel()
		if err != nil {
			gatewaySpan.RecordError(err)
			gatewaySpan.SetStatus(codes.Error, err.Error())
			metrics.GatewayStopCounter.WithLabelValues(g.Name(), "failed").Inc()
			logger.Error(ctx, "Failed to stop gateway", zap.String("gateway", g.Name()), zap.Error(err))
			if firstErr == nil {
				firstErr = fmt.Errorf("stop gateway %s: %w", g.Name(), err)
			}
		} else {
			metrics.GatewayStopCounter.WithLabelValues(g.Name(), "success").Inc()
			logger.Info(ctx, "Gateway stopped", zap.String("gateway", g.Name()))
		}
		gatewaySpan.End()
	}
	// Then modules in reverse dependency order
	moduleStopTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
	for _, m := range orderedModulesToStop {
		timeout := m.ShutdownTimeout()
		if timeout <= 0 {
			timeout = moduleStopTimeout
		}
		stopCtx, stopCancel := context.WithTimeout(ctx, timeout)

		moduleCtx, moduleSpan := tracer.Start(stopCtx, fmt.Sprintf("Module.Stop: %s", m.Name()), trace.WithAttributes(attribute.String("module.name", m.Name())))
		metrics.ModuleStopCounter.WithLabelValues(m.Name(), "attempt").Inc()
		err := k.safelyExecute(moduleCtx, m.Name(), "module", "Stop", func() error {
			return m.Stop(moduleCtx)
		})
		stopCancel()
		if err != nil {
			moduleSpan.RecordError(err)
			moduleSpan.SetStatus(codes.Error, err.Error())
			metrics.ModuleStopCounter.WithLabelValues(m.Name(), "failed").Inc()
			logger.Error(ctx, "Failed to stop module", zap.String("module", m.Name()), zap.Error(err))
			if firstErr == nil {
				firstErr = fmt.Errorf("stop module %s: %w", m.Name(), err)
			}
		} else {
			metrics.ModuleStopCounter.WithLabelValues(m.Name(), "success").Inc()
			logger.Info(ctx, "Module stopped", zap.String("module", m.Name()))
		}
		moduleSpan.End()
	}
	logger.Info(ctx, "Kernel stopped.")
	return firstErr
}

// Running returns true if the kernel is currently running.
func (k *kernel) Running() bool {
	k.mu.RLock()         // Acquire a read lock.
	defer k.mu.RUnlock() // Ensure the read lock is released.
	return k.running
}

// Health returns the aggregated health status of all registered components.
func (k *kernel) Health(ctx context.Context) map[string]HealthStatus {
	k.mu.RLock()
	defer k.mu.RUnlock()

	healthStatuses := make(map[string]HealthStatus)

	// Check modules
	for name, m := range k.modules {
		status := HealthStatus{Status: "healthy", Message: "No specific health reporter implemented"}
		if hr, ok := m.(HealthReporter); ok {
			status = hr.Health(ctx)
		}
		healthStatuses[fmt.Sprintf("module:%s", name)] = status
	}

	// Check gateways
	for name, g := range k.gateways {
		status := HealthStatus{Status: "healthy", Message: "No specific health reporter implemented"}
		if hr, ok := g.(HealthReporter); ok {
			status = hr.Health(ctx)
		}
		healthStatuses[fmt.Sprintf("gateway:%s", name)] = status
	}

	return healthStatuses
}

// getModuleStartupOrder performs a topological sort to determine the correct module startup order.
func (k *kernel) getModuleStartupOrder(ctx context.Context) ([]Module, error) {
	// This function assumes the caller has already acquired the necessary lock (e.g., k.mu.RLock() or k.mu.Lock())

	// Filter for enabled modules
	enabledModules := make(map[string]Module)
	for name, m := range k.modules {
		if k.moduleStates[name] {
			enabledModules[name] = m
		} else {
			logger.Info(ctx, "Skipping disabled module in startup order calculation", zap.String("module", name))
		}
	}

	// Build adjacency list and in-degree map
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	for name := range enabledModules {
		inDegree[name] = 0 // Initialize all enabled modules with 0 in-degree
	}

	for name, m := range enabledModules {
		// Get module version
		if _, err := semver.NewVersion(m.Version()); err != nil {
			return nil, fmt.Errorf("module %q has invalid version %q: %w", name, m.Version(), err)
		}

		for depName, constraintStr := range m.Dependencies() {
			dep, exists := enabledModules[depName]
			if !exists {
				return nil, fmt.Errorf("module %q depends on non-existent or disabled module %q", name, depName)
			}

			// Check version constraint
			c, err := semver.NewConstraint(constraintStr)
			if err != nil {
				return nil, fmt.Errorf("module %q has invalid version constraint for dependency %q: %w", name, depName, err)
			}

			depVersion, err := semver.NewVersion(dep.Version())
			if err != nil {
				return nil, fmt.Errorf("dependency module %q has invalid version %q: %w", depName, dep.Version(), err)
			}

			if !c.Check(depVersion) {
				return nil, fmt.Errorf("module %q requires version %q of %q, but found version %q: %w", name, constraintStr, depName, dep.Version(), errVersion)
			}

			graph[depName] = append(graph[depName], name) // Dependency -> Dependent
			inDegree[name]++
		}
	}

	// Kahn's algorithm for topological sort
	var queue []Module
	for name, m := range enabledModules {
		if inDegree[name] == 0 {
			queue = append(queue, m)
		}
	}

	var orderedModules []Module
	for len(queue) > 0 {
		m := queue[0]
		queue = queue[1:]
		orderedModules = append(orderedModules, m)

		for _, dependentName := range graph[m.Name()] {
			inDegree[dependentName]--
			if inDegree[dependentName] == 0 {
				queue = append(queue, enabledModules[dependentName])
			}
		}
	}

	if len(orderedModules) != len(enabledModules) {
		return nil, errors.New("circular dependency detected among modules")
	}

	return orderedModules, nil
}

// getModuleShutdownOrder determines the correct module shutdown order (reverse of startup).
func (k *kernel) getModuleShutdownOrder(ctx context.Context) ([]Module, error) {
	// This function assumes the caller has already acquired the necessary lock (e.g., k.mu.RLock() or k.mu.Lock())
	startupOrder, err := k.getModuleStartupOrder(ctx)
	if err != nil {
		return nil, err
	}

	// Reverse the startup order for shutdown
	shutdownOrder := make([]Module, len(startupOrder))
	for i, j := 0, len(startupOrder)-1; i < len(startupOrder); i, j = i+1, j-1 {
		shutdownOrder[i] = startupOrder[j]
	}
	return shutdownOrder, nil
}

// EnableModule marks a module as enabled and starts it if the kernel is running.
// Requires context with principal for security validation.
func (k *kernel) EnableModule(ctx context.Context, name string) error {
	// Security check: Require principal in context
	principal := auth.PrincipalFromContext(ctx)
	if principal == nil {
		logger.Error(ctx, "No principal in context for EnableModule", zap.String("module", name))
		return fmt.Errorf("security violation: no principal in context for EnableModule %s", name)
	}

	// Check permission to enable modules
	if !k.accessController.HasPermission(principal, "kernel.module.enable") {
		logger.Error(ctx, "Access denied for EnableModule", zap.String("module", name), zap.String("principal", principal.ID()))
		return fmt.Errorf("access denied: principal %s cannot enable module %s", principal.ID(), name)
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	m, exists := k.modules[name]
	if !exists {
		logger.Warn(ctx, "Attempted to enable non-existent module", zap.String("module", name))
		return fmt.Errorf("module %s: %w", name, errNotFound)
	}

	k.moduleStates[name] = true // Mark as enabled
	logger.Info(ctx, "Module marked as enabled", zap.String("module", name), zap.String("principal", principal.ID()))

	if k.running {
		logger.Info(ctx, "Kernel is running, attempting to start enabled module", zap.String("module", name))
		// NOTE: When enabling a module in a running kernel, its dependencies are not fully re-evaluated
		// against all currently running modules. For a more robust dynamic integration, consider
		// a mechanism to re-evaluate the full dependency graph or trigger a kernel reload. This is a
		// significant architectural decision and is currently out of scope for this polishing task.
		startTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
		startCtx, startCancel := context.WithTimeout(ctx, startTimeout)
		defer startCancel()
		metrics.ModuleStartCounter.WithLabelValues(name, "attempt").Inc()
		if err := m.Start(startCtx); err != nil {
			metrics.ModuleStartCounter.WithLabelValues(name, "failed").Inc()
			logger.Error(ctx, "Failed to start enabled module", zap.String("module", name), zap.Error(err))
			return fmt.Errorf("start enabled module %s: %w", name, err)
		}
		metrics.ModuleStartCounter.WithLabelValues(name, "success").Inc()
		logger.Info(ctx, "Enabled module started successfully", zap.String("module", name))

		// Services should already be registered if the module was previously started.
		// OnReady should also not be re-triggered on a simple enable/disable cycle.
	}
	return nil
}

// DisableModule marks a module as disabled and stops it if the kernel is running.
// Requires context with principal for security validation.
func (k *kernel) DisableModule(ctx context.Context, name string) error {
	// Security check: Require principal in context
	principal := auth.PrincipalFromContext(ctx)
	if principal == nil {
		logger.Error(ctx, "No principal in context for DisableModule", zap.String("module", name))
		return fmt.Errorf("security violation: no principal in context for DisableModule %s", name)
	}

	// Check permission to disable modules
	if !k.accessController.HasPermission(principal, "kernel.module.disable") {
		logger.Error(ctx, "Access denied for DisableModule", zap.String("module", name), zap.String("principal", principal.ID()))
		return fmt.Errorf("access denied: principal %s cannot disable module %s", principal.ID(), name)
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	m, exists := k.modules[name]
	if !exists {
		logger.Warn(ctx, "Attempted to disable non-existent module", zap.String("module", name))
		return fmt.Errorf("module %s: %w", name, errNotFound)
	}

	k.moduleStates[name] = false // Mark as disabled
	logger.Info(ctx, "Module marked as disabled", zap.String("module", name), zap.String("principal", principal.ID()))

	if k.running {
		logger.Info(ctx, "Kernel is running, attempting to stop disabled module", zap.String("module", name))
		stopTimeout := time.Duration(k.config.Timeouts.ModuleOperation) * time.Second
		stopCtx, stopCancel := context.WithTimeout(ctx, stopTimeout)
		defer stopCancel()
		metrics.ModuleStopCounter.WithLabelValues(name, "attempt").Inc()
		if err := m.Stop(stopCtx); err != nil {
			metrics.ModuleStopCounter.WithLabelValues(name, "failed").Inc()
			logger.Error(ctx, "Failed to stop disabled module", zap.String("module", name), zap.Error(err))
			return fmt.Errorf("stop disabled module %s: %w", name, err)
		}
		metrics.ModuleStopCounter.WithLabelValues(name, "success").Inc()
		logger.Info(ctx, "Disabled module stopped successfully", zap.String("module", name))
	}
	return nil
}

// RunDev starts the kernel if not already running, executes a controlled development/testing
// cycle according to opts, and stops the kernel if it was started by this call.
func (k *kernel) RunDev(ctx context.Context, opts DevOptions) error {
	tracer := otel.Tracer("acacia-kernel")          // Get OpenTelemetry tracer.
	ctx, span := tracer.Start(ctx, "Kernel.RunDev") // Start a new span for the Kernel.RunDev operation.
	defer span.End()                                // Ensure the span is ended when the function exits.

	// Determine if we need to start/stop the kernel
	k.mu.RLock() // Acquire a read lock to check running state.
	running := k.running
	k.mu.RUnlock()       // Release the read lock.
	startedHere := false // Flag to track if the kernel was started by this RunDev call.
	if !running {
		logger.Info(ctx, "RunDev: Kernel not running, starting now.") // Updated logger call
		if err := k.Start(ctx); err != nil {
			span.RecordError(err)                                               // Record error in span.
			span.SetStatus(codes.Error, err.Error())                            // Set span status to error.
			logger.Error(ctx, "RunDev: Failed to start kernel", zap.Error(err)) // Updated logger call
			return err                                                          // Return error if kernel fails to start.
		}
		startedHere = true                          // Mark that the kernel was started by this function.
		logger.Info(ctx, "RunDev: Kernel started.") // Updated logger call
	} else {
		logger.Info(ctx, "RunDev: Kernel already running.") // Updated logger call
	}

	// Execute ticks
	if opts.Ticks <= 0 {
		// Run until context canceled
		i := 0
		for {
			tickCtx, tickSpan := tracer.Start(ctx, fmt.Sprintf("RunDev.Tick: %d", i+1), trace.WithAttributes(attribute.Int("tick.number", i+1)))
			logger.Debug(tickCtx, "RunDev: Tick", zap.Int("tick", i+1))
			if opts.OnTick != nil {
				opts.OnTick(i + 1)
			}
			tickSpan.End()

			i++
			select {
			case <-ctx.Done():
				logger.Info(ctx, "RunDev: Context cancelled, stopping.")
				if startedHere {
					_ = k.Stop(ctx) // Use the provided context for shutdown
				}
				span.SetStatus(codes.Error, ctx.Err().Error())
				return ctx.Err()
			case <-time.After(opts.Delay):
				// Continue after delay, or immediately if delay is 0
			}
		}
	} else {
		for i := 1; i <= opts.Ticks; i++ {
			tickCtx, tickSpan := tracer.Start(ctx, fmt.Sprintf("RunDev.Tick: %d", i), trace.WithAttributes(attribute.Int("tick.number", i)))
			logger.Debug(tickCtx, "RunDev: Tick", zap.Int("tick", i)) // Updated logger call
			if opts.OnTick != nil {
				opts.OnTick(i) // Invoke the OnTick callback if provided.
			}
			if opts.Delay > 0 && i != opts.Ticks { // no need to delay after last tick
				select {
				case <-time.After(opts.Delay): // Wait for the specified delay.
				case <-tickCtx.Done():
					logger.Info(tickCtx, "RunDev: Tick context cancelled during delay, stopping.") // Updated logger call
					if startedHere {
						_ = k.Stop(ctx) // Use the provided context for shutdown
					}
					tickSpan.SetStatus(codes.Error, tickCtx.Err().Error()) // Set span status to error.
					tickSpan.End()                                         // End tick span.
					return tickCtx.Err()                                   // Return context cancellation error.
				}
			}
			tickSpan.End() // End tick span.
		}
		if startedHere {
			logger.Info(ctx, "RunDev: Stopping kernel after ticks.") // Updated logger call
			return k.Stop(ctx)                                       // Stop kernel if it was started by this call.
		}
		logger.Info(ctx, "RunDev: Finished ticks, kernel remains running.") // Updated logger call
		return nil                                                          // Return nil if ticks finished and kernel was already running.
	}
}
