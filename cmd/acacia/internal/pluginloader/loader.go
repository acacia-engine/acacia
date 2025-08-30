package pluginloader

import (
	"acacia/core/auth"
	"acacia/core/config" // Import the config package
	"acacia/core/kernel"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"go.uber.org/zap"
)

// Logger is a simple interface for logging within the plugin loader.
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field) // Add Debug method
}

// LoadPlugins scans the specified plugin directory, loads Go plugin binaries,
// and adds them to the kernel, providing them with their respective configurations.
func LoadPlugins(k kernel.Kernel, pluginDir string, cfg *config.Config, logger Logger) error {
	logger.Info("Scanning for plugins", zap.String("directory", pluginDir))

	files, err := fs.ReadDir(osFS{}, pluginDir) // Use osFS for ReadDir
	if err != nil {
		logger.Error("Failed to read plugin directory", zap.String("directory", pluginDir), zap.Error(err))
		return fmt.Errorf("read plugin directory %s: %w", pluginDir, err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".so") && !strings.HasSuffix(fileName, ".dylib") && !strings.HasSuffix(fileName, ".dll") {
			logger.Debug("Skipping non-plugin file", zap.String("file", fileName))
			continue
		}

		pluginPath := filepath.Join(pluginDir, fileName)
		logger.Info("Attempting to load plugin", zap.String("path", pluginPath))

		p, err := plugin.Open(pluginPath)
		if err != nil {
			logger.Error("Failed to open plugin", zap.String("path", pluginPath), zap.Error(err))
			return fmt.Errorf("open plugin %s: %w", pluginPath, err)
		}

		// Try to load as a Module
		newModuleSym, err := p.Lookup("NewModule")
		if err == nil {
			if newModuleFunc, ok := newModuleSym.(func() kernel.Module); ok {
				moduleInstance := newModuleFunc()
				// Configure the module before adding/starting
				if moduleConfig, ok := cfg.Modules[moduleInstance.Name()]; ok {
					if err := moduleInstance.Configure(moduleConfig); err != nil {
						logger.Error("Failed to configure module", zap.String("module", moduleInstance.Name()), zap.Error(err))
						return fmt.Errorf("configure module %s: %w", moduleInstance.Name(), err)
					}
				} else {
					logger.Warn("No configuration found for module", zap.String("module", moduleInstance.Name()))
					// Allow module to proceed without specific config if not found.
					// Modules should handle nil or empty config gracefully if it's optional.
					if err := moduleInstance.Configure(nil); err != nil {
						logger.Error("Failed to configure module with nil config", zap.String("module", moduleInstance.Name()), zap.Error(err))
						return fmt.Errorf("configure module %s with nil config: %w", moduleInstance.Name(), err)
					}
				}
				// Create context with system principal for plugin loading
				systemPrincipal := auth.NewDefaultPrincipal("plugin-loader", "system", []string{"kernel.module.*"})
				ctx := auth.ContextWithPrincipal(context.Background(), systemPrincipal)

				if err := k.AddModule(ctx, moduleInstance); err != nil {
					logger.Error("Failed to add loaded module to kernel", zap.String("module", moduleInstance.Name()), zap.Error(err))
					return fmt.Errorf("add module %s: %w", moduleInstance.Name(), err)
				}
				logger.Info("Successfully loaded and added module plugin", zap.String("module", moduleInstance.Name()), zap.String("path", pluginPath))
				continue // Move to next file
			}
		}

		// Try to load as a Gateway
		newGatewaySym, err := p.Lookup("NewGateway")
		if err == nil {
			if newGatewayFunc, ok := newGatewaySym.(func() kernel.Gateway); ok {
				gatewayInstance := newGatewayFunc()
				// Configure the gateway before adding/starting
				// Pass only the specific gateway's configuration
				if gatewayConfig, ok := cfg.Gateways[gatewayInstance.Name()]; ok {
					if err := gatewayInstance.Configure(gatewayConfig); err != nil {
						logger.Error("Failed to configure gateway", zap.String("gateway", gatewayInstance.Name()), zap.Error(err))
						return fmt.Errorf("configure gateway %s: %w", gatewayInstance.Name(), err)
					}
				} else {
					logger.Warn("No configuration found for gateway", zap.String("gateway", gatewayInstance.Name()))
					// Continue without configuration, or return an error if configuration is mandatory
					// For now, we'll allow it to proceed without specific config if not found.
				}
				if err := k.AddGateway(gatewayInstance); err != nil {
					logger.Error("Failed to add loaded gateway to kernel", zap.String("gateway", gatewayInstance.Name()), zap.Error(err))
					return fmt.Errorf("add gateway %s: %w", gatewayInstance.Name(), err)
				}
				logger.Info("Successfully loaded and added gateway plugin", zap.String("gateway", gatewayInstance.Name()), zap.String("path", pluginPath))
				continue // Move to next file
			}
		}

		logger.Warn("Plugin does not export NewModule or NewGateway function", zap.String("path", pluginPath))
	}
	return nil
}

// osFS implements fs.FS for os operations.
type osFS struct{}

func (osFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (osFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}
