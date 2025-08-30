# Domain-Driven Module Architecture

This guide covers domain-driven design (DDD) principles applied to Acacia modules, based on the auth module's architecture and the framework's patterns.

## Domain-Driven Design Overview

Domain-Driven Design focuses on:
- **Domain modeling** - Expressing business concepts in code
- **Layered architecture** - Separating concerns (domain, application, infrastructure)
- **Ubiquitous language** - Consistent terminology across code
- **Bounded contexts** - Clear module boundaries

## Layered Architecture in Acacia Modules

### Standard Module Structure

```
modules/mymodule/
├── domain/           # Business domain layer
│   ├── model.go     # Domain entities and value objects
│   ├── service.go   # Domain services
│   ├── repository.go # Repository interfaces
│   └── events.go    # Domain events
├── application/     # Application layer
│   ├── service.go   # Application services (use cases)
│   ├── dto.go       # Data transfer objects
│   └── handlers.go  # Event handlers, HTTP handlers
├── infrastructure/  # Infrastructure layer
│   ├── repository.go # Repository implementations
│   ├── http.go      # HTTP/gateway adapters
│   └── external.go  # External service adapters
├── go.mod          # Module dependencies
├── go.sum
├── registry.json   # Module metadata
└── mymodule.go     # Main module file
```

### Domain Layer

**Purpose**: Contains business logic and rules, independent of any framework or external concerns.

**Characteristics**:
- Pure Go code with no external dependencies
- Business rules and validations
- Domain entities and value objects
- Repository interfaces (not implementations)
- Domain events

**Example Domain Entity**:

```go
package domain

import (
    "errors"
    "time"
)

// Order represents a customer order in the system.
type Order struct {
    orderID     string
    customerID  string
    orderItems  []OrderItem
    orderStatus OrderStatus
    createdAt   time.Time
    updatedAt   time.Time
}

// OrderItem represents an item in an order.
type OrderItem struct {
    productID string
    quantity  int
    unitPrice float64
}

// OrderStatus represents the status of an order.
type OrderStatus string

const (
    OrderPending   OrderStatus = "pending"
    OrderConfirmed OrderStatus = "confirmed"
    OrderShipped   OrderStatus = "shipped"
    OrderDelivered OrderStatus = "delivered"
    OrderCancelled OrderStatus = "cancelled"
)

// NewOrder creates a new Order with validation.
func NewOrder(id, customerID string, items []OrderItem) (*Order, error) {
    if customerID == "" {
        return nil, errors.New("customer ID cannot be empty")
    }
    if len(items) == 0 {
        return nil, errors.New("order must have at least one item")
    }

    now := time.Now()
    return &Order{
        orderID:     id,
        customerID:  customerID,
        orderItems:  items,
        orderStatus: OrderPending,
        createdAt:   now,
        updatedAt:   now,
    }, nil
}

// AddItem adds an item to the order with validation.
func (o *Order) AddItem(item OrderItem) error {
    if o.orderStatus != OrderPending {
        return errors.New("cannot add items to non-pending order")
    }
    if item.quantity <= 0 {
        return errors.New("quantity must be positive")
    }

    o.orderItems = append(o.orderItems, item)
    o.updatedAt = time.Now()
    return nil
}

// CalculateTotal calculates the total price of the order.
func (o *Order) CalculateTotal() float64 {
    total := 0.0
    for _, item := range o.orderItems {
        total += item.unitPrice * float64(item.quantity)
    }
    return total
}
```

### Repository Pattern

**Purpose**: Abstract data access behind interfaces.

```go
// modules/auth/domain/user_repository.go
package domain

import (
    "context"
)

// AuthPersistenceRepository defines the interface for user data persistence.
type AuthPersistenceRepository interface {
    SaveUser(ctx context.Context, user *User) error
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    GetUserByID(ctx context.Context, id string) (*User, error)
    DeleteUser(ctx context.Context, id string) error
    CreateTable(ctx context.Context) error // For schema management
}
```

### Domain Events

**Purpose**: Represent business events that occurred in the domain.

```go
// modules/auth/domain/events.go
package domain

import "time"

// UserRegisteredEvent is published when a new user registers.
type UserRegisteredEvent struct {
    UserID    string
    Username  string
    Timestamp time.Time
}

// UserLoggedInEvent is published when a user logs in.
type UserLoggedInEvent struct {
    UserID    string
    Timestamp time.Time
}

// Event types as constants
const (
    UserRegisteredEventType = "user.registered"
    UserLoggedInEventType   = "user.logged_in"
)
```

## Application Layer

**Purpose**: Contains use cases and application-specific logic, orchestrates domain objects.

**Characteristics**:
- Depends on domain layer
- Contains application services (use cases)
- Handles cross-cutting concerns (logging, transactions)
- Publishes domain events
- Contains DTOs for external communication

**Example from Auth Module**:

```go
// modules/auth/application/auth_service.go
package application

import (
    "context"
    "fmt"

    "acacia/core/events"
    "acacia/modules/auth/domain"
)

// AuthService provides use cases for authentication and user management.
type AuthService struct {
    UserRepo   domain.AuthPersistenceRepository // Domain repository
    JwtService JWTService                       // Infrastructure service
    EventBus   events.Bus                       // Infrastructure (event publishing)
}

// RegisterUser registers a new user.
func (s *AuthService) RegisterUser(ctx context.Context, id, username, password string, roles []string) (*domain.User, error) {
    // Check if username already exists
    existingUser, err := s.UserRepo.GetUserByUsername(ctx, username)
    if existingUser != nil {
        return nil, fmt.Errorf("username %s already exists", username)
    }
    if err != nil && err.Error() != "user not found" {
        return nil, fmt.Errorf("failed to check existing username: %w", err)
    }

    // Hash the password securely before saving
    newUser, err := domain.NewUser(id, username, password, roles)
    if err != nil {
        return nil, fmt.Errorf("failed to create new user: %w", err)
    }

    if err := s.UserRepo.SaveUser(ctx, newUser); err != nil {
        return nil, fmt.Errorf("failed to save new user: %w", err)
    }

    // Publish domain event
    if s.EventBus != nil {
        event := NewUserRegisteredEvent(newUser)
        s.EventBus.Publish(ctx, UserRegisteredEventType, event)
    }

    return newUser, nil
}
```

### Application Services vs Domain Services

**Application Services** (Use Cases):
- Orchestrate domain objects
- Handle transactions
- Publish domain events
- Interface with external systems
- May have side effects

**Domain Services**:
- Pure business logic
- No side effects
- Operate on domain objects
- May span multiple entities

## Infrastructure Layer

**Purpose**: Implements interfaces defined in the domain layer, handles external concerns.

**Characteristics**:
- Contains repository implementations
- Handles external APIs and databases
- Contains HTTP/gateway adapters
- Implements event publishing
- May contain external service clients

**Example Repository Implementation**:

```go
// modules/auth/infrastructure/inmemory_user_repository.go
package infrastructure

import (
    "context"
    "errors"
    "sync"

    "acacia/modules/auth/domain"
)

// InMemoryUserRepository is an in-memory implementation of AuthPersistenceRepository.
type InMemoryUserRepository struct {
    mu    sync.RWMutex
    users map[string]*domain.User // username -> user
}

// NewInMemoryUserRepository creates a new in-memory user repository.
func NewInMemoryUserRepository() domain.AuthPersistenceRepository {
    return &InMemoryUserRepository{
        users: make(map[string]*domain.User),
    }
}

// SaveUser saves a user to the repository.
func (r *InMemoryUserRepository) SaveUser(ctx context.Context, user *domain.User) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.users[user.Username] = user
    return nil
}

// GetUserByUsername retrieves a user by username.
func (r *InMemoryUserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    user, exists := r.users[username]
    if !exists {
        return nil, errors.New("user not found")
    }
    return user, nil
}

// Other methods...
```

## Module Structure and Dependencies

### Clean Architecture Dependencies

```
┌─────────────────┐
│   External      │  ← HTTP, Database, Message Queue
├─────────────────┤
│ Infrastructure  │  ← Repository implementations, HTTP adapters
├─────────────────┤
│  Application    │  ← Use cases, Application services
├─────────────────┤
│    Domain       │  ← Business logic, Entities, Repository interfaces
└─────────────────┘
```

**Dependency Rule**: Inner layers should not depend on outer layers.

### Module Dependencies

```go
// modules/mymodule/mymodule.go
type MyModule struct {
    // Domain services
    domainService *domain.MyDomainService

    // Application services
    appService *application.MyAppService

    // Infrastructure
    repository infrastructure.MyRepository
    eventBus   events.Bus
    registry   registry.Registry
}
```

## Configuration and Dependency Injection

### Configuration Pattern

```go
// modules/auth/auth.go
type AuthModuleConfig struct {
    Enabled         bool   `json:"enabled"`
    PersistenceType string `json:"persistenceType"` // e.g., "in-memory", "sqlite"
}

// Configure method
func (m *AuthModule) Configure(cfg interface{}) error {
    authConfigMap, ok := cfg.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid configuration type for AuthModule")
    }

    if enabled, ok := authConfigMap["enabled"].(bool); ok {
        m.Config.Enabled = enabled
    }

    if persistenceType, ok := authConfigMap["persistenceType"].(string); ok {
        m.Config.PersistenceType = persistenceType
    }

    return nil
}
```

### Dependency Resolution in OnReady

```go
// modules/auth/auth.go - OnReady method
func (m *AuthModule) OnReady(ctx context.Context) error {
    var userRepo domain.AuthPersistenceRepository

    switch m.Config.PersistenceType {
    case "in-memory":
        userRepo = infrastructure.NewInMemoryUserRepository()
    case "sqlite":
        // Get service from registry (external module)
        service, err := m.registry.GetService(ctx, "db-lite.Repository")
        if err != nil {
            return fmt.Errorf("failed to get db-lite.Repository service: %w", err)
        }
        dbliteRepo, ok := service.(domain.AuthPersistenceRepository)
        if !ok {
            return fmt.Errorf("db-lite.Repository service is not compatible")
        }
        userRepo = dbliteRepo
    }

    // Inject dependency
    m.authService.UserRepo = userRepo
    return nil
}
```

## Domain Events and Event-Driven Architecture

### Event Publishing Pattern

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

### Event Handler Pattern

```go
// modules/auth/application/events.go
package application

import (
    "acacia/core/events"
    "acacia/modules/auth/domain"
    "context"
    "fmt"
)

// UserEventHandler handles user-related domain events.
type UserEventHandler struct {
    // dependencies
}

// HandleUserRegistered handles user registration events.
func (h *UserEventHandler) HandleUserRegistered(ctx context.Context, event UserRegisteredEvent) error {
    fmt.Printf("User registered: %s\n", event.Username)
    // Send welcome email, create audit log, etc.
    return nil
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

## Testing Domain-Driven Modules

### Testing Domain Layer (Unit Tests)

```go
func TestUser_VerifyPassword(t *testing.T) {
    user, err := domain.NewUser("id", "username", "password", []string{"user"})
    assert.NoError(t, err)

    assert.True(t, user.VerifyPassword("password"))
    assert.False(t, user.VerifyPassword("wrongpassword"))
}
```

### Testing Application Layer (Integration Tests)

```go
func TestAuthService_RegisterUser(t *testing.T) {
    // Mock repository
    repo := &MockUserRepository{}
    eventBus := &MockEventBus{}

    // Create service
    service := &AuthService{
        UserRepo: repo,
        EventBus: eventBus,
    }

    // Test registration
    user, err := service.RegisterUser(context.Background(), "id", "username", "password", []string{"user"})
    assert.NoError(t, err)
    assert.NotNil(t, user)

    // Verify event was published
    assert.True(t, eventBus.EventPublished)
}
```

### Testing Infrastructure Layer

```go
func TestInMemoryUserRepository(t *testing.T) {
    repo := infrastructure.NewInMemoryUserRepository()

    user := &domain.User{ /* ... */ }

    err := repo.SaveUser(context.Background(), user)
    assert.NoError(t, err)

    retrieved, err := repo.GetUserByUsername(context.Background(), user.Username)
    assert.NoError(t, err)
    assert.Equal(t, user.Username, retrieved.Username)
}
```

## Best Practices

### 1. Keep Domain Layer Pure

```go
// ✅ Good: Pure domain logic
func (u *User) CanChangePassword() bool {
    return time.Since(u.LastPasswordChange) > 24*time.Hour
}

// ❌ Bad: Infrastructure concern in domain
func (u *User) CanChangePassword(db *sql.DB) bool {
    // Database access in domain layer
    return false
}
```

### 2. Use Repository Pattern for Data Access

```go
// ✅ Good: Repository interface in domain
type UserRepository interface {
    Save(user *User) error
    FindByID(id string) (*User, error)
}

// ❌ Bad: Direct database access
func (u *User) SaveToDatabase(db *sql.DB) error {
    // Direct DB access in domain
    return nil
}
```

### 3. Application Services as Orchestrators

```go
// ✅ Good: Application service orchestrates domain objects
func (s *AuthService) ChangePassword(ctx context.Context, userID, newPassword string) error {
    user, err := s.UserRepo.FindByID(ctx, userID)
    if err != nil {
        return err
    }

    if !user.CanChangePassword() {
        return errors.New("cannot change password yet")
    }

    user.ChangePassword(newPassword)
    return s.UserRepo.Save(ctx, user)
}
```

### 4. Value Objects for Domain Concepts

```go
// Value object example
type Email struct {
    address string
}

func NewEmail(addr string) (Email, error) {
    if !isValidEmail(addr) {
        return Email{}, errors.New("invalid email")
    }
    return Email{address: addr}, nil
}

func (e Email) String() string {
    return e.address
}
```

## Common Patterns and Anti-Patterns

### ✅ Good Patterns

1. **Rich Domain Model** - Put business logic in domain entities
2. **Dependency Inversion** - Domain defines interfaces, infrastructure implements them
3. **Event-Driven** - Use domain events to decouple components
4. **Factory Methods** - Use constructors that enforce invariants
5. **Immutable Value Objects** - Use value objects for domain concepts

### ❌ Anti-Patterns

1. **Anemic Domain Model** - Business logic in services instead of entities
2. **Repository Implementation in Domain** - Concrete database code in domain layer
3. **God Objects** - Large objects with too many responsibilities
4. **Primitive Obsession** - Using primitives instead of domain-specific types
5. **Layer Skipping** - Infrastructure accessing domain directly

## Module Template

Here's a complete template for a domain-driven module:

```go
// modules/mymodule/mymodule.go
package main

import (
    "acacia/core/events"
    "acacia/core/kernel"
    "acacia/core/registry"
    "acacia/modules/mymodule/application"
    "acacia/modules/mymodule/domain"
    "acacia/modules/mymodule/infrastructure"
    "context"
    "github.com/mitchellh/mapstructure"
)

type MyModuleConfig struct {
    Enabled bool `mapstructure:"enabled"`
}

type MyModule struct {
    name     string
    config   MyModuleConfig
    registry registry.Registry
    eventBus events.Bus

    // Domain
    domainService *domain.MyDomainService

    // Application
    appService *application.MyAppService

    // Infrastructure
    repository infrastructure.MyRepository
}

func NewModule() kernel.Module {
    return &MyModule{
        name: "mymodule",
    }
}

func (m *MyModule) Name() string { return m.name }
func (m *MyModule) Version() string { return "1.0.0" }
func (m *MyModule) Dependencies() map[string]string { return map[string]string{} }

func (m *MyModule) Configure(cfg interface{}) error {
    return mapstructure.Decode(cfg, &m.config)
}

func (m *MyModule) OnLoad(ctx context.Context) error {
    // Initialize domain and application services
    m.domainService = domain.NewMyDomainService()
    m.appService = application.NewMyAppService(m.domainService)
    m.repository = infrastructure.NewMyRepository()

    return nil
}

func (m *MyModule) OnReady(ctx context.Context) error {
    // Inject dependencies
    m.appService.SetRepository(m.repository)
    return nil
}

func (m *MyModule) Start(ctx context.Context) error {
    // Subscribe to events if needed
    return nil
}

func (m *MyModule) RegisterServices(reg registry.Registry) error {
    return reg.RegisterService("myservice", m.appService, m.Name())
}

func (m *MyModule) Stop(ctx context.Context) error {
    return nil
}

func (m *MyModule) ShutdownTimeout() time.Duration {
    return time.Second * 5
}
```

This domain-driven architecture ensures that your modules are maintainable, testable, and follow clean architecture principles while integrating seamlessly with the Acacia framework.
