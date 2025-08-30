# Metrics Module Documentation

## 1. Introduction to the Metrics Module
The `metrics` package provides application-wide metrics collection using Prometheus client libraries (`github.com/prometheus/client_golang/prometheus`). It defines various counters and histograms to track operational aspects of the Acacia application, such as request counts, request durations, and module/gateway lifecycle events. This module also integrates with the `auth` package to control access to metrics.

## 2. Key Concepts

### 2.1. Global Access Controller Integration (`SetAccessController`)
*   `SetAccessController(ac auth.AccessController)`: This function allows the `auth.AccessController` to be injected into the metrics module. When an `AccessController` is set, certain metric operations (like incrementing request counters or observing request durations) will first create a `auth.DefaultPrincipal` (representing the metrics-emitting component) and then check if this principal is authorized to access/modify metrics using `ac.CanAccessMetrics()`. If not authorized, the metric operation will be suppressed.

### 2.2. Component Name Context
Similar to the `logger` package, the `metrics` package can extract a component name from the `context.Context` to provide more granular labeling for metrics. This allows metrics to be associated with the specific module or gateway that generated them.

### 2.3. Defined Metrics
The `metrics` package defines several global Prometheus metric vectors:

*   **`RequestCounter`** (`acacia_requests_total`): A counter that tracks the total number of requests.
    *   **Labels**: `handler` (e.g., API endpoint), `method` (e.g., GET, POST).
*   **`RequestDuration`** (`acacia_request_duration_seconds`): A histogram that measures the duration of requests in seconds.
    *   **Labels**: `handler`, `method`.
    *   **Buckets**: Uses Prometheus's default histogram buckets.
*   **`ModuleStartCounter`** (`acacia_module_starts_total`): A counter that tracks module start attempts.
    *   **Labels**: `module` (module name), `status` (e.g., "attempt", "success", "failed").
*   **`ModuleStopCounter`** (`acacia_module_stops_total`): A counter that tracks module stop attempts.
    *   **Labels**: `module` (module name), `status` (e.g., "attempt", "success", "failed").
*   **`GatewayStartCounter`** (`acacia_gateway_starts_total`): A counter that tracks gateway start attempts.
    *   **Labels**: `gateway` (gateway name), `status` (e.g., "attempt", "success", "failed").
*   **`GatewayStopCounter`** (`acacia_gateway_stops_total`): A counter that tracks gateway stop attempts.
    *   **Labels**: `gateway` (gateway name), `status` (e.g., "attempt", "success", "failed").

### 2.4. Wrapper Functions for Controlled Access
To enforce access control, wrapper functions are provided for certain metrics:

*   `IncrementRequestCounter(ctx context.Context, handler, method string)`: Safely increments the `RequestCounter` after checking permissions.
*   `ObserveRequestDuration(ctx context.Context, handler, method string, duration float64)`: Safely observes the `RequestDuration` after checking permissions.

*(Note: `ModuleStartCounter`, `ModuleStopCounter`, `GatewayStartCounter`, and `GatewayStopCounter` are currently used internally by the kernel and are directly accessed. In a more complete implementation, these might also be restricted via wrapper functions.)*

## 3. Usage Examples

### Incrementing Request Counter
```go
package main

import (
	"acacia/core/metrics"
	"acacia/core/logger" // For WithComponentName
	"context"
	"fmt"
	"time"
)

func handleAPIRequest(ctx context.Context, handlerName, method string) {
	ctx = logger.WithComponentName(ctx, "HTTPGateway") // Identify the component
	start := time.Now()

	// Increment request counter
	metrics.IncrementRequestCounter(ctx, handlerName, method)

	// Simulate request processing
	time.Sleep(50 * time.Millisecond)

	// Observe request duration
	duration := time.Since(start).Seconds()
	metrics.ObserveRequestDuration(ctx, handlerName, method, duration)

	fmt.Printf("Handled %s %s in %.2f seconds.\n", method, handlerName, duration)
}

func main() {
	// In a real application, you would expose Prometheus metrics via an HTTP endpoint.
	// For example: http.Handle("/metrics", promhttp.Handler())

	handleAPIRequest(context.Background(), "/api/v1/users", "GET")
	handleAPIRequest(context.Background(), "/api/v1/products", "POST")
}
```

### Using Metrics with Access Control
```go
package main

import (
	"acacia/core/auth"
	"acacia/core/metrics"
	"acacia/core/logger" // For WithComponentName
	"context"
	"fmt"
)

// CustomAccessController allows only "authorized-metrics-component" to access metrics
type CustomAccessController struct{}

func (c *CustomAccessController) CanLog(ctx context.Context, p auth.Principal) bool {
	return true // Not relevant for this example
}

func (c *CustomAccessController) CanAccessMetrics(ctx context.Context, p auth.Principal) bool {
	return p.ID() == "authorized-metrics-component"
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
	// Inject the custom access controller into the metrics module
	metrics.SetAccessController(&CustomAccessController{})

	// This metric increment will be suppressed
	unauthorizedPrincipal := auth.NewDefaultPrincipal("unauthorized-component", "component", []string{})
	ctxUnauthorized := logger.WithComponentName(context.Background(), unauthorizedPrincipal.ID())
	metrics.IncrementRequestCounter(ctxUnauthorized, "/data", "GET")
	fmt.Println("Attempted to increment metric from unauthorized component.")

	// This metric increment will be allowed
	authorizedPrincipal := auth.NewDefaultPrincipal("authorized-metrics-component", "component", []string{})
	ctxAuthorized := logger.WithComponentName(context.Background(), authorizedPrincipal.ID())
	metrics.IncrementRequestCounter(ctxAuthorized, "/status", "GET")
	fmt.Println("Incremented metric from authorized component.")

	// Reset access controller (optional, for testing/cleanup)
	metrics.SetAccessController(nil)
	fmt.Println("Access controller reset, all metrics access allowed again.")
}
