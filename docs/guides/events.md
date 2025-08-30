# Event-Driven Architecture

This guide covers event-driven patterns in Acacia modules, based on the framework's event system and real module implementations like the auth module.

## Event System Overview

Acacia uses a centralized event bus (`core/events`) for inter-module and system communication:

- **Decoupled Communication** - Modules communicate without direct dependencies
- **Asynchronous Processing** - Events can be processed asynchronously
- **System Integration** - Built-in integration with kernel lifecycle events
- **Type Safety** - Strongly typed events with interfaces

## Core Event Types

### System Events

The kernel publishes system events that modules can subscribe to:

```go
// From core/kernel/kernel.go
const (
    ModuleAddedEventType    = "module.added"
    ModuleStartedEventType  = "module.started"
    ModuleStoppedEventType  = "module.stopped"
    GatewayAddedEventType   = "gateway.added"
    GatewayStartedEventType = "gateway.started"
    GatewayStoppedEventType = "gateway.stopped"
)
```

### Custom Domain Events

Modules can define their own domain events:

```go
// From modules/auth/domain/events.go
const (
    UserRegisteredEventType = "user.registered"
    UserLoggedInEventType   = "user.logged_in"
)
```

## Event Publishing Patterns

### Basic Event Publishing

```go
// In application service
func (s *AuthService) RegisterUser(ctx context.Context, id, username, password string, roles []string) (*domain.User, error) {
    // ... business logic ...

    // Publish domain event
    if s.EventBus != nil {
        event := NewUserRegisteredEvent(newUser)
        s.EventBus.Publish(ctx, UserRegisteredEventType, event)
    }

    return newUser, nil
}
```

### Event Creation Pattern

```go
// modules/auth/application/events.go
package application

import (
    "acacia/modules/auth/domain"
    "time"
)

// UserRegisteredEvent represents a user registration event.
type UserRegisteredEvent struct {
    UserID    string
    Username  string
    Timestamp time.Time
}

// EventType implements core/events.TypedEvent
func (e UserRegisteredEvent) EventType() string {
    return "user.registered"
}

// NewUserRegisteredEvent creates a new user registered event.
func NewUserRegisteredEvent(user *domain.User) UserRegisteredEvent {
    return UserRegisteredEvent{
        UserID:    user.ID(),
        Username:  user.Username,
        Timestamp: time.Now(),
    }
}
```

## Event Subscription Patterns

### Basic Event Subscription

```go
func (m *MyModule) Start(ctx context.Context) error {
    // Subscribe to system event
    eventCh, cancel, err := m.eventBus.Subscribe(kernel.GatewayStartedEventType)
    if err != nil {
        return fmt.Errorf("failed to subscribe to GatewayStartedEventType: %w", err)
    }
    m.cancelEventSubscription = cancel

    // Start event handler
    go m.handleGatewayStartedEvents(ctx, eventCh)
    return nil
}
```

### Multiple Event Subscriptions

```go
func (m *MyModule) Start(ctx context.Context) error {
    // Subscribe to multiple events
    subscriptions := []string{
        kernel.ModuleStartedEventType,
        kernel.GatewayStartedEventType,
        "my.custom.event",
    }

    for _, eventType := range subscriptions {
        eventCh, cancel, err := m.eventBus.Subscribe(eventType)
        if err != nil {
            return fmt.Errorf("failed to subscribe to %s: %w", eventType, err)
        }
        m.subscriptions = append(m.subscriptions, cancel)

        // Start handler for this event type
        go m.handleEvents(ctx, eventType, eventCh)
    }

    return nil
}
```

## Real-World Event Handling Examples

### Notification Service Event Pattern

```go
type NotificationModule struct {
    eventBus events.Bus
    registry registry.Registry
    cancelSubscription func()
}

func (m *NotificationModule) Start(ctx context.Context) error {
    // Subscribe to user-related events
    eventCh, cancel, err := m.eventBus.Subscribe("user.registered")
    if err != nil {
        return fmt.Errorf("failed to subscribe to user events: %w", err)
    }
    m.cancelSubscription = cancel

    go m.handleUserEvents(ctx, eventCh)
    return nil
}

func (m *NotificationModule) handleUserEvents(ctx context.Context, eventCh <-chan events.TypedEvent) {
    for {
        select {
        case event, ok := <-eventCh:
            if !ok {
                return
            }
            if userEvent, isUserEvent := event.(UserRegisteredEvent); isUserEvent {
                // Send welcome notification
                if err := m.sendWelcomeNotification(ctx, userEvent.UserID); err != nil {
                    logger.Error(ctx, "Failed to send welcome notification",
                        zap.String("userID", userEvent.UserID), zap.Error(err))
                }
            }
        case <-ctx.Done():
            return
        }
    }
}

func (m *NotificationModule) sendWelcomeNotification(ctx context.Context, userID string) error {
    // Implementation for sending notification
    logger.Info(ctx, "Sending welcome notification", zap.String("userID", userID))
    return nil
}
```

### Event Handler Organization

```go
// modules/auth/application/events.go
package application

import (
    "acacia/core/events"
    "acacia/modules/auth/domain"
    "context"
    "fmt"
)

// Event handler struct
type AuthEventHandler struct {
    // dependencies
    userRepo    domain.AuthPersistenceRepository
    emailSender EmailSender
}

// HandleUserRegistered processes user registration events
func (h *AuthEventHandler) HandleUserRegistered(ctx context.Context, event UserRegisteredEvent) error {
    fmt.Printf("Processing user registration event for user: %s\n", event.Username)

    // Send welcome email
    return h.emailSender.SendWelcomeEmail(ctx, event.UserID, event.Username)
}

// HandleUserLoggedIn processes user login events
func (h *AuthEventHandler) HandleUserLoggedIn(ctx context.Context, event UserLoggedInEvent) error {
    fmt.Printf("Processing user login event for user: %s\n", event.UserID)

    // Update last login time
    user, err := h.userRepo.GetUserByID(ctx, event.UserID)
    if err != nil {
        return fmt.Errorf("failed to get user for login event: %w", err)
    }

    // Update login statistics, audit logs, etc.
    return nil
}
```

## Advanced Event Patterns

### Event Filtering and Routing

```go
type EventRouter struct {
    handlers map[string][]EventHandler
}

type EventHandler interface {
    Handle(ctx context.Context, event events.TypedEvent) error
}

func (r *EventRouter) Route(ctx context.Context, event events.TypedEvent) error {
    eventType := event.EventType()
    handlers, exists := r.handlers[eventType]
    if !exists {
        return nil // No handlers for this event type
    }

    for _, handler := range handlers {
        if err := handler.Handle(ctx, event); err != nil {
            // Log error but continue processing other handlers
            logger.Error(ctx, "Event handler failed", zap.Error(err))
        }
    }

    return nil
}
```

### Event Enrichment

```go
type EventEnricher struct {
    eventBus events.Bus
}

func (e *EventEnricher) EnrichAndPublish(ctx context.Context, event events.TypedEvent) error {
    // Add context information
    enrichedEvent := e.addContextInfo(ctx, event)

    // Add metadata
    enrichedEvent = e.addMetadata(ctx, enrichedEvent)

    // Publish enriched event
    return e.eventBus.Publish(ctx, enrichedEvent.EventType(), enrichedEvent)
}

func (e *EventEnricher) addContextInfo(ctx context.Context, event events.TypedEvent) events.TypedEvent {
    // Add principal information, correlation ID, etc.
    principal := auth.PrincipalFromContext(ctx)
    if principal != nil {
        // Add principal info to event
        event = e.withPrincipal(event, principal)
    }
    return event
}
```

### Event Replay and Persistence

```go
type EventStore interface {
    Save(ctx context.Context, event events.TypedEvent) error
    GetEvents(ctx context.Context, eventType string, since time.Time) ([]events.TypedEvent, error)
}

type PersistentEventBus struct {
    eventBus events.Bus
    eventStore EventStore
}

func (p *PersistentEventBus) Publish(ctx context.Context, eventType string, event events.TypedEvent) error {
    // Store event first
    if err := p.eventStore.Save(ctx, event); err != nil {
        return fmt.Errorf("failed to store event: %w", err)
    }

    // Then publish
    return p.eventBus.Publish(ctx, eventType, event)
}
```

## Error Handling in Event Handlers

### Graceful Error Handling

```go
func (m *MyModule) handleEvents(ctx context.Context, eventCh <-chan events.TypedEvent) {
    for {
        select {
        case event, ok := <-eventCh:
            if !ok {
                return
            }

            // Handle event with error recovery
            if err := m.processEvent(ctx, event); err != nil {
                // Log error but don't crash the handler
                logger.Error(ctx, "Failed to process event",
                    zap.String("eventType", event.EventType()),
                    zap.Error(err))

                // Optionally send to dead letter queue
                m.handleFailedEvent(ctx, event, err)
            }

        case <-ctx.Done():
            return
        }
    }
}

func (m *MyModule) processEvent(ctx context.Context, event events.TypedEvent) error {
    // Process event with timeout
    processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Process the event
    switch e := event.(type) {
    case kernel.ModuleStartedEvent:
        return m.handleModuleStarted(processCtx, e)
    case MyCustomEvent:
        return m.handleCustomEvent(processCtx, e)
    default:
        logger.Warn(ctx, "Unknown event type", zap.String("eventType", event.EventType()))
        return nil
    }
}
```

### Dead Letter Queue Pattern

```go
type DeadLetterQueue struct {
    events []FailedEvent
    mu     sync.RWMutex
}

type FailedEvent struct {
    Event     events.TypedEvent
    Error     error
    Timestamp time.Time
    Retries   int
}

func (dlq *DeadLetterQueue) Add(event events.TypedEvent, err error) {
    dlq.mu.Lock()
    defer dlq.mu.Unlock()

    dlq.events = append(dlq.events, FailedEvent{
        Event:     event,
        Error:     err,
        Timestamp: time.Now(),
        Retries:   0,
    })
}

func (dlq *DeadLetterQueue) Retry(ctx context.Context, processor EventProcessor) error {
    dlq.mu.Lock()
    eventsToRetry := make([]FailedEvent, len(dlq.events))
    copy(eventsToRetry, dlq.events)
    dlq.mu.Unlock()

    var lastErr error
    for _, failedEvent := range eventsToRetry {
        if err := processor.Process(ctx, failedEvent.Event); err != nil {
            lastErr = err
            // Increment retry count
            failedEvent.Retries++

            // If max retries exceeded, move to permanent failure
            if failedEvent.Retries >= 3 {
                logger.Error(ctx, "Event failed permanently",
                    zap.String("eventType", failedEvent.Event.EventType()),
                    zap.Error(err))
            }
        } else {
            // Success - remove from DLQ
            dlq.remove(failedEvent)
        }
    }

    return lastErr
}
```

## Event Testing Patterns

### Mock Event Bus

```go
type MockEventBus struct {
    publishedEvents []PublishedEvent
    subscriptions   map[string][]chan events.TypedEvent
    mu              sync.RWMutex
}

type PublishedEvent struct {
    EventType string
    Event     events.TypedEvent
}

func (m *MockEventBus) Publish(ctx context.Context, eventType string, event events.TypedEvent) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.publishedEvents = append(m.publishedEvents, PublishedEvent{
        EventType: eventType,
        Event:     event,
    })

    // Send to subscribers
    if subscribers, exists := m.subscriptions[eventType]; exists {
        for _, ch := range subscribers {
            select {
            case ch <- event:
            default:
                // Non-blocking send
            }
        }
    }

    return nil
}

func (m *MockEventBus) Subscribe(eventType string) (<-chan events.TypedEvent, func(), error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    ch := make(chan events.TypedEvent, 100)
    if m.subscriptions == nil {
        m.subscriptions = make(map[string][]chan events.TypedEvent)
    }
    m.subscriptions[eventType] = append(m.subscriptions[eventType], ch)

    cancel := func() {
        m.mu.Lock()
        defer m.mu.Unlock()
        // Remove channel from subscriptions
        if subs, exists := m.subscriptions[eventType]; exists {
            for i, subscriber := range subs {
                if subscriber == ch {
                    m.subscriptions[eventType] = append(subs[:i], subs[i+1:]...)
                    break
                }
            }
        }
        close(ch)
    }

    return ch, cancel, nil
}

func (m *MockEventBus) GetPublishedEvents() []PublishedEvent {
    m.mu.RLock()
    defer m.mu.RUnlock()

    events := make([]PublishedEvent, len(m.publishedEvents))
    copy(events, m.publishedEvents)
    return events
}
```

### Testing Event Handlers

```go
func TestAuthModule_GatewayStartedEvent(t *testing.T) {
    // Setup
    mockRegistry := &MockRegistry{}
    mockEventBus := &MockEventBus{}
    mockGateway := &MockGateway{}

    // Register mock gateway
    mockRegistry.RegisterGateway("httpapi", mockGateway)

    module := &AuthModule{
        registry: mockRegistry,
        eventBus: mockEventBus,
    }

    // Start module
    ctx := context.Background()
    err := module.Start(ctx)
    assert.NoError(t, err)

    // Publish gateway started event
    event := kernel.GatewayStartedEvent{
        GatewayName: "httpapi",
    }
    err = mockEventBus.Publish(ctx, kernel.GatewayStartedEventType, event)
    assert.NoError(t, err)

    // Wait for event processing
    time.Sleep(100 * time.Millisecond)

    // Verify handlers were registered
    assert.True(t, mockGateway.HandlersRegistered)
}

func TestAuthService_PublishesEvents(t *testing.T) {
    // Setup
    mockRepo := &MockUserRepository{}
    mockEventBus := &MockEventBus{}

    service := &AuthService{
        UserRepo: mockRepo,
        EventBus: mockEventBus,
    }

    // Register user
    user, err := service.RegisterUser(context.Background(), "id", "username", "password", []string{"user"})
    assert.NoError(t, err)
    assert.NotNil(t, user)

    // Verify event was published
    publishedEvents := mockEventBus.GetPublishedEvents()
    assert.Len(t, publishedEvents, 1)
    assert.Equal(t, "user.registered", publishedEvents[0].EventType)

    // Verify event data
    userRegisteredEvent, ok := publishedEvents[0].Event.(UserRegisteredEvent)
    assert.True(t, ok)
    assert.Equal(t, "id", userRegisteredEvent.UserID)
    assert.Equal(t, "username", userRegisteredEvent.Username)
}
```

## Best Practices

### 1. Event Design

```go
// ✅ Good: Rich event with business meaning
type OrderPlacedEvent struct {
    OrderID     string
    CustomerID  string
    Items       []OrderItem
    TotalAmount float64
    Timestamp   time.Time
}

// ❌ Bad: Anemic event
type GenericEvent struct {
    Data map[string]interface{}
}
```

### 2. Event Publishing

```go
// ✅ Good: Publish after successful operation
func (s *OrderService) PlaceOrder(ctx context.Context, order Order) error {
    // Validate and save order
    if err := s.repo.Save(ctx, order); err != nil {
        return err
    }

    // Publish event only after successful save
    event := NewOrderPlacedEvent(order)
    return s.eventBus.Publish(ctx, OrderPlacedEventType, event)
}

// ❌ Bad: Publish before operation completes
func (s *OrderService) PlaceOrder(ctx context.Context, order Order) error {
    // Publish event before save
    event := NewOrderPlacedEvent(order)
    s.eventBus.Publish(ctx, OrderPlacedEventType, event) // Wrong!

    return s.repo.Save(ctx, order)
}
```

### 3. Event Handler Robustness

```go
// ✅ Good: Idempotent handlers
func (h *OrderHandler) HandleOrderPlaced(ctx context.Context, event OrderPlacedEvent) error {
    // Check if already processed
    if h.isProcessed(event.OrderID) {
        return nil
    }

    // Process order
    if err := h.sendConfirmationEmail(ctx, event); err != nil {
        return err
    }

    // Mark as processed
    return h.markProcessed(event.OrderID)
}
```

### 4. Resource Management

```go
// ✅ Good: Proper cleanup
func (m *MyModule) Start(ctx context.Context) error {
    eventCh, cancel, err := m.eventBus.Subscribe("my.event")
    if err != nil {
        return err
    }
    m.cancelSubscription = cancel

    go m.handleEvents(ctx, eventCh)
    return nil
}

func (m *MyModule) Stop(ctx context.Context) error {
    if m.cancelSubscription != nil {
        m.cancelSubscription()
    }
    return nil
}
```

## Common Patterns and Anti-Patterns

### ✅ Good Patterns

1. **Event Sourcing** - Use events as the primary source of truth
2. **CQRS** - Separate read and write models with events
3. **Event-Driven Microservices** - Loose coupling between modules
4. **Saga Pattern** - Handle distributed transactions with events
5. **Event Replay** - Rebuild state from event history

### ❌ Anti-Patterns

1. **Event Chaining** - Events triggering events in long chains
2. **Large Events** - Events containing too much data
3. **Event Overloading** - Single event handling multiple concerns
4. **Synchronous Events** - Blocking on event processing
5. **Missing Error Handling** - Events failing silently

## Performance Considerations

### Event Throughput

```go
type BufferedEventBus struct {
    eventBus events.Bus
    buffer   chan events.TypedEvent
    workers  int
}

func (b *BufferedEventBus) Publish(ctx context.Context, eventType string, event events.TypedEvent) error {
    select {
    case b.buffer <- event:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Buffer full, drop event or return error
        return fmt.Errorf("event buffer full")
    }
}
```

### Event Filtering

```go
type FilteredEventBus struct {
    eventBus events.Bus
    filters  []EventFilter
}

type EventFilter interface {
    ShouldProcess(event events.TypedEvent) bool
}

func (b *FilteredEventBus) Subscribe(eventType string) (<-chan events.TypedEvent, func(), error) {
    ch, cancel, err := b.eventBus.Subscribe(eventType)
    if err != nil {
        return nil, nil, err
    }

    filteredCh := make(chan events.TypedEvent, 100)

    go func() {
        defer close(filteredCh)
        for event := range ch {
            if b.shouldProcess(event) {
                select {
                case filteredCh <- event:
                default:
                    // Drop event if buffer full
                }
            }
        }
    }()

    return filteredCh, cancel, nil
}
```

This comprehensive guide covers event-driven patterns from basic usage to advanced scenarios, providing module developers with the knowledge to build robust, event-driven Acacia modules.
