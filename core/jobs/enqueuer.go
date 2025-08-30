package jobs

import "context"

// Enqueuer is an interface for enqueuing background jobs.
// Concrete implementations will interact with specific queuing technologies
// (e.g., Redis, RabbitMQ, Kafka).
type Enqueuer interface {
	// Enqueue adds a job to the queue.
	// The context can be used for tracing, deadlines, or cancellation.
	Enqueue(ctx context.Context, job Job) error
}
