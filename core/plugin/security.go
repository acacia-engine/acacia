package plugin

import (
	"acacia/core/auth"
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

// SecurityEvent represents different types of security events
type SecurityEvent string

const (
	EventPluginLoadAttempt SecurityEvent = "plugin.load.attempt"
	EventPluginLoadSuccess SecurityEvent = "plugin.load.success"
	EventPluginLoadFailure SecurityEvent = "plugin.load.failure"
	EventPluginUnload      SecurityEvent = "plugin.unload"
	EventSecurityViolation SecurityEvent = "security.violation"
	EventPermissionGranted SecurityEvent = "permission.granted"
	EventPermissionDenied  SecurityEvent = "permission.denied"
	EventPermissionRevoked SecurityEvent = "permission.revoked"
)

// PluginMetadata contains information about a plugin
type PluginMetadata struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"` // "module", "gateway", "security", etc.
	Version      string    `json:"version"`
	Description  string    `json:"description,omitempty"`
	Permissions  []string  `json:"permissions"` // Required permissions
	Dependencies []string  `json:"dependencies,omitempty"`
	Checksum     string    `json:"checksum"`            // SHA256 of plugin binary
	Signature    string    `json:"signature,omitempty"` // Digital signature
	ApprovedAt   time.Time `json:"approved_at"`
	ApprovedBy   string    `json:"approved_by"`
	Enabled      bool      `json:"enabled"`
}

// SecurityContext provides security information for plugins
type SecurityContext struct {
	Principal      auth.Principal `json:"principal"`
	PluginID       string         `json:"plugin_id"`
	GrantedPerms   []string       `json:"granted_permissions"`
	EffectivePerms []string       `json:"effective_permissions"`
	StartTime      time.Time      `json:"start_time"`
	LastActivity   time.Time      `json:"last_activity"`
}

// SecurityEventData contains data for security events
type SecurityEventData struct {
	EventType   SecurityEvent          `json:"event_type"`
	PluginName  string                 `json:"plugin_name"`
	Principal   string                 `json:"principal"`
	Permissions []string               `json:"permissions,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityManager interface defines security operations
type SecurityManager interface {
	// Plugin validation
	ValidatePlugin(ctx context.Context, pluginPath string, metadata *PluginMetadata) error
	VerifyPluginSignature(ctx context.Context, pluginPath string, signature string) error
	CheckPluginIntegrity(ctx context.Context, pluginPath string, expectedHash string) error

	// Permission management
	GrantPluginPermissions(ctx context.Context, pluginName string, permissions []string) error
	RevokePluginPermissions(ctx context.Context, pluginName string) error
	GetPluginPermissions(ctx context.Context, pluginName string) ([]string, error)

	// Security monitoring
	LogSecurityEvent(ctx context.Context, event SecurityEventData) error
	GetSecurityEvents(ctx context.Context, pluginName string, limit int) ([]SecurityEventData, error)

	// Plugin lifecycle
	OnPluginLoad(ctx context.Context, pluginName string, principal auth.Principal) (*SecurityContext, error)
	OnPluginUnload(ctx context.Context, pluginName string) error

	// Access control
	CheckPermission(ctx context.Context, principal auth.Principal, permission string, resource string) error
}

// PluginVerifier handles plugin verification
type PluginVerifier interface {
	VerifySignature(data []byte, signature string) error
	CalculateChecksum(data []byte) string
	ValidateMetadata(metadata *PluginMetadata) error
}

// AuditLogger handles security event logging
type AuditLogger interface {
	LogEvent(event SecurityEventData) error
	QueryEvents(pluginName string, limit int) ([]SecurityEventData, error)
}

// PermissionManager handles plugin permissions
type PermissionManager interface {
	GrantPermissions(ctx context.Context, pluginName string, permissions []string) error
	RevokePermissions(ctx context.Context, pluginName string) error
	GetEffectivePermissions(ctx context.Context, pluginName string) ([]string, error)
	HasPermission(ctx context.Context, pluginName string, permission string) (bool, error)
}

// SecurityError represents security-related errors
type SecurityError struct {
	Code    string
	Message string
	Plugin  string
	Details map[string]interface{}
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("security error [%s]: %s (plugin: %s)", e.Code, e.Message, e.Plugin)
}

// Common security error codes
const (
	ErrCodePluginNotApproved    = "PLUGIN_NOT_APPROVED"
	ErrCodeSignatureInvalid     = "SIGNATURE_INVALID"
	ErrCodeIntegrityCheckFailed = "INTEGRITY_CHECK_FAILED"
	ErrCodePermissionDenied     = "PERMISSION_DENIED"
	ErrCodePluginDisabled       = "PLUGIN_DISABLED"
	ErrCodeSecurityViolation    = "SECURITY_VIOLATION"
)

// NewSecurityError creates a new security error
func NewSecurityError(code, message, plugin string) *SecurityError {
	return &SecurityError{
		Code:    code,
		Message: message,
		Plugin:  plugin,
		Details: make(map[string]interface{}),
	}
}

// CalculateChecksum calculates SHA256 checksum of data
func CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}
