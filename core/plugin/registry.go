package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PluginRegistry manages the approved plugins and their metadata
type PluginRegistry struct {
	mu           sync.RWMutex
	plugins      map[string]*PluginMetadata // plugin name -> metadata
	registryPath string
	autoReload   bool
	lastModified time.Time
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(registryPath string) *PluginRegistry {
	return &PluginRegistry{
		plugins:      make(map[string]*PluginMetadata),
		registryPath: registryPath,
		autoReload:   true,
	}
}

// LoadRegistry loads the plugin registry from disk
func (r *PluginRegistry) LoadRegistry(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty registry if it doesn't exist
			r.plugins = make(map[string]*PluginMetadata)
			return r.saveRegistryLocked()
		}
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	var registry struct {
		Plugins map[string]*PluginMetadata `json:"plugins"`
		Version string                     `json:"version"`
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return fmt.Errorf("failed to parse registry: %w", err)
	}

	r.plugins = registry.Plugins
	if r.plugins == nil {
		r.plugins = make(map[string]*PluginMetadata)
	}

	return nil
}

// SaveRegistry saves the plugin registry to disk
func (r *PluginRegistry) SaveRegistry() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveRegistryLocked()
}

func (r *PluginRegistry) saveRegistryLocked() error {
	registry := struct {
		Plugins map[string]*PluginMetadata `json:"plugins"`
		Version string                     `json:"version"`
		Updated time.Time                  `json:"updated"`
	}{
		Plugins: r.plugins,
		Version: "1.0",
		Updated: time.Now(),
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.registryPath), 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	if err := os.WriteFile(r.registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// RegisterPlugin adds or updates a plugin in the registry
func (r *PluginRegistry) RegisterPlugin(ctx context.Context, metadata *PluginMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate metadata
	if err := r.validateMetadata(metadata); err != nil {
		return fmt.Errorf("invalid plugin metadata: %w", err)
	}

	// Set approval timestamp if not set
	if metadata.ApprovedAt.IsZero() {
		metadata.ApprovedAt = time.Now()
	}

	// Get principal for approved_by if available
	if principal := getPrincipalFromContext(ctx); principal != nil {
		metadata.ApprovedBy = principal.ID()
	} else {
		metadata.ApprovedBy = "system"
	}

	r.plugins[metadata.Name] = metadata
	return r.saveRegistryLocked()
}

// UnregisterPlugin removes a plugin from the registry
func (r *PluginRegistry) UnregisterPlugin(ctx context.Context, pluginName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[pluginName]; !exists {
		return fmt.Errorf("plugin %s not found in registry", pluginName)
	}

	delete(r.plugins, pluginName)
	return r.saveRegistryLocked()
}

// GetPlugin retrieves plugin metadata by name
func (r *PluginRegistry) GetPlugin(pluginName string) (*PluginMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[pluginName]
	return plugin, exists
}

// ListPlugins returns all registered plugins
func (r *PluginRegistry) ListPlugins() map[string]*PluginMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	plugins := make(map[string]*PluginMetadata)
	for name, metadata := range r.plugins {
		plugins[name] = metadata
	}
	return plugins
}

// ListEnabledPlugins returns only enabled plugins
func (r *PluginRegistry) ListEnabledPlugins() map[string]*PluginMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	enabled := make(map[string]*PluginMetadata)
	for name, metadata := range r.plugins {
		if metadata.Enabled {
			enabled[name] = metadata
		}
	}
	return enabled
}

// IsPluginApproved checks if a plugin is approved and enabled
func (r *PluginRegistry) IsPluginApproved(pluginName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.plugins[pluginName]
	return exists && metadata.Enabled
}

// GetPluginPermissions returns the permissions for a plugin
func (r *PluginRegistry) GetPluginPermissions(pluginName string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	return metadata.Permissions, nil
}

// EnablePlugin enables a plugin
func (r *PluginRegistry) EnablePlugin(ctx context.Context, pluginName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata, exists := r.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	metadata.Enabled = true
	return r.saveRegistryLocked()
}

// DisablePlugin disables a plugin
func (r *PluginRegistry) DisablePlugin(ctx context.Context, pluginName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata, exists := r.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	metadata.Enabled = false
	return r.saveRegistryLocked()
}

// ScanPluginDirectory scans a directory for potential plugins and returns metadata
func (r *PluginRegistry) ScanPluginDirectory(dir string) ([]*PluginMetadata, error) {
	var plugins []*PluginMetadata

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check if it's a plugin file
		if r.isPluginFile(path) {
			metadata, err := r.extractPluginMetadata(path)
			if err != nil {
				// Log warning but continue scanning
				fmt.Printf("Warning: failed to extract metadata from %s: %v\n", path, err)
				return nil
			}
			plugins = append(plugins, metadata)
		}

		return nil
	})

	return plugins, err
}

// validateMetadata validates plugin metadata
func (r *PluginRegistry) validateMetadata(metadata *PluginMetadata) error {
	if metadata.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if metadata.Type == "" {
		return fmt.Errorf("plugin type is required")
	}
	if len(metadata.Permissions) == 0 {
		return fmt.Errorf("plugin must have at least one permission")
	}
	return nil
}

// isPluginFile checks if a file is a plugin file
func (r *PluginRegistry) isPluginFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".so" || ext == ".dylib" || ext == ".dll"
}

// extractPluginMetadata extracts metadata from a plugin file
func (r *PluginRegistry) extractPluginMetadata(path string) (*PluginMetadata, error) {
	// Read file to calculate checksum
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %w", err)
	}

	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Create basic metadata - in real implementation, this might
	// extract metadata from the plugin itself
	metadata := &PluginMetadata{
		Name:        name,
		Type:        "unknown", // Would be determined by examining the plugin
		Version:     "1.0.0",
		Description: fmt.Sprintf("Plugin %s", name),
		Permissions: []string{fmt.Sprintf("kernel.module.%s.*", name)}, // Scoped wildcard
		Checksum:    CalculateChecksum(data),
		Enabled:     false, // Disabled by default until approved
		ApprovedAt:  time.Time{},
		ApprovedBy:  "",
	}

	return metadata, nil
}

// getPrincipalFromContext extracts principal from context
func getPrincipalFromContext(ctx context.Context) interface {
	ID() string
} {
	// This would need to be implemented based on your auth system
	// For now, return nil
	return nil
}
