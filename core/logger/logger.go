package logger

import (
	"acacia/core/auth" // Import the auth package for AccessController
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the application-wide logger.
var Logger *zap.Logger

// componentNameKey is a context key for storing the component name.
type componentNameKeyType string

const componentNameKey componentNameKeyType = "componentName"

var globalAccessController auth.AccessController // New global variable

func init() {
	// Configure development logger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Add color to level output
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	var err error
	Logger, err = config.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(Logger) // Set as global logger
}

// SetAccessController allows the kernel to inject the controller
func SetAccessController(ac auth.AccessController) {
	globalAccessController = ac
}

// getComponentNameFromContext extracts the component name from the context.
func getComponentNameFromContext(ctx context.Context) string {
	if name, ok := ctx.Value(componentNameKey).(string); ok {
		return name
	}
	return "unknown" // Default if not found in context
}

// WithComponentName creates a new context with the component name set.
// This is a helper function to make it easier for modules and gateways to identify themselves.
func WithComponentName(ctx context.Context, componentName string) context.Context {
	return context.WithValue(ctx, componentNameKey, componentName)
}

// Info checks permission before logging
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	componentName := getComponentNameFromContext(ctx)
	principal := auth.NewDefaultPrincipal(componentName, "component", []string{}) // Create a principal
	if globalAccessController != nil && !globalAccessController.CanLog(ctx, principal) {
		return // Not authorized to log
	}
	Logger.Info(msg, fields...)
}

// Warn checks permission before logging
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	componentName := getComponentNameFromContext(ctx)
	principal := auth.NewDefaultPrincipal(componentName, "component", []string{}) // Create a principal
	if globalAccessController != nil && !globalAccessController.CanLog(ctx, principal) {
		return // Not authorized to log
	}
	Logger.Warn(msg, fields...)
}

// Error checks permission before logging
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	componentName := getComponentNameFromContext(ctx)
	principal := auth.NewDefaultPrincipal(componentName, "component", []string{}) // Create a principal
	if globalAccessController != nil && !globalAccessController.CanLog(ctx, principal) {
		return // Not authorized to log
	}
	Logger.Error(msg, fields...)
}

// Fatal checks permission before logging
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	componentName := getComponentNameFromContext(ctx)
	principal := auth.NewDefaultPrincipal(componentName, "component", []string{}) // Create a principal
	if globalAccessController != nil && !globalAccessController.CanLog(ctx, principal) {
		return // Not authorized to log
	}
	Logger.Fatal(msg, fields...)
}

// Debug checks permission before logging
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	componentName := getComponentNameFromContext(ctx)
	principal := auth.NewDefaultPrincipal(componentName, "component", []string{}) // Create a principal
	if globalAccessController != nil && !globalAccessController.CanLog(ctx, principal) {
		return // Not authorized to log
	}
	Logger.Debug(msg, fields...)
}

// SetLogger allows external packages to set the internal zap.Logger instance.
// This is primarily for testing purposes or advanced logger re-configuration.
func SetLogger(l *zap.Logger) {
	Logger = l
}
