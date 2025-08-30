// Package auth provides interfaces and types for access control within the Acacia application.
package auth

import (
	"context"
	"regexp"
)

var (
	// permissionSanitizer is a regex to allow only safe characters in permission components.
	// It allows alphanumeric characters, dots, and hyphens.
	permissionSanitizer = regexp.MustCompile(`[^a-zA-Z0-9.-]+`)
)

// contextKey is an unexported type for context keys.
type contextKey int

const (
	principalContextKey contextKey = iota
)

// PrincipalFromContext retrieves the Principal from the given context.
// Returns nil if no Principal is found.
func PrincipalFromContext(ctx context.Context) Principal {
	if p, ok := ctx.Value(principalContextKey).(Principal); ok {
		return p
	}
	return nil
}

// ContextWithPrincipal returns a new context with the given Principal embedded.
func ContextWithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, p)
}

// Permission defines a specific action that can be performed.
type Permission string

// Role defines a collection of permissions.
type Role struct {
	Name        string
	Permissions []Permission
}

// RBACProvider defines the interface for a source of RBAC rules.
// This allows permissions to be loaded from config, a database, etc.
type RBACProvider interface {
	GetRole(name string) (*Role, bool)
}

// AccessController defines the interface for checking permissions for various sensitive operations.
type AccessController interface {
	// CanLog determines if the given principal is authorized to log.
	CanLog(ctx context.Context, p Principal) bool
	// CanAccessMetrics determines if the given principal is authorized to access/modify metrics.
	CanAccessMetrics(ctx context.Context, p Principal) bool
	// CanPublishEvent determines if the given principal is authorized to publish a specific event type.
	CanPublishEvent(ctx context.Context, p Principal, eventType string) bool
	// CanSubscribeEvent determines if the given principal is authorized to subscribe to a specific event type.
	CanSubscribeEvent(ctx context.Context, p Principal, eventType string) bool
	// CanAccessConfig determines if the given principal is authorized to access a specific configuration key.
	CanAccessConfig(ctx context.Context, p Principal, configKey string) bool
	// CanReloadModule determines if the given principal is authorized to reload a specific module.
	CanReloadModule(ctx context.Context, p Principal, moduleToReload string) bool
	// HasPermission checks if a principal has a specific permission.
	HasPermission(p Principal, perm Permission) bool
}

// DefaultAccessController implements AccessController, allowing all access by default.
// It can be configured with an RBACProvider to enforce rules.
type DefaultAccessController struct {
	rbacProvider RBACProvider
}

// NewDefaultAccessController creates a new DefaultAccessController.
// If the provider is nil, it returns a controller that allows all actions.
func NewDefaultAccessController(provider RBACProvider) AccessController {
	if provider == nil {
		return &allowAllAccessController{}
	}
	return &DefaultAccessController{rbacProvider: provider}
}

// allowAllAccessController is an implementation of AccessController that grants all permissions.
type allowAllAccessController struct{}

func (a *allowAllAccessController) CanLog(ctx context.Context, p Principal) bool { return true }
func (a *allowAllAccessController) CanAccessMetrics(ctx context.Context, p Principal) bool {
	return true
}
func (a *allowAllAccessController) CanPublishEvent(ctx context.Context, p Principal, eventType string) bool {
	return true
}
func (a *allowAllAccessController) CanSubscribeEvent(ctx context.Context, p Principal, eventType string) bool {
	return true
}
func (a *allowAllAccessController) CanAccessConfig(ctx context.Context, p Principal, configKey string) bool {
	return true
}
func (a *allowAllAccessController) CanReloadModule(ctx context.Context, p Principal, moduleToReload string) bool {
	return true
}
func (a *allowAllAccessController) HasPermission(p Principal, perm Permission) bool { return true }

func (d *DefaultAccessController) CanLog(ctx context.Context, p Principal) bool {
	return d.HasPermission(p, "core.log")
}

func (d *DefaultAccessController) CanAccessMetrics(ctx context.Context, p Principal) bool {
	return d.HasPermission(p, "core.metrics.access")
}

func (d *DefaultAccessController) CanPublishEvent(ctx context.Context, p Principal, eventType string) bool {
	// Example of a more granular permission check
	return d.HasPermission(p, Permission("core.events.publish."+sanitizePermissionComponent(eventType)))
}

func (d *DefaultAccessController) CanSubscribeEvent(ctx context.Context, p Principal, eventType string) bool {
	// Example of a more granular permission check
	return d.HasPermission(p, Permission("core.events.subscribe."+sanitizePermissionComponent(eventType)))
}

func (d *DefaultAccessController) CanAccessConfig(ctx context.Context, p Principal, configKey string) bool {
	return d.HasPermission(p, Permission("core.config.access."+sanitizePermissionComponent(configKey)))
}

func (d *DefaultAccessController) CanReloadModule(ctx context.Context, p Principal, moduleToReload string) bool {
	return d.HasPermission(p, Permission("core.module.reload."+sanitizePermissionComponent(moduleToReload)))
}

// sanitizePermissionComponent ensures that dynamic parts of a permission string are safe
// by replacing any disallowed characters with an underscore.
func sanitizePermissionComponent(component string) string {
	return permissionSanitizer.ReplaceAllString(component, "_")
}

// HasPermission checks if the principal has the required permission, either directly or through one of its roles.
// It supports wildcards, where a permission like "core.module.*" grants access to any permission starting with "core.module.".
func (d *DefaultAccessController) HasPermission(p Principal, perm Permission) bool {
	// First check if the principal has the permission directly in its roles
	for _, roleName := range p.Roles() {
		// Check if the role name itself matches the permission (direct role-based access)
		if roleName == string(perm) {
			return true
		}
		// Check for wildcard permissions in role names
		if len(roleName) > 2 && roleName[len(roleName)-2:] == ".*" {
			prefix := roleName[:len(roleName)-2]
			if len(perm) > len(prefix) && string(perm)[:len(prefix)] == prefix {
				return true
			}
		}
	}

	// If no RBAC provider is configured, we rely on the direct role check above
	if d.rbacProvider == nil {
		return false
	}

	// Check permissions from all roles assigned to the principal via RBAC provider
	for _, roleName := range p.Roles() {
		if role, ok := d.rbacProvider.GetRole(roleName); ok {
			for _, rolePerm := range role.Permissions {
				if rolePerm == perm {
					return true
				}
				// Check for wildcard permission
				if len(rolePerm) > 2 && rolePerm[len(rolePerm)-2:] == ".*" {
					prefix := rolePerm[:len(rolePerm)-2]
					if len(perm) > len(prefix) && perm[:len(prefix)] == prefix {
						return true
					}
				}
			}
		}
	}
	return false
}

// Principal represents the entity performing an action (e.g., a user, a module, a gateway).
// It can be stored in the context.
type Principal interface {
	// ID returns a unique identifier for the principal.
	ID() string
	// Type returns the type of the principal (e.g., "user", "module", "gateway").
	Type() string
	// Roles returns a list of role names assigned to the principal.
	Roles() []string
}

// DefaultPrincipal is a simple implementation of Principal for internal components.
type DefaultPrincipal struct {
	id            string
	principalType string
	roles         []string
}

// NewDefaultPrincipal creates a new DefaultPrincipal.
func NewDefaultPrincipal(id, principalType string, roles []string) *DefaultPrincipal {
	return &DefaultPrincipal{
		id:            id,
		principalType: principalType,
		roles:         roles,
	}
}

func (p *DefaultPrincipal) ID() string {
	return p.id
}

func (p *DefaultPrincipal) Type() string {
	return p.principalType
}

func (p *DefaultPrincipal) Roles() []string {
	return p.roles
}
