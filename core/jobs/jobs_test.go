package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// MockJob is a concrete implementation of the Job interface for testing.
type MockJob struct {
	BaseJob
	Name string
}

func NewMockJob(name string, payload interface{}) *MockJob {
	return &MockJob{
		BaseJob: BaseJob{
			JobType: "mock_job",
			Data:    payload,
		},
		Name: name,
	}
}

// MockEnqueuer is a mock implementation of the Enqueuer interface for testing.
type MockEnqueuer struct {
	EnqueuedJobs []Job
	EnqueueError error
	mu           sync.Mutex
}

func (m *MockEnqueuer) Enqueue(ctx context.Context, job Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.EnqueueError != nil {
		return m.EnqueueError
	}
	m.EnqueuedJobs = append(m.EnqueuedJobs, job)
	return nil
}

// MockHandler is a mock implementation of the Handler interface for testing.
type MockHandler struct {
	HandledJobs []Job
	HandleError error
	mu          sync.Mutex
}

func (m *MockHandler) Handle(ctx context.Context, job Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.HandleError != nil {
		return m.HandleError
	}
	m.HandledJobs = append(m.HandledJobs, job)
	return nil
}

// MockHandlerRegistry is a mock implementation of the HandlerRegistry interface.
type MockHandlerRegistry struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

func NewMockHandlerRegistry() *MockHandlerRegistry {
	return &MockHandlerRegistry{
		handlers: make(map[string]Handler),
	}
}

func (m *MockHandlerRegistry) RegisterHandler(jobType string, handler Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.handlers[jobType]; exists {
		return errors.New("handler already registered for this job type")
	}
	m.handlers[jobType] = handler
	return nil
}

func (m *MockHandlerRegistry) GetHandler(jobType string) Handler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handlers[jobType]
}

// MockWorker is a mock implementation of the Worker interface for testing.
type MockWorker struct {
	JobsToProcess chan Job
	StopChan      chan struct{}
	IsRunning     bool
	mu            sync.Mutex
}

func NewMockWorker() *MockWorker {
	return &MockWorker{
		JobsToProcess: make(chan Job, 100), // Buffered channel for jobs
		StopChan:      make(chan struct{}),
	}
}

func (m *MockWorker) Start(ctx context.Context, registry HandlerRegistry) error {
	m.mu.Lock()
	if m.IsRunning {
		m.mu.Unlock()
		return errors.New("worker already running")
	}
	m.IsRunning = true
	m.mu.Unlock()

	go func() {
		for {
			select {
			case job := <-m.JobsToProcess:
				handler := registry.GetHandler(job.Type())
				if handler != nil {
					_ = handler.Handle(job.Context(), job) // In a real worker, error handling would be more robust
				}
			case <-m.StopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (m *MockWorker) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.IsRunning {
		return errors.New("worker not running")
	}
	close(m.StopChan)
	m.IsRunning = false
	return nil
}

// TestJobInterface tests the basic functionality of the Job interface and BaseJob.
func TestJobInterface(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	job := NewMockJob("test_job_name", map[string]string{"data": "some_data"})
	job.SetContext(ctx)

	if job.Type() != "mock_job" {
		t.Errorf("Expected job type 'mock_job', got '%s'", job.Type())
	}

	payload := job.Payload().(map[string]string)
	if payload["data"] != "some_data" {
		t.Errorf("Expected payload data 'some_data', got '%s'", payload["data"])
	}

	if job.Context().Value("key") != "value" {
		t.Errorf("Expected context value 'value', got '%v'", job.Context().Value("key"))
	}

	newPayload := "new_data_string"
	job.SetPayload(newPayload)
	if job.Payload().(string) != newPayload {
		t.Errorf("Expected new payload '%s', got '%s'", newPayload, job.Payload().(string))
	}
}

// TestEnqueuer tests the Enqueuer interface with MockEnqueuer.
func TestEnqueuer(t *testing.T) {
	enqueuer := &MockEnqueuer{}
	job := NewMockJob("test_enqueue", "payload_data")
	ctx := context.Background()

	err := enqueuer.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	if len(enqueuer.EnqueuedJobs) != 1 {
		t.Errorf("Expected 1 enqueued job, got %d", len(enqueuer.EnqueuedJobs))
	}
	if enqueuer.EnqueuedJobs[0].Type() != "mock_job" {
		t.Errorf("Expected enqueued job type 'mock_job', got '%s'", enqueuer.EnqueuedJobs[0].Type())
	}

	// Test enqueue error
	enqueuer.EnqueueError = errors.New("failed to enqueue")
	err = enqueuer.Enqueue(ctx, job)
	if err == nil {
		t.Error("Expected enqueue to fail, but it succeeded")
	}
}

// TestHandler tests the Handler interface with MockHandler.
func TestHandler(t *testing.T) {
	handler := &MockHandler{}
	job := NewMockJob("test_handle", "handler_payload")
	ctx := context.Background()

	err := handler.Handle(ctx, job)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if len(handler.HandledJobs) != 1 {
		t.Errorf("Expected 1 handled job, got %d", len(handler.HandledJobs))
	}
	if handler.HandledJobs[0].Type() != "mock_job" {
		t.Errorf("Expected handled job type 'mock_job', got '%s'", handler.HandledJobs[0].Type())
	}

	// Test handle error
	handler.HandleError = errors.New("failed to handle")
	err = handler.Handle(ctx, job)
	if err == nil {
		t.Error("Expected handle to fail, but it succeeded")
	}
}

// TestHandlerRegistry tests the HandlerRegistry interface.
func TestHandlerRegistry(t *testing.T) {
	registry := NewMockHandlerRegistry()
	mockHandler := &MockHandler{}

	// Test RegisterHandler
	err := registry.RegisterHandler("test_type", mockHandler)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	// Test GetHandler
	retrievedHandler := registry.GetHandler("test_type")
	if retrievedHandler == nil {
		t.Error("Expected to retrieve handler, got nil")
	}

	// Test registering duplicate handler
	err = registry.RegisterHandler("test_type", mockHandler)
	if err == nil {
		t.Error("Expected RegisterHandler to fail for duplicate, but it succeeded")
	}

	// Test GetHandler for non-existent type
	nonExistentHandler := registry.GetHandler("non_existent_type")
	if nonExistentHandler != nil {
		t.Error("Expected nil for non-existent handler, got non-nil")
	}
}

// TestWorker tests the Worker interface with MockWorker.
func TestWorker(t *testing.T) {
	worker := NewMockWorker()
	registry := NewMockHandlerRegistry()
	mockHandler := &MockHandler{}
	registry.RegisterHandler("mock_job", mockHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test Start
	err := worker.Start(ctx, registry)
	if err != nil {
		t.Fatalf("Worker Start failed: %v", err)
	}
	if !worker.IsRunning {
		t.Error("Worker should be running after Start")
	}

	// Test starting already running worker
	err = worker.Start(ctx, registry)
	if err == nil {
		t.Error("Expected Start to fail for already running worker, but it succeeded")
	}

	// Enqueue a job
	job := NewMockJob("mock_job", "worker_payload")
	worker.JobsToProcess <- job // Simulate job coming from a queue

	// Give worker time to process
	time.Sleep(100 * time.Millisecond)

	if len(mockHandler.HandledJobs) != 1 {
		t.Errorf("Expected 1 job to be handled by mockHandler, got %d", len(mockHandler.HandledJobs))
	}
	if mockHandler.HandledJobs[0].Payload().(string) != "worker_payload" {
		t.Errorf("Expected handled job payload 'worker_payload', got '%s'", mockHandler.HandledJobs[0].Payload().(string))
	}

	// Test Stop
	err = worker.Stop(ctx)
	if err != nil {
		t.Fatalf("Worker Stop failed: %v", err)
	}
	if worker.IsRunning {
		t.Error("Worker should not be running after Stop")
	}

	// Test stopping already stopped worker
	err = worker.Stop(ctx)
	if err == nil {
		t.Error("Expected Stop to fail for already stopped worker, but it succeeded")
	}
}
