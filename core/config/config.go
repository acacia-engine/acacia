package config

import (
	"acacia/core/auth"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// AuthConfig holds the RBAC rules.
type AuthConfig struct {
	Roles []auth.Role `mapstructure:"roles"`
}

// Config holds the application's configuration settings.
type Config struct {
	Environment    string                            `mapstructure:"environment"`
	Auth           AuthConfig                        `mapstructure:"auth"`
	Modules        map[string]map[string]interface{} `mapstructure:"modules"`        // Generic configuration for modules
	Gateways       map[string]map[string]interface{} `mapstructure:"gateways"`       // Generic configuration for gateways
	Infrastructure map[string]map[string]interface{} `mapstructure:"infrastructure"` // Generic configuration for infrastructure components
	Timeouts       TimeoutsConfig                    `mapstructure:"timeouts"`       // Timeout configurations
}

// TimeoutsConfig holds timeout settings for various operations.
type TimeoutsConfig struct {
	ConfigChange     int `mapstructure:"config_change_seconds"`
	ModuleOperation  int `mapstructure:"module_operation_seconds"`
	GatewayOperation int `mapstructure:"gateway_operation_seconds"`
}

// LoadConfig loads the application configuration from a specified source (e.g., file, environment variables).
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set configuration file name and type
	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("yaml")   // or "json", "toml", etc.

	// Add paths to search for the config file
	v.AddConfigPath(".")           // current directory
	v.AddConfigPath("./configs")   // a "configs" directory
	v.AddConfigPath("/etc/acacia") // system-wide config

	// Read environment variables
	v.AutomaticEnv()         // read in environment variables that match
	v.SetEnvPrefix("ACACIA") // prefix for environment variables (e.g., ACACIA_SERVER_PORT)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	v.SetDefault("server_port", 8080)
	v.SetDefault("environment", "development")
	v.SetDefault("timeouts.config_change_seconds", 5)
	v.SetDefault("timeouts.module_operation_seconds", 10)
	v.SetDefault("timeouts.gateway_operation_seconds", 10)

	// Attempt to read the config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error and proceed with defaults and environment variables
			fmt.Fprintln(os.Stderr, "Config file not found, using defaults and environment variables.")
		} else {
			// Config file found but another error occurred
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Store the viper instance to allow registering change hooks later
	currentViper = v

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		if err := v.Unmarshal(&cfg); err != nil {
			fmt.Println(fmt.Errorf("failed to re-unmarshal config: %w", err))
		}
		// Notify all registered hooks
		for _, hook := range configChangeHooks {
			hook(&cfg)
		}
	})

	// Load module defaults
	if err := LoadModuleDefaults(&cfg, "modules"); err != nil {
		fmt.Printf("Warning: failed to load module defaults: %v\n", err)
		// Continue with main config even if module defaults fail
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// configChangeHooks stores functions to be called when the config changes.
var configChangeHooks []func(*Config)
var currentViper *viper.Viper

// AddConfigChangeHook registers a function to be called when the configuration changes.
func (c *Config) AddConfigChangeHook(hook func(*Config)) {
	configChangeHooks = append(configChangeHooks, hook)
}

// LoadModuleDefaults loads default configurations from all modules
func LoadModuleDefaults(cfg *Config, modulesDir string) error {
	if cfg.Modules == nil {
		cfg.Modules = make(map[string]map[string]interface{})
	}

	// Walk through modules directory to find default configs
	return filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for default-config.yaml files
		if !d.IsDir() && d.Name() == "default-config.yaml" {
			// Extract module name from path: modules/{module_name}/configs/default-config.yaml
			parts := strings.Split(path, string(filepath.Separator))
			if len(parts) >= 3 && parts[len(parts)-3] == "modules" && parts[len(parts)-2] == "configs" {
				moduleName := parts[len(parts)-3]

				// Read the default config file
				data, err := os.ReadFile(path)
				if err != nil {
					fmt.Printf("Warning: failed to read default config for module %s: %v\n", moduleName, err)
					return nil // Continue with other modules
				}

				// Parse the YAML
				var moduleDefaults map[string]interface{}
				if err := yaml.Unmarshal(data, &moduleDefaults); err != nil {
					fmt.Printf("Warning: failed to parse default config for module %s: %v\n", moduleName, err)
					return nil // Continue with other modules
				}

				// Merge with existing module config (user config takes precedence)
				if existing, exists := cfg.Modules[moduleName]; exists {
					// Merge defaults with existing config, preserving user settings
					mergeModuleConfig(existing, moduleDefaults)
				} else {
					// No existing config, use defaults
					cfg.Modules[moduleName] = moduleDefaults
				}

				fmt.Printf("Loaded default config for module: %s\n", moduleName)
			}
		}
		return nil
	})
}

// mergeModuleConfig merges default config into existing config, preserving user settings
func mergeModuleConfig(existing, defaults map[string]interface{}) {
	for key, defaultValue := range defaults {
		if _, exists := existing[key]; !exists {
			// Only add default if user hasn't set it
			existing[key] = defaultValue
		}
	}
}

// GenerateFromModules generates a complete config from all module defaults
func GenerateFromModules(modulesDir string) (*Config, error) {
	cfg := &Config{
		Environment: "development",
		Modules:     make(map[string]map[string]interface{}),
		Gateways:    make(map[string]map[string]interface{}),
	}

	// Load all module defaults
	err := LoadModuleDefaults(cfg, modulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load module defaults: %w", err)
	}

	return cfg, nil
}

// UpdateModuleDefaults regenerates default-config.yaml files for existing modules
func UpdateModuleDefaults(modulesDir string) error {
	return filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for registry.json files to identify modules
		if !d.IsDir() && d.Name() == "registry.json" {
			// Extract module name from path: modules/{module_name}/registry.json
			parts := strings.Split(path, string(filepath.Separator))
			if len(parts) >= 2 && parts[len(parts)-2] == "modules" {
				moduleName := parts[len(parts)-2]

				// Generate default config for this module
				err := generateModuleDefaultConfig(moduleName, filepath.Dir(path))
				if err != nil {
					fmt.Printf("Warning: failed to generate default config for module %s: %v\n", moduleName, err)
				} else {
					fmt.Printf("Generated default config for module: %s\n", moduleName)
				}
			}
		}
		return nil
	})
}

// GenerateMinimalConfig creates a minimal config with essential settings
func GenerateMinimalConfig() *Config {
	return &Config{
		Environment: "development",
		Modules:     make(map[string]map[string]interface{}),
		Gateways: map[string]map[string]interface{}{
			"httpapi": {
				"address":       ":8080",
				"debug":         true,
				"read_timeout":  "15s",
				"write_timeout": "15s",
				"idle_timeout":  "75s",
				"allowed_origins": []string{
					"http://localhost:3000",
					"http://localhost:8080",
				},
			},
			"websocket": {
				"address":      ":8081",
				"read_timeout": "60s",
			},
		},
		Timeouts: TimeoutsConfig{
			ConfigChange:     5,
			ModuleOperation:  10,
			GatewayOperation: 10,
		},
	}
}

// SaveGeneratedConfig saves a generated config to a file
func SaveGeneratedConfig(cfg *Config, filename string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// generateModuleDefaultConfig generates a default-config.yaml for a specific module
func generateModuleDefaultConfig(moduleName, modulePath string) error {
	configsDir := filepath.Join(modulePath, "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return fmt.Errorf("failed to create configs directory: %w", err)
	}

	// Generate default config content
	defaultConfig := fmt.Sprintf(`# Default configuration for %s module
# This file contains default settings that will be merged into the main config.yaml
# when the module is added to the system.

modules:
  %s:
    enabled: true
    # Add module-specific configuration here
    # Example settings (customize as needed):
    # debug: false
    # timeout: "30s"
    # workers: 2
`, moduleName, moduleName)

	configPath := filepath.Join(configsDir, "default-config.yaml")
	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

// Validate checks the configuration for required fields and valid values.
func (c *Config) Validate() error {
	switch c.Environment {
	case "development", "staging", "production":
		// valid
	default:
		return fmt.Errorf("invalid environment: %q", c.Environment)
	}
	return nil
}
