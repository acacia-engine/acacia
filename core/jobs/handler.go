package jobs

import "context"

// Handler is an interface for processing a specific type of background job.
// Each job type should have a corresponding Handler implementation.
type Handler interface {
	// Handle processes the given job.
	// The context can be used for tracing, deadlines, or cancellation.
	Handle(ctx context.Context, job Job) error
}

// HandlerRegistry is a collection of job handlers, typically managed by a Worker.
type HandlerRegistry interface {
	// RegisterHandler registers a handler for a specific job type.
	// If a handler for the given job type already exists, it should return an error.
	RegisterHandler(jobType string, handler Handler) error

	// GetHandler retrieves the handler for a given job type.
	// Returns nil if no handler is registered for the type.
	GetHandler(jobType string) Handler
}
