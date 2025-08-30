package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// PermissionManagerImpl manages plugin permissions
type PermissionManagerImpl struct {
	mu           sync.RWMutex
	permissions  map[string][]string // plugin -> permissions
	activeGrants map[string]*PermissionGrant
	auditLogger  AuditLogger
}

// PermissionGrant represents an active permission grant
type PermissionGrant struct {
	PluginName  string
	Permissions []string
	GrantedBy   string
	GrantedAt   time.Time
	ExpiresAt   *time.Time
	Reason      string
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(auditLogger AuditLogger) *PermissionManagerImpl {
	return &PermissionManagerImpl{
		permissions:  make(map[string][]string),
		activeGrants: make(map[string]*PermissionGrant),
		auditLogger:  auditLogger,
	}
}

// GrantPermissions grants permissions to a plugin
func (pm *PermissionManagerImpl) GrantPermissions(ctx context.Context, pluginName string, permissions []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Validate permissions
	for _, perm := range permissions {
		if err := pm.validatePermissionFormat(perm); err != nil {
			return fmt.Errorf("invalid permission '%s': %w", perm, err)
		}
	}

	// Get principal for audit
	principal := "system"
	if p := getPrincipalFromContext(ctx); p != nil {
		principal = p.ID()
	}

	// Create permission grant
	grant := &PermissionGrant{
		PluginName:  pluginName,
		Permissions: make([]string, len(permissions)),
		GrantedBy:   principal,
		GrantedAt:   time.Now(),
		Reason:      "plugin load",
	}
	copy(grant.Permissions, permissions)

	// Store permissions and grant
	pm.permissions[pluginName] = permissions
	pm.activeGrants[pluginName] = grant

	// Log security event
	if pm.auditLogger != nil {
		event := SecurityEventData{
			EventType:   EventPermissionGranted,
			PluginName:  pluginName,
			Principal:   principal,
			Permissions: permissions,
			Reason:      "plugin load",
			Timestamp:   time.Now(),
		}
		pm.auditLogger.LogEvent(event)
	}

	return nil
}

// RevokePermissions revokes all permissions for a plugin
func (pm *PermissionManagerImpl) RevokePermissions(ctx context.Context, pluginName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Get current permissions for audit
	oldPermissions := pm.permissions[pluginName]

	// Remove permissions and grant
	delete(pm.permissions, pluginName)
	delete(pm.activeGrants, pluginName)

	// Log security event
	if pm.auditLogger != nil && len(oldPermissions) > 0 {
		principal := "system"
		if p := getPrincipalFromContext(ctx); p != nil {
			principal = p.ID()
		}

		event := SecurityEventData{
			EventType:   EventPermissionRevoked,
			PluginName:  pluginName,
			Principal:   principal,
			Permissions: oldPermissions,
			Reason:      "plugin unload",
			Timestamp:   time.Now(),
		}
		pm.auditLogger.LogEvent(event)
	}

	return nil
}

// GetEffectivePermissions returns the effective permissions for a plugin
func (pm *PermissionManagerImpl) GetEffectivePermissions(ctx context.Context, pluginName string) ([]string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	permissions, exists := pm.permissions[pluginName]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to prevent external modification
	result := make([]string, len(permissions))
	copy(result, permissions)

	return result, nil
}

// HasPermission checks if a plugin has a specific permission
func (pm *PermissionManagerImpl) HasPermission(ctx context.Context, pluginName, permission string) (bool, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pluginPerms, exists := pm.permissions[pluginName]
	if !exists {
		return false, nil
	}

	// Check for exact match
	for _, p := range pluginPerms {
		if p == permission {
			return true, nil
		}

		// Check for wildcard match
		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(permission, prefix) {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetAllPermissions returns all plugin permissions
func (pm *PermissionManagerImpl) GetAllPermissions() map[string][]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string][]string)
	for plugin, perms := range pm.permissions {
		result[plugin] = append([]string(nil), perms...)
	}
	return result
}

// GetActiveGrants returns all active permission grants
func (pm *PermissionManagerImpl) GetActiveGrants() map[string]*PermissionGrant {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*PermissionGrant)
	for plugin, grant := range pm.activeGrants {
		grantCopy := *grant // Create a copy
		result[plugin] = &grantCopy
	}
	return result
}

// ValidatePermissionRequest validates a permission request
func (pm *PermissionManagerImpl) ValidatePermissionRequest(ctx context.Context, pluginName, permission string) error {
	hasPermission, err := pm.HasPermission(ctx, pluginName, permission)
	if err != nil {
		return err
	}

	if !hasPermission {
		// Log security violation
		if pm.auditLogger != nil {
			principal := "unknown"
			if p := getPrincipalFromContext(ctx); p != nil {
				principal = p.ID()
			}

			event := SecurityEventData{
				EventType:   EventPermissionDenied,
				PluginName:  pluginName,
				Principal:   principal,
				Permissions: []string{permission},
				Reason:      "permission denied",
				Timestamp:   time.Now(),
			}
			pm.auditLogger.LogEvent(event)
		}

		return NewSecurityError(ErrCodePermissionDenied,
			fmt.Sprintf("permission denied: %s", permission), pluginName)
	}

	return nil
}

// ExpandWildcardPermissions expands wildcard permissions to specific permissions
func (pm *PermissionManagerImpl) ExpandWildcardPermissions(ctx context.Context, pluginName string, requestedPerm string) ([]string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pluginPerms, exists := pm.permissions[pluginName]
	if !exists {
		return []string{}, nil
	}

	var expanded []string

	for _, p := range pluginPerms {
		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(requestedPerm, prefix) {
				// Add the specific permission
				expanded = append(expanded, requestedPerm)
			}
		} else if p == requestedPerm {
			expanded = append(expanded, requestedPerm)
		}
	}

	return expanded, nil
}

// GetPermissionSummary returns a summary of plugin permissions
func (pm *PermissionManagerImpl) GetPermissionSummary() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	summary := map[string]interface{}{
		"total_plugins": len(pm.permissions),
		"total_grants":  len(pm.activeGrants),
	}

	// Count permissions by type
	permCounts := make(map[string]int)
	for _, perms := range pm.permissions {
		for _, perm := range perms {
			// Extract permission type (first part)
			parts := strings.Split(perm, ".")
			if len(parts) > 0 {
				permCounts[parts[0]]++
			}
		}
	}
	summary["permission_types"] = permCounts

	return summary
}

// validatePermissionFormat validates permission string format
func (pm *PermissionManagerImpl) validatePermissionFormat(permission string) error {
	if permission == "" {
		return fmt.Errorf("permission cannot be empty")
	}

	if len(permission) > 255 {
		return fmt.Errorf("permission too long")
	}

	parts := strings.Split(permission, ".")
	if len(parts) < 2 {
		return fmt.Errorf("permission must have at least domain.action format")
	}

	// Check for dangerous patterns
	if strings.Contains(permission, "..") {
		return fmt.Errorf("permission cannot contain consecutive dots")
	}

	// Check wildcard usage
	if strings.Contains(permission, "*") {
		if !strings.HasSuffix(permission, ".*") {
			return fmt.Errorf("wildcard (*) can only be at the end preceded by a dot")
		}
		if strings.Count(permission, "*") > 1 {
			return fmt.Errorf("permission can contain only one wildcard")
		}
	}

	// Validate characters
	for _, char := range permission {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '.' || char == '-' || char == '_' || char == '*') {
			return fmt.Errorf("permission contains invalid character: %c", char)
		}
	}

	return nil
}

// CleanupExpiredGrants removes expired permission grants
func (pm *PermissionManagerImpl) CleanupExpiredGrants() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	var expiredPlugins []string

	for plugin, grant := range pm.activeGrants {
		if grant.ExpiresAt != nil && grant.ExpiresAt.Before(now) {
			expiredPlugins = append(expiredPlugins, plugin)
		}
	}

	for _, plugin := range expiredPlugins {
		delete(pm.permissions, plugin)
		delete(pm.activeGrants, plugin)

		// Log expiration
		if pm.auditLogger != nil {
			event := SecurityEventData{
				EventType:  EventPermissionRevoked,
				PluginName: plugin,
				Principal:  "system",
				Reason:     "grant expired",
				Timestamp:  now,
			}
			pm.auditLogger.LogEvent(event)
		}
	}
}

// GetPluginsByPermission returns plugins that have a specific permission
func (pm *PermissionManagerImpl) GetPluginsByPermission(permission string) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var plugins []string
	for plugin, perms := range pm.permissions {
		for _, p := range perms {
			if p == permission || (strings.HasSuffix(p, ".*") && strings.HasPrefix(permission, strings.TrimSuffix(p, ".*"))) {
				plugins = append(plugins, plugin)
				break
			}
		}
	}

	sort.Strings(plugins)
	return plugins
}
