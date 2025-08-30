package events

import (
	"context"
	"testing"
	"time"
)

// TestEvent implements the TypedEvent interface for testing purposes.
type TestEvent struct {
	Type    string
	Payload string
}

func (e TestEvent) EventType() string {
	return e.Type
}

func TestBus_PublishSubscribe(t *testing.T) {
	b := New()
	ch, cancel, err := b.Subscribe("topic")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer cancel()

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second)
	defer cancelCtx()
	b.Publish(ctx, "topic", TestEvent{Type: "test", Payload: "hello"})

	select {
	case v := <-ch:
		typedV, ok := v.(TestEvent)
		if !ok {
			t.Fatalf("expected TestEvent, got %T", v)
		}
		if typedV.Payload != "hello" {
			t.Fatalf("unexpected payload: %v", typedV.Payload)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestBus_CancelUnsubscribe(t *testing.T) {
	b := New()
	ch, cancel, err := b.Subscribe("topic")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	cancel()
	// After cancel, channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel after cancel")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for channel close")
	}
	// Should not panic on publish after cancel
	b.Publish(context.Background(), "topic", TestEvent{Type: "test", Payload: "ignored"})
}

func TestBus_Close(t *testing.T) {
	b := New()
	ch1, _, _ := b.Subscribe("t")
	ch2, _, _ := b.Subscribe("t")
	b.Close()
	// both channels should be closed
	for i, ch := range []<-chan TypedEvent{ch1, ch2} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Fatalf("expected ch%d closed", i+1)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timeout waiting ch%d to close", i+1)
		}
	}
}
