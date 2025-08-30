package devnull

import (
	"acacia/core/events"
	"acacia/core/kernel"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
)

// DevNullConfig holds configuration settings specific to the DevNull gateway.
type DevNullConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	ListenTopic  string `mapstructure:"listen_topic"`
	ResponseData string `mapstructure:"response_data"`
}

// A simple event for testing.
type ResponseEvent struct {
	Data string
}

func (e *ResponseEvent) EventType() string {
	return "devnull.response"
}

// DevNullGateway implements kernel.Gateway.
type DevNullGateway struct {
	name                    string
	mu                      sync.Mutex
	started                 bool
	config                  DevNullConfig
	eventBus                events.Bus
	lastEvent               events.TypedEvent
	cancelEventSubscription func()
}

// NewGateway creates a new DevNullGateway instance.
func NewGateway() kernel.Gateway {
	return &DevNullGateway{name: "devnull"}
}

// SetEventBus provides the gateway with the kernel's event bus.
func (g *DevNullGateway) SetEventBus(bus events.Bus) {
	g.eventBus = bus
}

// Configure is called by the kernel to provide the gateway with its specific configuration.
func (g *DevNullGateway) Configure(cfg interface{}) error {
	if cfg == nil {
		fmt.Printf("DevNull gateway %s: No specific configuration found, using defaults.\n", g.name)
		return nil
	}

	var devnullCfg DevNullConfig
	if err := mapstructure.Decode(cfg, &devnullCfg); err != nil {
		return fmt.Errorf("failed to decode DevNullGateway config: %w", err)
	}
	g.config = devnullCfg
	fmt.Printf("DevNull gateway %s: Kernel invoked Configure with config: %+v\n", g.name, g.config)
	return nil
}

// Name returns the unique name of the gateway.
func (g *DevNullGateway) Name() string { return g.name }

// Start initializes and starts the gateway.
func (g *DevNullGateway) Start(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.started {
		return nil
	}

	if g.eventBus != nil && g.config.ListenTopic != "" {
		ch, cancel, err := g.eventBus.Subscribe(g.config.ListenTopic)
		if err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", g.config.ListenTopic, err)
		}
		g.cancelEventSubscription = cancel
		go g.handleEvents(ch)
	}

	g.started = true
	fmt.Printf("DevNull gateway %s: Kernel invoked Start. Gateway is now running.\n", g.name)
	return nil
}

// handleEvents processes incoming events from the subscribed topic.
func (g *DevNullGateway) handleEvents(ch <-chan events.TypedEvent) {
	for event := range ch {
		g.mu.Lock()
		fmt.Printf("DevNull gateway %s: Received event: %+v\n", g.name, event)
		g.lastEvent = event

		// Optionally, publish a response
		if g.eventBus != nil && g.config.ResponseData != "" {
			response := &ResponseEvent{Data: g.config.ResponseData}
			g.eventBus.Publish(context.Background(), "devnull.response", response)
		}
		g.mu.Unlock()
	}
}

// Stop gracefully shuts down the gateway.
func (g *DevNullGateway) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.started {
		return nil
	}

	if g.cancelEventSubscription != nil {
		g.cancelEventSubscription()
	}

	g.started = false
	fmt.Printf("DevNull gateway %s: Kernel invoked Stop. Gateway is shutting down.\n", g.name)
	return nil
}

// ShutdownTimeout returns the duration to wait for the gateway to stop gracefully.
func (g *DevNullGateway) ShutdownTimeout() time.Duration {
	return 5 * time.Second
}
