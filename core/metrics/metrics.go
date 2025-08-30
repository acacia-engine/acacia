package metrics

import (
	"acacia/core/auth" // Import the auth package for AccessController
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var globalAccessController auth.AccessController // Global access controller

// SetAccessController allows the kernel to inject the controller
func SetAccessController(ac auth.AccessController) {
	globalAccessController = ac
}

// getComponentNameFromContext extracts the component name from the context.
func getComponentNameFromContext(ctx context.Context) string {
	// componentNameKey type matches logger package
	type componentNameKeyType string
	const componentNameKey componentNameKeyType = "componentName"
	if name, ok := ctx.Value(componentNameKey).(string); ok {
		return name
	}
	return "unknown" // Default if not found in context
}

// Define common metrics (these are the original global metrics)
var (
	// RequestCounter counts the total number of requests.
	RequestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "acacia_requests_total",
		Help: "Total number of requests.",
	}, []string{"handler", "method"})

	// RequestDuration measures the duration of requests.
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "acacia_request_duration_seconds",
		Help:    "Duration of requests in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"handler", "method"})

	// ModuleStartCounter counts module start attempts.
	ModuleStartCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "acacia_module_starts_total",
		Help: "Total number of module start attempts.",
	}, []string{"module", "status"})

	// ModuleStopCounter counts module stop attempts.
	ModuleStopCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "acacia_module_stops_total",
		Help: "Total number of module stop attempts.",
	}, []string{"module", "status"})

	// GatewayStartCounter counts gateway start attempts.
	GatewayStartCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "acacia_gateway_starts_total",
		Help: "Total number of gateway start attempts.",
	}, []string{"gateway", "status"})

	// GatewayStopCounter counts gateway stop attempts.
	GatewayStopCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "acacia_gateway_stops_total",
		Help: "Total number of gateway stop attempts.",
	}, []string{"gateway", "status"})
)

// Wrapper functions for controlled access to metrics

// IncrementRequestCounter safely increments the request counter with permission check
func IncrementRequestCounter(ctx context.Context, handler, method string) {
	componentName := getComponentNameFromContext(ctx)
	principal := auth.NewDefaultPrincipal(componentName, "component", []string{}) // Create a principal
	if globalAccessController != nil && !globalAccessController.CanAccessMetrics(ctx, principal) {
		return // Not authorized
	}
	RequestCounter.WithLabelValues(handler, method).Inc()
}

// ObserveRequestDuration safely observes request duration with permission check
func ObserveRequestDuration(ctx context.Context, handler, method string, duration float64) {
	componentName := getComponentNameFromContext(ctx)
	principal := auth.NewDefaultPrincipal(componentName, "component", []string{}) // Create a principal
	if globalAccessController != nil && !globalAccessController.CanAccessMetrics(ctx, principal) {
		return // Not authorized
	}
	RequestDuration.WithLabelValues(handler, method).Observe(duration)
}

// Note: ModuleStartCounter, ModuleStopCounter, GatewayStartCounter, and GatewayStopCounter
// are used internally by the kernel and remain as direct access for now.
// In a more complete implementation, we might want to restrict these as well.
