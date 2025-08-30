# Cross-Module Communication

This guide covers service registry patterns and inter-module communication based on the Acacia framework's architecture and real module implementations.

## Service Registry Overview

The service registry (`core/registry`) is the central hub for module communication in Acacia. It allows modules to:

- **Register services** they provide (interfaces, functions, structs)
- **Discover services** provided by other modules
- **Communicate safely** without tight coupling
- **Enable loose coupling** through dependency injection

## Service Registration Patterns

### Basic Service Registration

```go
func (m *MyModule) RegisterServices(reg registry.Registry) error {
    // Register your service with a unique name
    return reg.RegisterService("myservice", m.myService, m.Name())
}
```

### Interface-Based Registration

```go
// Define your service interface
type MyService interface {
    DoSomething(ctx context.Context, param string) (string, error)
}

// Implement the interface
type myServiceImpl struct {
    // dependencies
}

// Register the interface, not the implementation
func (m *MyModule) RegisterServices(reg registry.Registry) error {
    var service MyService = &myServiceImpl{}
    return reg.RegisterService("myservice", service, m.Name())
}
```

### Factory Pattern Registration

```go
func (m *MyModule) RegisterServices(reg registry.Registry) error {
    // Register a factory function that creates service instances
    factory := func() MyService {
        return &myServiceImpl{dependency: m.someDep}
    }
    return reg.RegisterService("myservice.factory", factory, m.Name())
}
```

## Service Discovery Patterns

### Basic Service Discovery

```go
func (m *MyModule) OnReady(ctx context.Context) error {
    // Get service from registry
    service, err := m.registry.GetService(ctx, "otherservice")
    if err != nil {
        return fmt.Errorf("failed to get otherservice: %w", err)
    }

    // Type assert to expected interface
    otherService, ok := service.(OtherServiceInterface)
    if !ok {
        return fmt.Errorf("otherservice is not of expected type")
    }

    m.otherService = otherService
    return nil
}
```

### Safe Service Discovery with Context

```go
// Always use context for service discovery
service, err := m.registry.GetService(ctx, "myservice")
if err != nil {
    // Handle different error types
    if errors.Is(err, registry.ErrServiceNotFound) {
        return fmt.Errorf("required service 'myservice' not available")
    }
    return fmt.Errorf("failed to get service: %w", err)
}
```

### Dependency Resolution Pattern

```go
func (m *MyModule) resolveDependencies(ctx context.Context) error {
    // Try to get optional service
    if service, err := m.registry.GetService(ctx, "optionalservice"); err == nil {
        if optionalSvc, ok := service.(OptionalService); ok {
            m.optionalService = optionalSvc
        }
    }

    // Require mandatory service
    mandatoryService, err := m.registry.GetService(ctx, "mandatoryservice")
    if err != nil {
        return fmt.Errorf("mandatory service not available: %w", err)
    }

    m.mandatoryService = mandatoryService.(MandatoryService)
    return nil
}
```

## Real-World Service Patterns

### E-commerce Service Registration

```go
type EcommerceModule struct {
    orderService *application.OrderService
    productService *application.ProductService
}

func (m *EcommerceModule) RegisterServices(reg registry.Registry) error {
    // Register multiple services from the module
    if err := reg.RegisterService("orderService", m.orderService, m.Name()); err != nil {
        return fmt.Errorf("failed to register order service: %w", err)
    }

    if err := reg.RegisterService("productService", m.productService, m.Name()); err != nil {
        return fmt.Errorf("failed to register product service: %w", err)
    }

    return nil
}
```

### Payment Processing Service Discovery

```go
type PaymentModule struct {
    paymentProcessor PaymentProcessor
    registry         registry.Registry
}

func (m *PaymentModule) OnReady(ctx context.Context) error {
    // Discover external payment service
    service, err := m.registry.GetService(ctx, "stripe.processor")
    if err != nil {
        // Fallback to mock processor if external service unavailable
        m.paymentProcessor = &MockPaymentProcessor{}
        logger.Info(ctx, "Using mock payment processor (Stripe unavailable)")
        return nil
    }

    processor, ok := service.(PaymentProcessor)
    if !ok {
        return fmt.Errorf("stripe.processor service has wrong type")
    }

    m.paymentProcessor = processor
    return nil
}
```

## Event-Driven Communication

### Publishing Events

```go
// From modules/auth/application/auth_service.go
func (s *AuthService) RegisterUser(ctx context.Context, id, username, password string, roles []string) (*domain.User, error) {
    // ... registration logic ...

    // Publish domain event
    if s.EventBus != nil {
        event := NewUserRegisteredEvent(newUser)
        s.EventBus.Publish(ctx, UserRegisteredEventType, event)
    }

    return newUser, nil
}
```

### Subscribing to Events

```go
// From modules/auth/auth.go
func (m *AuthModule) Start(ctx context.Context) error {
    // Subscribe to GatewayStartedEventType
    eventCh, cancel, err := m.eventBus.Subscribe(kernel.GatewayStartedEventType)
    if err != nil {
        return fmt.Errorf("failed to subscribe to GatewayStartedEventType: %w", err)
    }
    m.cancelEventSubscription = cancel

    // Start event handler
    go m.listenForGatewayEvents(ctx, eventCh)
    return nil
}

func (m *AuthModule) listenForGatewayEvents(ctx context.Context, eventCh <-chan events.TypedEvent) {
    for {
        select {
        case event, ok := <-eventCh:
            if !ok {
                return
            }
            if gatewayStartedEvent, isGatewayStarted := event.(kernel.GatewayStartedEvent); isGatewayStarted {
                if gatewayStartedEvent.GatewayName == "httpapi" {
                    // React to gateway being ready
                    m.RegisterHTTPHandlers(gatewayStartedEvent.Gateway)
                }
            }
        case <-ctx.Done():
            return
        }
    }
}
```

## Gateway Integration Patterns

### Gateway Service Discovery

```go
// From modules/auth/auth.go
func (m *AuthModule) RegisterHTTPHandlers(gateway interface{}) error {
    // ... handler registration logic ...
}

// In event handler
if gatewayStartedEvent.GatewayName == "httpapi" {
    // Create principal for gateway access
    principal := auth.NewDefaultPrincipal("auth", "module", []string{"core.*", "gateway.httpapi.access"})
    gatewayCtx := auth.ContextWithPrincipal(ctx, principal)

    // Get gateway from registry
    gateway, err := m.registry.GetGateway(gatewayCtx, "httpapi")
    if err != nil {
        return fmt.Errorf("failed to get httpapi gateway: %w", err)
    }

    // Register handlers
    if err := m.RegisterHTTPHandlers(gateway); err != nil {
        return fmt.Errorf("failed to register HTTP handlers: %w", err)
    }
}
```

### Gateway Service Registration

```go
// Register gateway with registry
func (k *kernel) AddGateway(gateway Gateway) error {
    // ... validation logic ...

    // Register gateway with registry
    if err := k.registry.RegisterGateway(gateway.Name(), gateway); err != nil {
        return fmt.Errorf("register gateway %s with registry: %w", name, err)
    }

    return nil
}
```

## Advanced Patterns

### Service Pooling and Management

```go
type ServicePool struct {
    services map[string]interface{}
    mu       sync.RWMutex
}

func (sp *ServicePool) GetService(ctx context.Context, name string) (interface{}, error) {
    sp.mu.RLock()
    defer sp.mu.RUnlock()

    service, exists := sp.services[name]
    if !exists {
        return nil, registry.ErrServiceNotFound
    }
    return service, nil
}

func (sp *ServicePool) RegisterService(name string, service interface{}, moduleName string) error {
    sp.mu.Lock()
    defer sp.mu.Unlock()

    if _, exists := sp.services[name]; exists {
        return fmt.Errorf("service %s already registered", name)
    }

    sp.services[name] = service
    return nil
}
```

### Service Health and Monitoring

```go
type HealthCheckedService struct {
    service interface{}
    health  HealthChecker
}

type HealthChecker interface {
    Health(ctx context.Context) error
}

func (m *MyModule) RegisterServices(reg registry.Registry) error {
    healthCheckedService := &HealthCheckedService{
        service: m.myService,
        health:  m.healthChecker,
    }

    return reg.RegisterService("myservice", healthCheckedService, m.Name())
}
```

### Circuit Breaker Pattern

```go
type CircuitBreakerService struct {
    service     interface{}
    failureCount int
    lastFailure time.Time
    state       string // "closed", "open", "half-open"
}

func (cbs *CircuitBreakerService) call(ctx context.Context, method string, args ...interface{}) (interface{}, error) {
    if cbs.state == "open" {
        if time.Since(cbs.lastFailure) > cbs.timeout {
            cbs.state = "half-open"
        } else {
            return nil, fmt.Errorf("circuit breaker is open")
        }
    }

    // Attempt call
    result, err := cbs.service.Call(method, args...)
    if err != nil {
        cbs.recordFailure()
        return nil, err
    }

    cbs.recordSuccess()
    return result, nil
}
```

## Error Handling Best Practices

### Service Discovery Error Handling

```go
func (m *MyModule) getRequiredService(ctx context.Context, serviceName string) (interface{}, error) {
    service, err := m.registry.GetService(ctx, serviceName)
    if err != nil {
        // Check for specific error types
        switch {
        case errors.Is(err, registry.ErrServiceNotFound):
            return nil, fmt.Errorf("required service '%s' not found - check module dependencies", serviceName)
        case errors.Is(err, context.DeadlineExceeded):
            return nil, fmt.Errorf("timeout getting service '%s' - service may be slow to start", serviceName)
        default:
            return nil, fmt.Errorf("failed to get service '%s': %w", serviceName, err)
        }
    }
    return service, nil
}
```

### Graceful Service Degradation

```go
func (m *MyModule) OnReady(ctx context.Context) error {
    // Try to get enhanced service, fall back to basic if not available
    if enhancedService, err := m.registry.GetService(ctx, "enhancedservice"); err == nil {
        m.service = enhancedService.(MyService)
        logger.Info(ctx, "Using enhanced service")
    } else {
        m.service = &BasicService{}
        logger.Info(ctx, "Using basic service (enhanced service not available)")
    }

    return nil
}
```

## Testing Service Registry

### Mock Registry for Testing

```go
type MockRegistry struct {
    services map[string]interface{}
    mu       sync.RWMutex
}

func (mr *MockRegistry) GetService(ctx context.Context, name string) (interface{}, error) {
    mr.mu.RLock()
    defer mr.mu.RUnlock()

    service, exists := mr.services[name]
    if !exists {
        return nil, registry.ErrServiceNotFound
    }
    return service, nil
}

func (mr *MockRegistry) RegisterService(name string, service interface{}, moduleName string) error {
    mr.mu.Lock()
    defer mr.mu.Unlock()
    mr.services[name] = service
    return nil
}
```

### Testing Service Interactions

```go
func TestMyModuleServiceInteraction(t *testing.T) {
    mockRegistry := &MockRegistry{
        services: make(map[string]interface{}),
    }

    // Register mock service
    mockService := &MockOtherService{}
    mockRegistry.RegisterService("otherservice", mockService, "test")

    // Create module with mock registry
    module := &MyModule{
        registry: mockRegistry,
    }

    // Test service discovery
    ctx := context.Background()
    err := module.OnReady(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, module.otherService)
}
```

## Common Patterns and Anti-Patterns

### ✅ Good Patterns

1. **Interface-based design** - Always register interfaces, not implementations
2. **Context propagation** - Always pass context to service calls
3. **Graceful degradation** - Handle missing optional services
4. **Clear error messages** - Provide meaningful error messages for missing services
5. **Health checks** - Implement health checks for critical services

### ❌ Anti-Patterns

1. **Tight coupling** - Direct imports between modules
2. **Service assumptions** - Assuming services are always available
3. **Global state** - Using global variables for service access
4. **Silent failures** - Ignoring service discovery errors
5. **Type assertions without checks** - Not checking types after discovery

This guide provides comprehensive patterns for building loosely coupled, maintainable modules that communicate effectively through the service registry.
