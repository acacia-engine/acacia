# Jobs Module

The `jobs` module in the Acacia Engine provides a robust and flexible system for handling asynchronous tasks and background processing. It defines interfaces for enqueuing, handling, and executing jobs, ensuring that long-running or resource-intensive operations do not block the main application flow.

## Key Components

### `Job` Interface

The `Job` interface represents a single unit of work to be processed. All jobs must implement this interface.

```go
type Job interface {
	Type() string
	Payload() interface{}
	SetPayload(payload interface{})
	Context() context.Context
	SetContext(ctx context.Context)
}
```

*   `Type()`: Returns a string identifier for the job type. This is used by the `HandlerRegistry` to dispatch the job to the correct handler.
*   `Payload()`: Returns the data associated with the job.
*   `SetPayload(payload interface{})`: Sets the data for the job.
*   `Context()`: Returns the `context.Context` associated with the job, allowing for cancellation and deadline propagation.
*   `SetContext(ctx context.Context)`: Sets the `context.Context` for the job.

### `Enqueuer` Interface

The `Enqueuer` interface is responsible for adding jobs to a queue for asynchronous processing.

```go
type Enqueuer interface {
	Enqueue(ctx context.Context, job Job) error
}
```

*   `Enqueue(ctx context.Context, job Job) error`: Adds the given `Job` to the queue. The `context.Context` can be used for cancellation or timeouts during the enqueue operation.

### `Handler` Interface

The `Handler` interface defines how a specific type of job is processed. Each job type should have a corresponding handler.

```go
type Handler interface {
	Handle(ctx context.Context, job Job) error
}
```

*   `Handle(ctx context.Context, job Job) error`: Processes the given `Job`. The `context.Context` can be used for cancellation or timeouts during job execution.

### `HandlerRegistry` Interface

The `HandlerRegistry` manages the mapping between job types and their respective `Handler` implementations.

```go
type HandlerRegistry interface {
	RegisterHandler(jobType string, handler Handler) error
	GetHandler(jobType string) Handler
}
```

*   `RegisterHandler(jobType string, handler Handler) error`: Registers a `Handler` for a specific `jobType`.
*   `GetHandler(jobType string) Handler`: Retrieves the `Handler` registered for the given `jobType`.

### `Worker` Interface

The `Worker` interface represents a component that consumes jobs from a queue and dispatches them to the appropriate handlers using the `HandlerRegistry`.

```go
type Worker interface {
	Start(ctx context.Context, registry HandlerRegistry) error
	Stop(ctx context.Context) error
}
```

*   `Start(ctx context.Context, registry HandlerRegistry) error`: Starts the worker, beginning to process jobs. It requires a `HandlerRegistry` to know which handler to use for each job type.
*   `Stop(ctx context.Context) error`: Gracefully stops the worker, allowing any currently processing jobs to complete.
