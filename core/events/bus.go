package events

import (
	"context"
	"sync"
)

// Bus is a minimal pub/sub event bus using Go channels.
// It is intentionally simple and dependency-free.
// Topics are identified by string; payloads are any type.

// TypedEvent is an interface that all event payloads should implement to provide type safety.
type TypedEvent interface {
	EventType() string // Returns a string identifier for the event type.
}

type Bus interface {
	Subscribe(topic string) (<-chan TypedEvent, func(), error)
	Publish(ctx context.Context, topic string, payload TypedEvent)
	Close()
}

type bus struct {
	mu     sync.RWMutex
	topics map[string]map[chan TypedEvent]struct{}
	closed bool
}

// New returns a new event bus instance.
func New() Bus {
	return &bus{topics: make(map[string]map[chan TypedEvent]struct{})}
}

func (b *bus) Subscribe(topic string) (<-chan TypedEvent, func(), error) {
	ch := make(chan TypedEvent, 16)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		close(ch)
		return ch, func() {}, nil
	}
	subs, ok := b.topics[topic]
	if !ok {
		subs = make(map[chan TypedEvent]struct{})
		b.topics[topic] = subs
	}
	subs[ch] = struct{}{}
	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if subs, ok := b.topics[topic]; ok {
			if _, exists := subs[ch]; exists {
				delete(subs, ch)
				close(ch)
				if len(subs) == 0 {
					delete(b.topics, topic)
				}
			}
		}
	}
	return ch, cancel, nil
}

func (b *bus) Publish(ctx context.Context, topic string, payload TypedEvent) {
	b.mu.RLock()
	subs := b.topics[topic]
	// Copy channels to avoid holding lock while sending
	chs := make([]chan TypedEvent, 0, len(subs))
	for ch := range subs {
		chs = append(chs, ch)
	}
	b.mu.RUnlock()
	for _, ch := range chs {
		select {
		case ch <- payload:
		case <-ctx.Done():
			return
		default:
			// drop if subscriber is slow
		}
	}
}

func (b *bus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	for topic, subs := range b.topics {
		for ch := range subs {
			close(ch)
		}
		delete(b.topics, topic)
	}
	b.mu.Unlock()
}
