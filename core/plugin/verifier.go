package plugin

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// PluginVerifierImpl implements plugin verification
type PluginVerifierImpl struct {
	publicKeyPath string
	publicKey     *rsa.PublicKey
}

// NewPluginVerifier creates a new plugin verifier
func NewPluginVerifier(publicKeyPath string) (*PluginVerifierImpl, error) {
	verifier := &PluginVerifierImpl{
		publicKeyPath: publicKeyPath,
	}

	if publicKeyPath != "" {
		if err := verifier.loadPublicKey(); err != nil {
			return nil, fmt.Errorf("failed to load public key: %w", err)
		}
	}

	return verifier, nil
}

// loadPublicKey loads the public key for signature verification
func (v *PluginVerifierImpl) loadPublicKey() error {
	data, err := os.ReadFile(v.publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an RSA public key")
	}

	v.publicKey = rsaPub
	return nil
}

// VerifySignature verifies a plugin's digital signature
func (v *PluginVerifierImpl) VerifySignature(data []byte, signature string) error {
	if v.publicKey == nil {
		// If no public key is configured, signature verification is skipped
		return nil
	}

	// Decode base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Calculate hash of the data
	hash := sha256.Sum256(data)

	// Verify signature
	err = rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, hash[:], sigBytes)
	if err != nil {
		return NewSecurityError(ErrCodeSignatureInvalid, "signature verification failed", "")
	}

	return nil
}

// CalculateChecksum calculates SHA256 checksum of data
func (v *PluginVerifierImpl) CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// ValidateMetadata validates plugin metadata
func (v *PluginVerifierImpl) ValidateMetadata(metadata *PluginMetadata) error {
	if metadata.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if metadata.Type == "" {
		return fmt.Errorf("plugin type is required")
	}

	if len(metadata.Permissions) == 0 {
		return fmt.Errorf("plugin must have at least one permission")
	}

	// Validate permission format
	for _, perm := range metadata.Permissions {
		if err := v.validatePermission(perm); err != nil {
			return fmt.Errorf("invalid permission '%s': %w", perm, err)
		}
	}

	if metadata.Checksum == "" {
		return fmt.Errorf("plugin checksum is required")
	}

	return nil
}

// validatePermission validates a permission string format
func (v *PluginVerifierImpl) validatePermission(permission string) error {
	if permission == "" {
		return fmt.Errorf("permission cannot be empty")
	}

	// Check for basic format: domain.action.resource
	parts := strings.Split(permission, ".")
	if len(parts) < 2 {
		return fmt.Errorf("permission must have at least domain.action format")
	}

	// Check for dangerous patterns
	if strings.Contains(permission, "..") {
		return fmt.Errorf("permission cannot contain consecutive dots")
	}

	if strings.Contains(permission, "*") && !strings.HasSuffix(permission, ".*") {
		return fmt.Errorf("wildcard (*) can only be at the end preceded by a dot")
	}

	return nil
}

// VerifyPluginIntegrity verifies the integrity of a plugin file
func (v *PluginVerifierImpl) VerifyPluginIntegrity(ctx context.Context, pluginPath, expectedChecksum string) error {
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %w", err)
	}

	actualChecksum := v.CalculateChecksum(data)
	if actualChecksum != expectedChecksum {
		return NewSecurityError(ErrCodeIntegrityCheckFailed,
			"plugin checksum mismatch", pluginPath)
	}

	return nil
}

// SignPlugin signs plugin data (for development/build time)
func (v *PluginVerifierImpl) SignPlugin(data []byte, privateKeyPath string) (string, error) {
	// This would be used during plugin compilation
	// For now, return a placeholder signature
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// ValidatePlugin performs comprehensive plugin validation
func (v *PluginVerifierImpl) ValidatePlugin(ctx context.Context, pluginPath string, metadata *PluginMetadata) error {
	// Validate metadata
	if err := v.ValidateMetadata(metadata); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	// Verify file integrity
	if err := v.VerifyPluginIntegrity(ctx, pluginPath, metadata.Checksum); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	// Verify signature if present
	if metadata.Signature != "" {
		data, err := os.ReadFile(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to read plugin for signature verification: %w", err)
		}

		if err := v.VerifySignature(data, metadata.Signature); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	return nil
}

// SecurityValidator provides additional security validations
type SecurityValidator struct {
	*PluginVerifierImpl
	registry *PluginRegistry
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(verifier *PluginVerifierImpl, registry *PluginRegistry) *SecurityValidator {
	return &SecurityValidator{
		PluginVerifierImpl: verifier,
		registry:           registry,
	}
}

// ValidatePluginSecurity performs complete security validation
func (sv *SecurityValidator) ValidatePluginSecurity(ctx context.Context, pluginPath string) (*PluginMetadata, error) {
	// Extract basic metadata
	metadata, err := sv.registry.extractPluginMetadata(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract plugin metadata: %w", err)
	}

	// Check if plugin is in registry and approved
	if !sv.registry.IsPluginApproved(metadata.Name) {
		return nil, NewSecurityError(ErrCodePluginNotApproved,
			"plugin is not approved in registry", metadata.Name)
	}

	// Get full metadata from registry
	registeredMetadata, exists := sv.registry.GetPlugin(metadata.Name)
	if !exists {
		return nil, NewSecurityError(ErrCodePluginNotApproved,
			"plugin not found in registry", metadata.Name)
	}

	// Use registered metadata for validation
	metadata = registeredMetadata

	// Perform full validation
	if err := sv.ValidatePlugin(ctx, pluginPath, metadata); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %w", err)
	}

	return metadata, nil
}

// CheckPluginDependencies validates plugin dependencies
func (sv *SecurityValidator) CheckPluginDependencies(ctx context.Context, metadata *PluginMetadata) error {
	for _, dep := range metadata.Dependencies {
		if !sv.registry.IsPluginApproved(dep) {
			return NewSecurityError(ErrCodePluginNotApproved,
				fmt.Sprintf("dependency %s is not approved", dep), metadata.Name)
		}
	}
	return nil
}
