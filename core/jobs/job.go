package jobs

import "context"

// Job represents a background job that can be enqueued and processed.
// Implementations should provide methods for serialization/deserialization
// if the job needs to be persisted or transmitted across processes.
type Job interface {
	// Type returns a string identifier for the job type.
	Type() string

	// Payload returns the job's data. The concrete type of the payload
	// will depend on the specific job implementation.
	Payload() interface{}

	// SetPayload sets the job's data. This is useful for deserialization.
	SetPayload(interface{})

	// Context returns the context associated with the job.
	Context() context.Context

	// SetContext sets the context for the job.
	SetContext(context.Context)
}

// BaseJob provides a basic implementation for common Job methods.
// It can be embedded in concrete job structs to reduce boilerplate.
type BaseJob struct {
	JobType string          `json:"job_type"`
	Data    interface{}     `json:"data"`
	Ctx     context.Context `json:"-"` // Context is not serialized
}

// Type returns the job type.
func (b *BaseJob) Type() string {
	return b.JobType
}

// Payload returns the job's data.
func (b *BaseJob) Payload() interface{} {
	return b.Data
}

// SetPayload sets the job's data.
func (b *BaseJob) SetPayload(payload interface{}) {
	b.Data = payload
}

// Context returns the context associated with the job.
func (b *BaseJob) Context() context.Context {
	if b.Ctx == nil {
		return context.Background()
	}
	return b.Ctx
}

// SetContext sets the context for the job.
func (b *BaseJob) SetContext(ctx context.Context) {
	b.Ctx = ctx
}
