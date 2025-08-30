package main

import (
	"acacia/core/events"
	"acacia/core/kernel"
	"acacia/core/registry"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
)

// TestEvent is a simple event for testing.
type TestEvent struct {
	Data string
}

func (e *TestEvent) EventType() string {
	return "test.event"
}

// NoopConfig holds configuration settings specific to the Noop module.
type NoopConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Message string        `mapstructure:"message"`
	Retries int           `mapstructure:"retries"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// A simple service for testing.
type NoopService struct{}

func (s *NoopService) DoSomething() string {
	return "did something"
}

// NoopModule implements kernel.Module.
type NoopModule struct {
	name                    string
	mu                      sync.Mutex
	started                 bool
	config                  NoopConfig
	registry                registry.Registry
	eventBus                events.Bus
	lastEvent               events.TypedEvent // To store the last received event for testing
	eventReceived           chan struct{}     // To signal when an event is received
	serviceInstance         *NoopService
	cancelEventSubscription func()
}

// SetRegistry provides the module with the kernel's service registry.
func (m *NoopModule) SetRegistry(reg registry.Registry) {
	m.registry = reg
}

// NewModule creates a new NoopModule instance.
func NewModule() kernel.Module {
	return &NoopModule{
		name:          "noop",
		eventReceived: make(chan struct{}, 1),
	}
}

// Name returns the unique name of the module.
func (m *NoopModule) Name() string { return m.name }

// Version returns the semantic version of the module.
func (m *NoopModule) Version() string { return "1.0.0" }

// Dependencies returns a map of module names to semantic version constraints.
func (m *NoopModule) Dependencies() map[string]string {
	return map[string]string{}
}

// SetEventBus provides the module with the kernel's event bus.
func (m *NoopModule) SetEventBus(bus events.Bus) {
	m.eventBus = bus
}

// OnLoad is called once when the module is first loaded by the kernel.
func (m *NoopModule) OnLoad(ctx context.Context) error {
	fmt.Printf("Noop module %s: Kernel invoked OnLoad.\n", m.name)
	return nil
}

// Configure is called by the kernel to provide the module with its specific configuration.
func (m *NoopModule) Configure(cfg interface{}) error {
	if cfg == nil {
		return nil
	}

	var noopCfg NoopConfig
	if err := mapstructure.Decode(cfg, &noopCfg); err != nil {
		return fmt.Errorf("failed to decode NoopModule config: %w", err)
	}
	m.config = noopCfg
	fmt.Printf("Noop module %s: Kernel invoked Configure with config: %+v\n", m.name, m.config)
	return nil
}

// Start initializes and starts the module.
func (m *NoopModule) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}

	if m.eventBus != nil {
		// Subscribe to a test event
		ch, cancel, err := m.eventBus.Subscribe("test.event")
		if err != nil {
			return fmt.Errorf("failed to subscribe to test.event: %w", err)
		}
		m.cancelEventSubscription = cancel
		go m.handleTestEvents(ch)
	}

	m.started = true
	fmt.Printf("Noop module %s: Kernel invoked Start. Module is now running.\n", m.name)
	return nil
}

// handleTestEvents is the handler for "test.event".
func (m *NoopModule) handleTestEvents(ch <-chan events.TypedEvent) {
	for event := range ch {
		m.mu.Lock()

		fmt.Printf("Noop module %s: Received event: %+v\n", m.name, event)
		m.lastEvent = event

		// Signal that an event was received.
		select {
		case m.eventReceived <- struct{}{}:
		default:
		}

		// Publish a response event
		if m.eventBus != nil {
			responseEvent := &TestEvent{Data: "response"}
			m.eventBus.Publish(context.Background(), "noop.response", responseEvent)
		}
		m.mu.Unlock()
	}
}

// OnReady is called after the module and all its dependencies have successfully started.
func (m *NoopModule) OnReady(ctx context.Context) error {
	fmt.Printf("Noop module %s: Kernel invoked OnReady.\n", m.name)
	return nil
}

// RegisterServices is called after the module has successfully started.
func (m *NoopModule) RegisterServices(reg registry.Registry) error {
	fmt.Printf("Noop module %s: Kernel invoked RegisterServices.\n", m.name)
	m.serviceInstance = &NoopService{}
	if err := reg.RegisterService("noop_service", m.serviceInstance, m.name); err != nil {
		return fmt.Errorf("failed to register noop_service: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the module.
func (m *NoopModule) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return nil
	}

	if m.cancelEventSubscription != nil {
		m.cancelEventSubscription()
	}

	m.started = false
	fmt.Printf("Noop module %s: Kernel invoked Stop. Module is shutting down.\n", m.name)
	return nil
}

// OnConfigChanged is called when the application's configuration is reloaded.
func (m *NoopModule) OnConfigChanged(ctx context.Context, newCfg interface{}) error {
	fmt.Printf("Noop module %s: Kernel invoked OnConfigChanged.\n", m.name)
	return m.Configure(newCfg)
}

// ShutdownTimeout returns the duration to wait for the module to stop gracefully.
func (m *NoopModule) ShutdownTimeout() time.Duration {
	return 5 * time.Second
}

// UnregisterServices is called when the module is stopped or removed.
func (m *NoopModule) UnregisterServices(reg registry.Registry) {
	fmt.Printf("Noop module %s: Kernel invoked UnregisterServices.\n", m.name)
	if m.serviceInstance != nil {
		reg.UnregisterService("noop_service")
	}
}
