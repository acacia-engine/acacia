# Logger Module Documentation

## 1. Introduction to the Logger Module
The `logger` package provides the application-wide logging functionality for the Acacia application. It is built on top of `go.uber.org/zap`, a fast, structured, and leveled logging library. This module integrates with the `auth` package to provide permission-based logging, ensuring that only authorized components can emit logs.

## 2. Key Concepts

### 2.1. Global Logger Instance (`logger.Logger`)
*   `var Logger *zap.Logger`: This is the global `zap.Logger` instance used throughout the application. It is initialized with a development configuration that includes colored output for log levels, ISO8601 timestamps, and outputs to `stdout` and `stderr`.

### 2.2. Access Control Integration (`SetAccessController`)
*   `SetAccessController(ac auth.AccessController)`: This function allows the `auth.AccessController` to be injected into the logger. When an `AccessController` is set, all logging functions (`Info`, `Warn`, `Error`, `Fatal`, `Debug`) will first create a `auth.DefaultPrincipal` (representing the logging component) and then check if this principal is authorized to log using `ac.CanLog()`. If not authorized, the log message will be suppressed.

### 2.3. Component Name Context (`WithComponentName`)
*   `WithComponentName(ctx context.Context, componentName string) context.Context`: This helper function creates a new `context.Context` that includes a `componentName`. This allows modules and gateways to identify themselves in log messages, making it easier to trace logs back to their source.

### 2.4. Logging Functions
The `logger` package provides wrapper functions for standard logging levels. These functions automatically extract the component name from the provided `context.Context` and perform an authorization check via the `AccessController` (if set) before logging.

*   `Info(ctx context.Context, msg string, fields ...zap.Field)`: Logs an informational message.
*   `Warn(ctx context.Context, msg string, fields ...zap.Field)`: Logs a warning message.
*   `Error(ctx context.Context, msg string, fields ...zap.Field)`: Logs an error message.
*   `Fatal(ctx context.Context, msg string, fields ...zap.Field)`: Logs a fatal message and then exits the application.
*   `Debug(ctx context.Context, msg string, fields ...zap.Field)`: Logs a debug message.

### 2.5. Setting Custom Logger (`SetLogger`)
*   `SetLogger(l *zap.Logger)`: Allows external packages or tests to replace the internal `zap.Logger` instance. This is primarily for advanced re-configuration or testing scenarios.

## 3. Usage Examples

### Basic Logging
```go
package main

import (
	"acacia/core/logger"
	"context"
	"go.uber.org/zap"
)

func main() {
	// Basic info log
	logger.Info(context.Background(), "Application started.")

	// Log with fields
	logger.Warn(context.Background(), "Configuration file not found", zap.String("path", "/etc/acacia/config.yaml"))

	// Error log
	logger.Error(context.Background(), "Failed to connect to database", zap.Error(fmt.Errorf("connection refused")))
}
```

### Logging with Component Name
```go
package main

import (
	"acacia/core/logger"
	"context"
	"fmt"
	"go.uber.org/zap"
)

func processRequest(ctx context.Context, requestID string) {
	// Create a context with the component name
	ctx = logger.WithComponentName(ctx, "HTTPGateway")
	logger.Info(ctx, "Processing new request", zap.String("request_id", requestID))

	// Simulate some work
	if requestID == "invalid-123" {
		logger.Error(ctx, "Invalid request format", zap.String("request_id", requestID))
	} else {
		logger.Debug(ctx, "Request processed successfully", zap.String("request_id", requestID))
	}
}

func main() {
	processRequest(context.Background(), "req-001")
	processRequest(context.Background(), "invalid-123")
}
```

### Logging with Access Control
```go
package main

import (
	"acacia/core/auth"
	"acacia/core/logger"
	"context"
	"fmt"
	"go.uber.org/zap"
)

// CustomAccessController allows only "authorized-component" to log
type CustomAccessController struct{}

func (c *CustomAccessController) CanLog(ctx context.Context, p auth.Principal) bool {
	return p.ID() == "authorized-component"
}

func (c *CustomAccessController) CanAccessMetrics(ctx context.Context, p auth.Principal) bool {
	return true // Not relevant for this example
}

func (c *CustomAccessController) CanPublishEvent(ctx context.Context, p auth.Principal, eventType string) bool {
	return true
}

func (c *CustomAccessController) CanAccessConfig(ctx context.Context, p auth.Principal, configKey string) bool {
	return true
}

func (c *CustomAccessController) CanReloadModule(ctx context.Context, p auth.Principal, moduleToReload string) bool {
	return true
}

func (c *CustomAccessController) HasPermission(p auth.Principal, perm auth.Permission) bool {
	return true
}

func main() {
	// Inject the custom access controller into the logger
	logger.SetAccessController(&CustomAccessController{})

	// This log will be suppressed
	unauthorizedPrincipal := auth.NewDefaultPrincipal("unauthorized-component", "component", []string{})
	ctxUnauthorized := logger.WithComponentName(context.Background(), unauthorizedPrincipal.ID())
	logger.Info(ctxUnauthorized, "Attempting to log from unauthorized component")

	// This log will be allowed
	authorizedPrincipal := auth.NewDefaultPrincipal("authorized-component", "component", []string{})
	ctxAuthorized := logger.WithComponentName(context.Background(), authorizedPrincipal.ID())
	logger.Info(ctxAuthorized, "Logging from authorized component", zap.String("data", "sensitive info"))

	// Reset access controller (optional, for testing/cleanup)
	logger.SetAccessController(nil)
	logger.Info(context.Background(), "Access controller reset, all logs allowed again.")
}
