# Config Module Documentation

## 1. Introduction to the Config Module
The `config` package is responsible for loading and managing the application's configuration settings. It utilizes the `viper` library to provide flexible configuration loading from various sources, including configuration files, environment variables, and default values.

## 2. Key Concepts

### 2.1. Config Struct
The `Config` struct holds all the application's configuration settings.

**Fields:**
*   `Environment string`: Represents the current operating environment (e.g., "development", "production"). Mapped from `environment`.
*   `Auth AuthConfig`: Holds the Role-Based Access Control (RBAC) configuration, including roles and their associated permissions. Mapped from `auth`.
*   `Modules map[string]map[string]interface{}`: A generic map to hold configuration specific to different modules. Mapped from `modules`.
*   `Gateways map[string]map[string]interface{}`: A generic map to hold configuration specific to different gateways. Mapped from `gateways`.
*   `Infrastructure map[string]map[string]interface{}`: A generic map to hold configuration specific to infrastructure components. Mapped from `infrastructure`.

### 2.2. TimeoutsConfig Struct
The `TimeoutsConfig` struct holds timeout configurations for various system operations.

**Fields:**
*   `ConfigChange int`: Timeout in seconds for configuration change operations (default: 5).
*   `ModuleOperation int`: Timeout in seconds for module lifecycle operations (default: 10).
*   `GatewayOperation int`: Timeout in seconds for gateway operations (default: 10).

### 2.3. AddConfigChangeHook Method
`(c *Config) AddConfigChangeHook(hook func(*Config))`
*   Registers a function to be called when the application's configuration changes (e.g., when the `config.yaml` file is modified and reloaded).
*   `hook`: A function that receives the updated `Config` struct.
*   Automatically called when the config file is modified at runtime.

### 2.4. Validate Method
`(c *Config) Validate() error`
*   Performs validation checks on the loaded configuration.
*   Currently, it validates the `Environment` field to ensure it's one of "development", "staging", or "production".
*   Returns an error if the configuration is invalid.

### 2.5. AuthConfig Struct
The `AuthConfig` struct defines the structure for Role-Based Access Control (RBAC) configuration within the application. It contains a list of roles, each with a name and a set of permissions.

**Fields:**
*   `Roles []auth.Role`: A slice of `auth.Role` structs, where each `Role` defines a name and a list of `auth.Permission`s.

### 2.7. Additional Configuration Functions

**LoadModuleDefaults(cfg *Config, modulesDir string) error**
*   Loads default configurations from all modules and merges them into the main config.
*   Looks for `default-config.yaml` files in module directories.
*   User settings take precedence over module defaults.

**GenerateFromModules(modulesDir string) (*Config, error)**
*   Generates a complete configuration by loading defaults from all available modules.
*   Useful for bootstrapping or generating configuration templates.

**GenerateMinimalConfig() *Config**
*   Creates a minimal configuration with essential settings for common use cases.
*   Includes default configurations for httpapi and websocket gateways.

**SaveGeneratedConfig(cfg *Config, filename string) error**
*   Saves a configuration struct to a YAML file.
*   Useful for generating and persisting configuration files.

### 2.6. LoadConfig Function
`LoadConfig() (*Config, error)`
*   Loads the application configuration using Viper for flexible configuration management.
*   **Configuration File Search Paths:**
    *   Current directory (`.`): `config.yaml`, `config.json`
    *   `configs` subdirectory (`./configs`)
    *   System-wide configuration (`/etc/acacia`)
*   **Environment Variables:** Automatically reads environment variables prefixed with `ACACIA_` (e.g., `ACACIA_ENVIRONMENT`, `ACACIA_SERVER_PORT`).
*   **Default Values:**
    *   `environment`: `"development"`
    *   `server_port`: `8080` (for backwards compatibility, accessed via Viper)
    *   `timeouts.config_change_seconds`: `5`
    *   `timeouts.module_operation_seconds`: `10`
    *   `timeouts.gateway_operation_seconds`: `10`
*   **Dynamic Reloading:** Automatically watches config file for changes and reloads when modified.
*   **Error Handling:** If the config file is not found, proceeds with defaults and environment variables. Other file reading/parsing errors are returned.
*   **Module Defaults:** Automatically loads default configurations from modules' `default-config.yaml` files.

## 3. Usage Example

### Loading and Accessing Configuration
To use the `config` module, you typically call `LoadConfig` at the application's startup.

```go
package main

import (
	"acacia/core/config"
	"fmt"
	"log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	fmt.Printf("Application Environment: %s\n", cfg.Environment)
	fmt.Printf("Config Change Timeout: %d seconds\n", cfg.Timeouts.ConfigChange)
	fmt.Printf("Module Operation Timeout: %d seconds\n", cfg.Timeouts.ModuleOperation)

	// Accessing a default value (e.g., server_port, if not overridden in config file or env)
	// Note: To access default values set by viper, you might need to use viper directly
	// or define them in your Config struct if they are part of a specific module/gateway config.
	// For simplicity, this example focuses on fields directly in the Config struct.

	// Example of accessing module-specific configuration (if defined in your config file)
	if moduleCfg, ok := cfg.Modules["my-module"]; ok {
		fmt.Printf("My Module Config: %+v\n", moduleCfg)
		if setting, ok := moduleCfg["some_setting"]; ok {
			fmt.Printf("My Module's 'some_setting': %v\n", setting)
		}
	} else {
		fmt.Println("No configuration found for 'my-module'.")
	}

	// Register a config change hook
	cfg.AddConfigChangeHook(func(newCfg *config.Config) {
		fmt.Printf("Config changed! New environment: %s\n", newCfg.Environment)
	})

	// You can also set environment variables like ACACIA_ENVIRONMENT=production
	// and ACACIA_MODULES_MY_MODULE_SOME_SETTING=value to override.
}
```

### Example `config.yaml`
```yaml
environment: production
timeouts:
  config_change_seconds: 10
  module_operation_seconds: 15
  gateway_operation_seconds: 12
auth:
  roles:
    - name: admin
      permissions:
        - "kernel.module.add"
        - "kernel.module.remove"
modules:
  my-module:
    some_setting: "production_value"
    another_setting: 123
gateways:
  httpapi:
    port: 8081
  websocket:
    address: ":8082"
infrastructure:
  database:
    driver: "postgres"
    url: "postgresql://localhost/acacia"
