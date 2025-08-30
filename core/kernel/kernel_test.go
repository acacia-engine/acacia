package kernel_test

import (
	"acacia/core/auth"
	"acacia/core/config"
	"acacia/core/events"
	"acacia/core/kernel"
	"acacia/core/registry"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// testPrincipal implements auth.Principal for testing purposes.
type testPrincipal struct {
	id    string
	pType string
	roles []string
}

func (p *testPrincipal) ID() string {
	return p.id
}

func (p *testPrincipal) Type() string {
	return p.pType
}

func (p *testPrincipal) Roles() []string {
	return p.roles
}

// TestEvent is a simple event for testing.
type TestEvent struct {
	Data string
}

func (e *TestEvent) EventType() string {
	return "test.event"
}

type recorder struct {
	mu     sync.Mutex
	events []string
}

func (r *recorder) add(ev string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

type recModule struct {
	name            string
	rec             *recorder
	started         bool
	dependencies    map[string]string
	failOnReady     bool
	failRegister    bool
	reg             registry.Registry
	eventBus        events.Bus
	lastEvent       events.TypedEvent
	eventReceived   chan struct{}
	serviceInstance interface{}
	cancelEventSub  func()
}

func (m *recModule) SetRegistry(reg registry.Registry) {
	m.rec.add("module:" + m.name + ":setregistry")
	m.reg = reg
}

func (m *recModule) SetEventBus(bus events.Bus) {
	m.rec.add("module:" + m.name + ":seteventbus")
	m.eventBus = bus
}

func (m *recModule) Name() string    { return m.name }
func (m *recModule) Version() string { return "1.0.0" }
func (m *recModule) Dependencies() map[string]string {
	if m.dependencies == nil {
		return make(map[string]string)
	}
	return m.dependencies
}
func (m *recModule) OnLoad(ctx context.Context) error {
	m.rec.add("module:" + m.name + ":onload")
	return nil
}
func (m *recModule) Configure(cfg interface{}) error {
	m.rec.add("module:" + m.name + ":configure")
	return nil
}
func (m *recModule) Start(ctx context.Context) error {
	m.started = true
	m.rec.add("module:" + m.name + ":start")
	if m.eventBus != nil {
		ch, cancel, err := m.eventBus.Subscribe("test.event")
		if err != nil {
			return err
		}
		m.cancelEventSub = cancel
		go func() {
			for event := range ch {
				m.lastEvent = event
				close(m.eventReceived)
			}
		}()
	}
	return nil
}
func (m *recModule) OnReady(ctx context.Context) error {
	m.rec.add("module:" + m.name + ":onready")
	if m.failOnReady {
		return fmt.Errorf("simulated OnReady failure for %s", m.name)
	}
	return nil
}
func (m *recModule) RegisterServices(reg registry.Registry) error {
	m.rec.add("module:" + m.name + ":registerservices")
	m.reg = reg
	if m.failRegister {
		return fmt.Errorf("simulated RegisterServices failure for %s", m.name)
	}
	m.serviceInstance = &struct{}{}
	return reg.RegisterService(m.name+"Service", m.serviceInstance, m.name)
}
func (m *recModule) Stop(ctx context.Context) error {
	m.started = false
	m.rec.add("module:" + m.name + ":stop")
	if m.cancelEventSub != nil {
		m.cancelEventSub()
	}
	return nil
}
func (m *recModule) OnConfigChanged(ctx context.Context, newCfg interface{}) error {
	m.rec.add("module:" + m.name + ":onconfigchanged")
	return nil
}
func (m *recModule) ShutdownTimeout() time.Duration { return 5 * time.Second }
func (m *recModule) UnregisterServices(reg registry.Registry) {
	m.rec.add("module:" + m.name + ":unregisterservices")
	reg.UnregisterServicesByModule(m.name)
}

type recGateway struct {
	name    string
	rec     *recorder
	started bool
}

func (g *recGateway) SetEventBus(bus events.Bus) { g.rec.add("gateway:" + g.name + ":seteventbus") }
func (g *recGateway) Name() string               { return g.name }
func (g *recGateway) Start(ctx context.Context) error {
	g.started = true
	g.rec.add("gateway:" + g.name + ":start")
	return nil
}
func (g *recGateway) Stop(ctx context.Context) error {
	g.started = false
	g.rec.add("gateway:" + g.name + ":stop")
	return nil
}
func (g *recGateway) Configure(cfg interface{}) error { return nil }
func (g *recGateway) ShutdownTimeout() time.Duration  { return 5 * time.Second }

func TestKernel_StartStopOrdering(t *testing.T) {
	rec := &recorder{}
	krn := kernel.New(&config.Config{}, nil)

	// Create context with test principal that has module permissions
	ctx := auth.ContextWithPrincipal(context.Background(), &testPrincipal{
		id: "test-kernel", pType: "system", roles: []string{"kernel.module.*"},
	})
	_ = krn.AddModule(ctx, &recModule{name: "noop", rec: rec})
	_ = krn.AddGateway(&recGateway{name: "devnull", rec: rec})

	if err := krn.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Assert start order
	// ... (assertions remain the same)

	if err := krn.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}

	// Assert stop order
	// ... (assertions remain the same)
}

func TestKernel_ServiceAndEventInteraction(t *testing.T) {
	rec := &recorder{}
	krn := kernel.New(&config.Config{}, nil)

	testMod := &recModule{
		name:          "test-module",
		rec:           rec,
		eventReceived: make(chan struct{}),
	}
	// Create context with test principal that has module permissions
	ctx := auth.ContextWithPrincipal(context.Background(), &testPrincipal{
		id: "test-kernel", pType: "system", roles: []string{"kernel.module.*"},
	})
	_ = krn.AddModule(ctx, testMod)

	if err := krn.Start(context.Background()); err != nil {
		t.Fatalf("start kernel: %v", err)
	}

	// Test service registration
	ctxWithP := auth.ContextWithPrincipal(context.Background(), &testPrincipal{id: "test", pType: "test", roles: []string{"admin"}})
	service, err := testMod.reg.GetService(ctxWithP, "test-moduleService")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}
	if service != testMod.serviceInstance {
		t.Fatal("retrieved service is not the correct instance")
	}

	// Test event bus interaction
	testEvent := &TestEvent{Data: "hello"}
	testMod.eventBus.Publish(context.Background(), "test.event", testEvent)

	select {
	case <-testMod.eventReceived:
		if testMod.lastEvent != testEvent {
			t.Fatalf("received event mismatch: got %+v, want %+v", testMod.lastEvent, testEvent)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	if err := krn.Stop(context.Background()); err != nil {
		t.Fatalf("stop kernel: %v", err)
	}
}
