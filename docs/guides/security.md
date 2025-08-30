# Security and Authentication Patterns

This guide covers security patterns and authentication integration for Acacia module developers, based on the core auth system and real module implementations.

## Principal-Based Security

### The Principal Interface

Every security context in Acacia is based on the `Principal` interface:

```go
type Principal interface {
    ID() string      // Unique identifier
    Type() string    // "user", "module", "gateway", etc.
    Roles() []string // Assigned roles
}
```

### Implementing Principal in Domain Models

Your domain models can implement `Principal` directly:

```go
type Customer struct {
    customerID string
    email      string
    userRoles  []string
}

// Implement core/auth.Principal interface
func (c *Customer) ID() string {
    return c.customerID
}

func (c *Customer) Type() string {
    return "customer"
}

func (c *Customer) Roles() []string {
    return c.userRoles
}
```

### Module Principals

Modules can create principals for accessing other services:

```go
// In your module's Start() method
principal := auth.NewDefaultPrincipal(
    "mymodule",
    "module",
    []string{"core.*", "gateway.httpapi.access"},
)

gatewayCtx := auth.ContextWithPrincipal(ctx, principal)
// Now use gatewayCtx to access gateway services
```

## Access Control Integration

### Using AccessController

Modules receive an `AccessController` through the kernel and can use it to check permissions:

```go
type MyModule struct {
    accessController auth.AccessController
    // ... other fields
}

func (m *MyModule) SetAccessController(ac auth.AccessController) {
    m.accessController = ac
}
```

### Permission Checking

Check permissions before sensitive operations:

```go
func (m *MyModule) performSensitiveOperation(ctx context.Context, principal auth.Principal) error {
    if !m.accessController.CanAccessConfig(ctx, principal, "myconfig.secret") {
        return fmt.Errorf("access denied to sensitive configuration")
    }

    if !m.accessController.HasPermission(principal, "module.mymodule.sensitive") {
        return fmt.Errorf("insufficient permissions")
    }

    // Proceed with operation
    return nil
}
```

## Context Propagation

### Principal Context

Always propagate principals through context:

```go
// When calling other services
principal := auth.PrincipalFromContext(ctx)
if principal == nil {
    return fmt.Errorf("no principal in context")
}

// Create context with principal for service calls
serviceCtx := auth.ContextWithPrincipal(ctx, principal)
result, err := otherService.DoSomething(serviceCtx)
```

### Gateway Access with Principal

```go
// From modules/auth/auth.go - Accessing gateway with proper principal
principal := auth.NewDefaultPrincipal("auth", "module", []string{"core.*", "gateway.httpapi.access"})
gatewayCtx := auth.ContextWithPrincipal(ctx, principal)

gateway, err := m.registry.GetGateway(gatewayCtx, "httpapi")
if err != nil {
    return fmt.Errorf("failed to get httpapi gateway: %w", err)
}
```

## RBAC Configuration

### Role Definition

Roles are collections of permissions:

```go
type Role struct {
    Name        string
    Permissions []Permission
}
```

### Permission Patterns

Use hierarchical permissions with wildcards:

```go
// Wildcard permissions
"core.*"                    // All core permissions
"core.config.access.*"      // All config access
"core.module.reload.auth"   // Specific module reload

// Dynamic permissions
Permission("core.events.publish." + eventType)
Permission("core.config.access." + configKey)
```

### Role-Based Access Control

```go
func (d *DefaultAccessController) HasPermission(p Principal, perm Permission) bool {
    // Check if principal has permission through roles
    for _, roleName := range p.Roles() {
        if role, ok := d.rbacProvider.GetRole(roleName); ok {
            for _, rolePerm := range role.Permissions {
                if rolePerm == perm {
                    return true
                }
                // Check wildcards
                if strings.HasSuffix(string(rolePerm), ".*") {
                    prefix := strings.TrimSuffix(string(rolePerm), ".*")
                    if strings.HasPrefix(string(perm), prefix) {
                        return true
                    }
                }
            }
        }
    }
    return false
}
```

## Security Best Practices

### Least Privilege Principle

```go
// Bad: Too broad permissions
principal := auth.NewDefaultPrincipal("mymodule", "module", []string{"*"})

// Good: Specific permissions only
principal := auth.NewDefaultPrincipal("mymodule", "module", []string{
    "gateway.httpapi.register",  // Only what we need
    "registry.service.read",
})
```

### Secure Defaults

```go
// In Configure method
type MyModuleConfig struct {
    Enabled      bool     `json:"enabled"`
    AllowedRoles []string `json:"allowedRoles"`
    AdminOnly    bool     `json:"adminOnly"`
}

func (m *MyModule) Configure(cfg interface{}) error {
    // Start with secure defaults
    m.config = MyModuleConfig{
        Enabled:      false,  // Disabled by default
        AdminOnly:    true,   // Admin only by default
        AllowedRoles: []string{"admin"},
    }

    // Override with provided config
    if err := mapstructure.Decode(cfg, &m.config); err != nil {
        return fmt.Errorf("failed to decode config: %w", err)
    }

    return nil
}
```

### Input Validation

```go
func (s *AuthService) RegisterUser(ctx context.Context, id, username, password string, roles []string) (*User, error) {
    // Validate inputs
    if username == "" {
        return nil, fmt.Errorf("username cannot be empty")
    }
    if len(password) < 8 {
        return nil, fmt.Errorf("password too weak")
    }

    // Sanitize roles
    for _, role := range roles {
        if !isValidRole(role) {
            return nil, fmt.Errorf("invalid role: %s", role)
        }
    }

    // Proceed with registration...
}
```

### Error Handling

```go
// Don't leak sensitive information
func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
    user, err := s.UserRepo.GetUserByUsername(ctx, username)
    if err != nil {
        // Don't reveal if user exists or not
        return "", fmt.Errorf("authentication failed")
    }

    if !user.VerifyPassword(password) {
        return "", fmt.Errorf("authentication failed")
    }

    // Success case...
}
```

## Common Security Patterns

### Service Authorization

```go
func (m *MyModule) RegisterServices(reg registry.Registry) error {
    // Wrap service with authorization
    authorizedService := &AuthorizedMyService{
        service: m.myService,
        accessController: m.accessController,
    }

    return reg.RegisterService("myservice", authorizedService, m.Name())
}

type AuthorizedMyService struct {
    service          *MyService
    accessController auth.AccessController
}

func (a *AuthorizedMyService) DoSomething(ctx context.Context, param string) error {
    principal := auth.PrincipalFromContext(ctx)
    if principal == nil {
        return fmt.Errorf("no principal in context")
    }

    if !a.accessController.HasPermission(principal, "service.myservice.use") {
        return fmt.Errorf("access denied")
    }

    return a.service.DoSomething(ctx, param)
}
```

### Event Authorization

```go
func (m *MyModule) publishEvent(ctx context.Context, eventType string, event interface{}) error {
    principal := auth.PrincipalFromContext(ctx)
    if principal != nil {
        if !m.accessController.CanPublishEvent(ctx, principal, eventType) {
            return fmt.Errorf("not authorized to publish event: %s", eventType)
        }
    }

    m.eventBus.Publish(ctx, eventType, event)
    return nil
}
```

## Testing Security

### Mock Principals for Testing

```go
func createTestPrincipal(id, ptype string, roles []string) auth.Principal {
    return auth.NewDefaultPrincipal(id, ptype, roles)
}

func TestSecureOperation(t *testing.T) {
    // Test with insufficient permissions
    userPrincipal := createTestPrincipal("user1", "user", []string{"user"})

    module := &MyModule{accessController: auth.NewDefaultAccessController(nil)}
    ctx := auth.ContextWithPrincipal(context.Background(), userPrincipal)

    err := module.performSensitiveOperation(ctx, userPrincipal)
    assert.Error(t, err) // Should fail

    // Test with sufficient permissions
    adminPrincipal := createTestPrincipal("admin1", "user", []string{"admin"})
    ctx = auth.ContextWithPrincipal(context.Background(), adminPrincipal)

    err = module.performSensitiveOperation(ctx, adminPrincipal)
    assert.NoError(t, err) // Should succeed
}
```

## Integration Examples

### HTTP Handler with Authentication

```go
func (m *MyModule) registerHTTPHandlers(gateway interface{}) error {
    httpGateway := gateway.(interface{
        RegisterRoute(method, path string, handler http.HandlerFunc) error
    })

    // Protected endpoint
    return httpGateway.RegisterRoute("GET", "/protected", func(w http.ResponseWriter, r *http.Request) {
        // Extract principal from request context (set by auth middleware)
        principal := auth.PrincipalFromContext(r.Context())
        if principal == nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Check permissions
        if !m.accessController.HasPermission(principal, "module.mymodule.access") {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }

        // Handle request...
    })
}
```

This guide provides the foundation for building secure modules within the Acacia framework. Always follow the principle of least privilege and validate all inputs and permissions.
