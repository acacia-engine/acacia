package auth

import "sync"

// ConfigRBACProvider implements the RBACProvider interface using roles defined in the application config.
type ConfigRBACProvider struct {
	mu    sync.RWMutex
	roles map[string]*Role
}

// NewConfigRBACProvider creates a new RBAC provider based on the auth configuration.
func NewConfigRBACProvider(roles []Role) RBACProvider {
	roleMap := make(map[string]*Role, len(roles))
	for i := range roles {
		role := roles[i]
		roleMap[role.Name] = &role
	}
	return &ConfigRBACProvider{
		roles: roleMap,
	}
}

// GetRole retrieves a role by name from the provider's internal map.
func (p *ConfigRBACProvider) GetRole(name string) (*Role, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	role, ok := p.roles[name]
	return role, ok
}
