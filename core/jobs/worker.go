package jobs

import "context"

// Worker is an interface for processing background jobs from a queue.
// Concrete implementations will interact with specific queuing technologies
// and use a HandlerRegistry to dispatch jobs to the correct handler.
type Worker interface {
	// Start begins processing jobs from the queue.
	// This method should typically run in a goroutine.
	Start(ctx context.Context, registry HandlerRegistry) error

	// Stop gracefully stops the worker, allowing any in-flight jobs to complete.
	Stop(ctx context.Context) error
}
