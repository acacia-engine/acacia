package plugin

import (
	"acacia/core/auth"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SecurityManagerImpl is the main security manager implementation
type SecurityManagerImpl struct {
	registry       *PluginRegistry
	verifier       *PluginVerifierImpl
	validator      *SecurityValidator
	auditLogger    *AuditLoggerImpl
	permissionMgr  *PermissionManagerImpl
	securityConfig *SecurityConfig
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	RegistryPath      string
	AuditLogPath      string
	PublicKeyPath     string
	MaxAuditEvents    int
	AllowUnapproved   bool   // Allow plugins not in registry (less secure)
	DefaultPermission string // Default permission for unknown plugins
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig) (*SecurityManagerImpl, error) {
	if config == nil {
		config = &SecurityConfig{
			RegistryPath:    "./config/plugins.json",
			AuditLogPath:    "./logs/plugin_audit.json",
			MaxAuditEvents:  10000,
			AllowUnapproved: false,
		}
	}

	// Initialize audit logger
	auditLogger, err := NewAuditLogger(config.AuditLogPath, config.MaxAuditEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	// Initialize registry
	registry := NewPluginRegistry(config.RegistryPath)

	// Initialize verifier
	verifier, err := NewPluginVerifier(config.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create verifier: %w", err)
	}

	// Initialize permission manager
	permissionMgr := NewPermissionManager(auditLogger)

	// Initialize security validator
	validator := NewSecurityValidator(verifier, registry)

	manager := &SecurityManagerImpl{
		registry:       registry,
		verifier:       verifier,
		validator:      validator,
		auditLogger:    auditLogger,
		permissionMgr:  permissionMgr,
		securityConfig: config,
	}

	return manager, nil
}

// ValidatePlugin performs comprehensive plugin validation
func (sm *SecurityManagerImpl) ValidatePlugin(ctx context.Context, pluginPath string, metadata *PluginMetadata) error {
	// Log validation attempt
	sm.auditLogger.LogEvent(SecurityEventData{
		EventType:  EventPluginLoadAttempt,
		PluginName: filepath.Base(pluginPath),
		Principal:  getPrincipalID(ctx),
		Reason:     "plugin validation",
		Timestamp:  time.Now(),
	})

	// Step 1: Registry validation
	if !sm.securityConfig.AllowUnapproved {
		if !sm.registry.IsPluginApproved(metadata.Name) {
			sm.auditLogger.LogEvent(SecurityEventData{
				EventType:  EventSecurityViolation,
				PluginName: metadata.Name,
				Principal:  getPrincipalID(ctx),
				Reason:     "plugin not approved in registry",
				Timestamp:  time.Now(),
			})
			return NewSecurityError(ErrCodePluginNotApproved,
				"plugin is not approved in registry", metadata.Name)
		}
	}

	// Step 2: Get approved metadata from registry
	approvedMetadata, exists := sm.registry.GetPlugin(metadata.Name)
	if exists {
		metadata = approvedMetadata
	} else if !sm.securityConfig.AllowUnapproved {
		return NewSecurityError(ErrCodePluginNotApproved,
			"plugin not found in registry", metadata.Name)
	}

	// Step 3: Integrity verification
	if err := sm.verifier.VerifyPluginIntegrity(ctx, pluginPath, metadata.Checksum); err != nil {
		sm.auditLogger.LogEvent(SecurityEventData{
			EventType:  EventSecurityViolation,
			PluginName: metadata.Name,
			Principal:  getPrincipalID(ctx),
			Reason:     "integrity check failed",
			Timestamp:  time.Now(),
		})
		return err
	}

	// Step 4: Signature verification (if signature is present)
	if metadata.Signature != "" {
		data, err := os.ReadFile(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to read plugin for signature verification: %w", err)
		}

		if err := sm.verifier.VerifySignature(data, metadata.Signature); err != nil {
			sm.auditLogger.LogEvent(SecurityEventData{
				EventType:  EventSecurityViolation,
				PluginName: metadata.Name,
				Principal:  getPrincipalID(ctx),
				Reason:     "signature verification failed",
				Timestamp:  time.Now(),
			})
			return err
		}
	}

	// Step 5: Permission validation
	for _, perm := range metadata.Permissions {
		if err := sm.verifier.ValidateMetadata(&PluginMetadata{Permissions: []string{perm}}); err != nil {
			sm.auditLogger.LogEvent(SecurityEventData{
				EventType:  EventSecurityViolation,
				PluginName: metadata.Name,
				Principal:  getPrincipalID(ctx),
				Reason:     fmt.Sprintf("invalid permission: %s", perm),
				Timestamp:  time.Now(),
			})
			return fmt.Errorf("invalid permission '%s': %w", perm, err)
		}
	}

	// Step 6: Dependency validation
	if err := sm.validator.CheckPluginDependencies(ctx, metadata); err != nil {
		sm.auditLogger.LogEvent(SecurityEventData{
			EventType:  EventSecurityViolation,
			PluginName: metadata.Name,
			Principal:  getPrincipalID(ctx),
			Reason:     "dependency validation failed",
			Timestamp:  time.Now(),
		})
		return err
	}

	// Log successful validation
	sm.auditLogger.LogEvent(SecurityEventData{
		EventType:  EventPluginLoadSuccess,
		PluginName: metadata.Name,
		Principal:  getPrincipalID(ctx),
		Reason:     "plugin validation successful",
		Timestamp:  time.Now(),
	})

	return nil
}

// GrantPluginPermissions grants permissions to a plugin
func (sm *SecurityManagerImpl) GrantPluginPermissions(ctx context.Context, pluginName string, permissions []string) error {
	// Validate plugin is approved
	if !sm.securityConfig.AllowUnapproved && !sm.registry.IsPluginApproved(pluginName) {
		return NewSecurityError(ErrCodePluginNotApproved,
			"cannot grant permissions to unapproved plugin", pluginName)
	}

	// Use scoped permissions if available from registry
	if metadata, exists := sm.registry.GetPlugin(pluginName); exists {
		permissions = metadata.Permissions
	}

	return sm.permissionMgr.GrantPermissions(ctx, pluginName, permissions)
}

// RevokePluginPermissions revokes all permissions for a plugin
func (sm *SecurityManagerImpl) RevokePluginPermissions(ctx context.Context, pluginName string) error {
	return sm.permissionMgr.RevokePermissions(ctx, pluginName)
}

// GetPluginPermissions returns the effective permissions for a plugin
func (sm *SecurityManagerImpl) GetPluginPermissions(ctx context.Context, pluginName string) ([]string, error) {
	return sm.permissionMgr.GetEffectivePermissions(ctx, pluginName)
}

// CheckPermission validates if a plugin has a specific permission
func (sm *SecurityManagerImpl) CheckPermission(ctx context.Context, principal auth.Principal, permission string, resource string) error {
	// Extract plugin name from principal (assuming it's embedded)
	pluginName := principal.ID()

	// Validate permission request
	if err := sm.permissionMgr.ValidatePermissionRequest(ctx, pluginName, permission); err != nil {
		return err
	}

	return nil
}

// OnPluginLoad handles plugin load event
func (sm *SecurityManagerImpl) OnPluginLoad(ctx context.Context, pluginName string, principal auth.Principal) (*SecurityContext, error) {
	// Grant permissions
	permissions, err := sm.GetPluginPermissions(ctx, pluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin permissions: %w", err)
	}

	if err := sm.GrantPluginPermissions(ctx, pluginName, permissions); err != nil {
		return nil, fmt.Errorf("failed to grant plugin permissions: %w", err)
	}

	// Create security context
	securityCtx := &SecurityContext{
		Principal:      principal,
		PluginID:       pluginName,
		GrantedPerms:   permissions,
		EffectivePerms: permissions,
		StartTime:      time.Now(),
		LastActivity:   time.Now(),
	}

	return securityCtx, nil
}

// OnPluginUnload handles plugin unload event
func (sm *SecurityManagerImpl) OnPluginUnload(ctx context.Context, pluginName string) error {
	// Revoke permissions
	if err := sm.RevokePluginPermissions(ctx, pluginName); err != nil {
		sm.auditLogger.LogEvent(SecurityEventData{
			EventType:  EventSecurityViolation,
			PluginName: pluginName,
			Principal:  getPrincipalID(ctx),
			Reason:     "failed to revoke permissions during unload",
			Timestamp:  time.Now(),
		})
	}

	// Log unload event
	sm.auditLogger.LogEvent(SecurityEventData{
		EventType:  EventPluginUnload,
		PluginName: pluginName,
		Principal:  getPrincipalID(ctx),
		Reason:     "plugin unloaded",
		Timestamp:  time.Now(),
	})

	return nil
}

// GetSecurityEvents retrieves security events for a plugin
func (sm *SecurityManagerImpl) GetSecurityEvents(ctx context.Context, pluginName string, limit int) ([]SecurityEventData, error) {
	return sm.auditLogger.QueryEvents(pluginName, limit)
}

// GetSecuritySummary returns a security summary
func (sm *SecurityManagerImpl) GetSecuritySummary(ctx context.Context, hours int) (map[string]int, error) {
	return sm.auditLogger.GetSecuritySummary(hours)
}

// RegisterPlugin adds a plugin to the registry
func (sm *SecurityManagerImpl) RegisterPlugin(ctx context.Context, metadata *PluginMetadata) error {
	return sm.registry.RegisterPlugin(ctx, metadata)
}

// ListApprovedPlugins returns all approved plugins
func (sm *SecurityManagerImpl) ListApprovedPlugins() map[string]*PluginMetadata {
	return sm.registry.ListEnabledPlugins()
}

// ScanAndRegisterPlugins scans a directory and registers found plugins
func (sm *SecurityManagerImpl) ScanAndRegisterPlugins(ctx context.Context, dir string) ([]string, error) {
	plugins, err := sm.registry.ScanPluginDirectory(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	var registered []string
	for _, metadata := range plugins {
		// Generate checksum
		pluginPath := filepath.Join(dir, metadata.Name+".so")
		if data, err := os.ReadFile(pluginPath); err == nil {
			metadata.Checksum = sm.verifier.CalculateChecksum(data)
		}

		// Register plugin (disabled by default)
		metadata.Enabled = false
		if err := sm.registry.RegisterPlugin(ctx, metadata); err != nil {
			sm.auditLogger.LogEvent(SecurityEventData{
				EventType:  EventSecurityViolation,
				PluginName: metadata.Name,
				Principal:  getPrincipalID(ctx),
				Reason:     fmt.Sprintf("failed to register plugin: %v", err),
				Timestamp:  time.Now(),
			})
			continue
		}

		registered = append(registered, metadata.Name)
	}

	return registered, nil
}

// EnablePlugin enables a plugin in the registry
func (sm *SecurityManagerImpl) EnablePlugin(ctx context.Context, pluginName string) error {
	return sm.registry.EnablePlugin(ctx, pluginName)
}

// DisablePlugin disables a plugin in the registry
func (sm *SecurityManagerImpl) DisablePlugin(ctx context.Context, pluginName string) error {
	return sm.registry.DisablePlugin(ctx, pluginName)
}

// GetPermissionSummary returns a summary of all plugin permissions
func (sm *SecurityManagerImpl) GetPermissionSummary() map[string]interface{} {
	return sm.permissionMgr.GetPermissionSummary()
}

// Shutdown gracefully shuts down the security manager
func (sm *SecurityManagerImpl) Shutdown(ctx context.Context) error {
	// Shutdown audit logger
	if err := sm.auditLogger.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown audit logger: %w", err)
	}

	return nil
}

// getPrincipalID extracts principal ID from context
func getPrincipalID(ctx context.Context) string {
	if principal := getPrincipalFromContext(ctx); principal != nil {
		return principal.ID()
	}
	return "system"
}
