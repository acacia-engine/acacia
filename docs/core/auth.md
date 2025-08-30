# Auth Module Documentation

## 1. Introduction to the Auth Package
The `auth` package provides core interfaces and types for access control within the Acacia application, specifically focusing on Role-Based Access Control (RBAC). Its primary role is to define how different components (modules, gateways, internal services) can be authorized to perform sensitive operations based on assigned roles and permissions. This package is intended to be a foundational layer for internal kernel operations, allowing a more comprehensive, external authentication/authorization module to build upon it.

## 2. Key Concepts

### 2.1. Permission Type
The `Permission` type is a string that defines a specific action that can be performed within the system (e.g., "core.log", "core.metrics.access", "core.module.reload.my-module"). Permissions are granular and are typically grouped into roles.

### 2.2. Role Struct
The `Role` struct defines a collection of `Permission`s. Roles are named sets of permissions that can be assigned to principals.

**Fields:**
*   `Name string`: The unique name of the role (e.g., "admin", "logger", "metrics-reader").
*   `Permissions []Permission`: A list of permissions associated with this role.

### 2.3. RBACProvider Interface
The `RBACProvider` interface defines the contract for a source of RBAC rules. This allows the kernel to load roles and their associated permissions from various sources, such as application configuration, a database, or an external service.

**Methods:**
*   `GetRole(name string) (*Role, bool)`: Retrieves a `Role` by its name. Returns the role and a boolean indicating if it was found.

### 2.4. ConfigRBACProvider Struct
The `ConfigRBACProvider` is a concrete implementation of the `RBACProvider` interface that loads roles and permissions from the application's configuration. It provides a simple, file-based mechanism for defining RBAC rules.

**Methods:**
*   `NewConfigRBACProvider(roles []Role) RBACProvider`: Creates a new `ConfigRBACProvider` instance, initializing it with a slice of `Role`s typically loaded from configuration.
*   `GetRole(name string) (*Role, bool)`: Retrieves a role by name from the provider's internal map, which is populated during initialization.

### 2.5. Principal Interface
The `Principal` interface represents the entity attempting to perform an action within the system. This could be a user, a module, a gateway, or any other identifiable component. The `Principal` carries information about its identity and the roles it possesses, enabling RBAC decisions. This interface is used by the `logger` and `metrics` packages for internal component authorization checks.

**Methods:**
*   `ID() string`: Returns a unique identifier for the principal (e.g., "user-123", "module-metrics-scraper").
*   `Type() string`: Returns the type of the principal (e.g., "user", "module", "gateway", "internal-service").
*   `Roles() []string`: Returns a list of role names assigned to the principal. These roles are then used by the `AccessController` to determine permissions.

### 2.6. AccessController Interface
The `AccessController` interface defines the contract for checking permissions for various sensitive operations within the kernel. It uses the `Principal`'s roles and an `RBACProvider` to make authorization decisions.

**Methods:**
*   `CanLog(ctx context.Context, p Principal) bool`: Determines if the given `Principal` is authorized to log (checks for `core.log` permission).
*   `CanAccessMetrics(ctx context.Context, p Principal) bool`: Determines if the given `Principal` is authorized to access or modify metrics (checks for `core.metrics.access` permission).
*   `CanPublishEvent(ctx context.Context, p Principal, eventType string) bool`: Determines if the given `Principal` is authorized to publish a specific type of event (checks for `core.events.publish.<eventType>` permission).
*   `CanSubscribeEvent(ctx context.Context, p Principal, eventType string) bool`: Determines if the given `Principal` is authorized to subscribe to a specific type of event (checks for `core.events.subscribe.<eventType>` permission).
*   `CanAccessConfig(ctx context.Context, p Principal, configKey string) bool`: Determines if the given `Principal` is authorized to access a specific configuration key (checks for `core.config.access.<configKey>` permission).
*   `CanReloadModule(ctx context.Context, p Principal, moduleToReload string) bool`: Determines if the given `Principal` is authorized to trigger a reload of a specific module (checks for `core.module.reload.<moduleToReload>` permission).
*   `HasPermission(p Principal, perm Permission) bool`: A generic method to check if a principal has a specific permission, either directly or through one of its assigned roles.

### 2.7. DefaultAccessController Struct
The `DefaultAccessController` is a concrete implementation of the `AccessController` interface. It can be configured with an `RBACProvider` to enforce rules.

**Methods:**
*   `NewDefaultAccessController(provider RBACProvider) AccessController`: Creates a new `DefaultAccessController` instance. If the `provider` is `nil`, it returns an internal `allowAllAccessController` that grants all permissions, aligning with the "allow all" default behavior. Otherwise, it returns a `DefaultAccessController` configured with the provided `RBACProvider`.
*   `CanLog(ctx context.Context, p Principal) bool`: Implements the `CanLog` method, delegating to `HasPermission`.
*   `CanAccessMetrics(ctx context.Context, p Principal) bool`: Implements the `CanAccessMetrics` method, delegating to `HasPermission`.
*   `CanPublishEvent(ctx context.Context, p Principal, eventType string) bool`: Implements the `CanPublishEvent` method, delegating to `HasPermission` with a dynamic permission string.
*   `CanSubscribeEvent(ctx context.Context, p Principal, eventType string) bool`: Implements the `CanSubscribeEvent` method, delegating to `HasPermission` with a dynamic permission string.
*   `CanAccessConfig(ctx context.Context, p Principal, configKey string) bool`: Implements the `CanAccessConfig` method, delegating to `HasPermission` with a dynamic permission string.
*   `CanReloadModule(ctx context.Context, p Principal, moduleToReload string) bool`: Implements the `CanReloadModule` method, delegating to `HasPermission` with a dynamic permission string.
*   `HasPermission(p Principal, perm Permission) bool`: Checks if the principal has the required permission, either directly or through one of its roles. This method now supports **wildcard permissions**, where a permission like `"core.module.*"` grants access to any permission starting with `"core.module."`. It iterates through the principal's roles and their associated permissions, checking for exact matches or wildcard matches.

### 2.8. DefaultPrincipal Struct
A basic implementation of the `Principal` interface, useful for representing internal components or for simple testing scenarios.

**Methods:**
*   `NewDefaultPrincipal(id, principalType string, roles []string) *DefaultPrincipal`: Creates a new `DefaultPrincipal` instance with a given ID, type, and a list of role names.
*   `ID() string`: Returns the ID provided during creation.
*   `Type() string`: Returns the type provided during creation.
*   `Roles() []string`: Returns the list of role names provided during creation.

### 2.9. Permission Sanitization
The `auth` package includes a `sanitizePermissionComponent` function that ensures dynamic parts of a permission string are safe. It uses a regular expression (`[^a-zA-Z0-9.-]+`) to replace any disallowed characters (i.e., anything not alphanumeric, a dot, or a hyphen) with an underscore. This prevents injection of malicious characters into permission strings.

## 3. Usage Example

### Initializing with Config-driven RBAC
To use the RBAC features, you typically define roles and permissions in your application's configuration and then initialize the `AccessController` with a `ConfigRBACProvider`.

**Example `config.yaml` snippet for Auth:**
```yaml
auth:
  roles:
    - name: admin
      permissions:
        - "core.log"
        - "core.metrics.access"
        - "core.events.publish.*" # Wildcard for all event types
        - "core.events.subscribe.*" # Wildcard for subscribing to all event types
        - "core.config.access.*"
        - "core.module.reload.*"
    - name: metrics-reader
      permissions:
        - "core.metrics.access"
        - "core.events.subscribe.kernel.*" # Subscribe to kernel events only
    - name: event-publisher
      permissions:
        - "core.events.publish.user.*" # Can publish user events
        - "core.events.subscribe.system.*" # Can subscribe to system events
    - name: module-reloader
      permissions:
        - "core.module.reload.my-module" # Specific module reload permission
```

**Go Code Example:**
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

func main() {
	// 1. Load application configuration, which includes AuthConfig
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// 2. Create an RBACProvider from the loaded AuthConfig
	rbacProvider := auth.NewConfigRBACProvider(cfg.Auth.Roles)

	// 3. Create an AccessController using the RBACProvider
	accessController := auth.NewDefaultAccessController(rbacProvider)

	// 4. Initialize the kernel with the configured AccessController
	krn := kernel.New(cfg, accessController)

	// Example usage of the access controller with different Principals

	// Principal with 'admin' role
	adminPrincipal := auth.NewDefaultPrincipal("admin-user", "user", []string{"admin"})
	if accessController.CanLog(context.Background(), adminPrincipal) {
		fmt.Printf("%s (type: %s, roles: %v) is authorized to log.\n", adminPrincipal.ID(), adminPrincipal.Type(), adminPrincipal.Roles())
	}
	if accessController.HasPermission(adminPrincipal, "core.module.reload.any-module") {
		fmt.Printf("%s (type: %s, roles: %v) has permission to reload any module.\n", adminPrincipal.ID(), adminPrincipal.Type(), adminPrincipal.Roles())
	}

	// Principal with 'metrics-reader' role
	metricsPrincipal := auth.NewDefaultPrincipal("metrics-service", "module", []string{"metrics-reader"})
	if accessController.CanAccessMetrics(context.Background(), metricsPrincipal) {
		fmt.Printf("%s (type: %s, roles: %v) is authorized to access metrics.\n", metricsPrincipal.ID(), metricsPrincipal.Type(), metricsPrincipal.Roles())
	}
	if !accessController.CanLog(context.Background(), metricsPrincipal) {
		fmt.Printf("%s (type: %s, roles: %v) is NOT authorized to log.\n", metricsPrincipal.ID(), metricsPrincipal.Type(), metricsPrincipal.Roles())
	}

	// Principal with 'module-reloader' role for a specific module
	specificReloaderPrincipal := auth.NewDefaultPrincipal("deploy-script", "internal-service", []string{"module-reloader"})
	if accessController.CanReloadModule(context.Background(), specificReloaderPrincipal, "my-module") {
		fmt.Printf("%s (type: %s, roles: %v) is authorized to reload 'my-module'.\n", specificReloaderPrincipal.ID(), specificReloaderPrincipal.Type(), specificReloaderPrincipal.Roles())
	}
	if !accessController.CanReloadModule(context.Background(), specificReloaderPrincipal, "other-module") {
		fmt.Printf("%s (type: %s, roles: %v) is NOT authorized to reload 'other-module'.\n", specificReloaderPrincipal.ID(), specificReloaderPrincipal.Type(), specificReloaderPrincipal.Roles())
	}

	// Principal with no roles
	guestPrincipal := auth.NewDefaultPrincipal("guest-user", "user", []string{})
	if !accessController.CanLog(context.Background(), guestPrincipal) {
		fmt.Printf("%s (type: %s, roles: %v) is NOT authorized to log.\n", guestPrincipal.ID(), guestPrincipal.Type(), guestPrincipal.Roles())
	}

	// Start and stop kernel (simplified for example)
	if err := krn.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start kernel: %v", err)
	}
	fmt.Println("Kernel started successfully.")
	if err := krn.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop kernel: %v", err)
	}
	fmt.Println("Kernel stopped successfully.")
}

## 4. Security Best Practices and Considerations

This section provides guidelines and best practices for enhancing the security of the authentication and authorization mechanisms within Acacia.

### 4.1. Securing Role Management

The `ConfigRBACProvider` loads roles from the application's configuration file. It is crucial to:
*   **Restrict Access**: Ensure that the configuration file containing RBAC roles and permissions is protected with appropriate file system permissions. Only authorized personnel or automated deployment systems should have read/write access.
*   **Version Control**: Store the configuration file in a secure version control system (e.g., Git) and follow best practices for managing sensitive data in repositories (e.g., using `.gitignore` for local overrides, avoiding hardcoding secrets).
*   **Review Regularly**: Periodically review the defined roles and permissions to ensure they align with the principle of least privilege and do not grant excessive access.

### 4.2. Authentication Integration

The `auth` package focuses on authorization. A secure and reliable authentication process is paramount for the overall effectiveness of the authorization system.
*   **Secure Authentication Layer**: Implement a robust authentication layer (e.g., OAuth2, JWT, session-based authentication) that securely verifies the identity of principals before they interact with the authorization system.
*   **Immutable Principal**: Once a `Principal` object is created and authenticated, its identity and roles should be considered immutable. Any changes to a principal's roles or permissions should necessitate re-authentication or re-issuance of the principal object. This prevents privilege escalation or unauthorized changes during a session.

### 4.3. Event Subscription Security
The `auth` package provides fine-grained control over event subscriptions to prevent unauthorized access to sensitive event streams.

**Event Subscription Permissions:**
*   `core.events.subscribe.<eventType>`: Grants permission to subscribe to a specific event type
*   `core.events.subscribe.*`: Grants permission to subscribe to all event types (wildcard)
*   `core.events.subscribe.system.*`: Grants permission to subscribe to all system events
*   `core.events.subscribe.user.*`: Grants permission to subscribe to all user-related events

**Example Security Configurations:**
```yaml
auth:
  roles:
    - name: audit-service
      permissions:
        - "core.events.subscribe.user.*"        # Monitor user activities
        - "core.events.subscribe.auth.*"        # Monitor authentication events
    - name: metrics-collector
      permissions:
        - "core.events.subscribe.kernel.*"      # System metrics only
        - "core.events.subscribe.metrics.*"     # Performance metrics
    - name: security-monitor
      permissions:
        - "core.events.subscribe.*"             # All events for security analysis
```

**Implementation:**
```go
// Check subscription authorization before subscribing
if accessController.CanSubscribeEvent(ctx, principal, "user.login") {
    eventBus.Subscribe("user.login") // Allowed
} else {
    return fmt.Errorf("unauthorized to subscribe to user.login events")
}
```

### 4.4. Context Propagation of Principal

Consistent and secure propagation of the `Principal` through the `context.Context` is vital for accurate authorization decisions.
*   **Middleware for Principal Injection**: For web applications or API services, implement middleware that extracts authentication information (e.g., from headers, cookies) and injects the `Principal` object into the `context.Context` at the beginning of a request lifecycle.
*   **Consistent Retrieval**: Ensure that all authorization checks retrieve the `Principal` from the `context.Context` using a consistent and secure method. Avoid passing `Principal` objects directly as function arguments unless absolutely necessary, as this can lead to inconsistencies.
*   **Example of Context Propagation**:
    ```go
    package main

    import (
    	"context"
    	"acacia/core/auth"
    )

    // contextKey is a private type to prevent collisions with other context keys.
    type contextKey string

    const principalContextKey contextKey = "principal"

    // WithPrincipal returns a new context with the given Principal.
    func WithPrincipal(ctx context.Context, p auth.Principal) context.Context {
    	return context.WithValue(ctx, principalContextKey, p)
    }

    // PrincipalFromContext retrieves the Principal from the context.
    // Returns nil if no Principal is found.
    func PrincipalFromContext(ctx context.Context) auth.Principal {
    	if p, ok := ctx.Value(principalContextKey).(auth.Principal); ok {
    		return p
    	}
    	return nil
    }

    // Example usage in a handler or service:
    func MyAuthorizedOperation(ctx context.Context) error {
    	p := PrincipalFromContext(ctx)
    	if p == nil {
    		return fmt.Errorf("unauthenticated access")
    	}

    	// Now use 'p' for authorization checks
    	// if !accessController.CanDoSomething(ctx, p) { ... }
    	return nil
    }
    ```
