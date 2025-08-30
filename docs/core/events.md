# Events Module Documentation

## 1. Introduction to the Events Module
The `events` package provides a minimal publish-subscribe (pub/sub) event bus implementation using Go channels. It is designed to be simple, lightweight, and dependency-free, facilitating inter-component communication within the Acacia application.

## 2. Key Concepts

### 2.1. Bus Interface
The `Bus` interface defines the contract for the event bus, allowing components to subscribe to topics, publish events, and close the bus. Topics are identified by strings, and event payloads can be of any type.

### 2.2. `TypedEvent` Interface
The `TypedEvent` interface encourages type-safe event payloads by requiring events to provide a string identifier for their type.

**Methods:**
*   `EventType() string`: Returns a string identifier for the event type.

### 2.3. Bus Interface
The `Bus` interface defines the contract for the event bus, allowing components to subscribe to topics, publish events, and close the bus. Topics are identified by strings, and event payloads must implement the `TypedEvent` interface.

**Methods:**
*   `Subscribe(topic string) (<-chan TypedEvent, func(), error)`: Subscribes to a given `topic`. Returns a buffered receive-only channel (capacity 16) for events, a `cancel` function to unsubscribe cleanly, and an error if the bus is already closed. The cancel function safely removes the subscription and closes the channel.
*   `Publish(ctx context.Context, topic string, payload TypedEvent)`: Publishes a `payload` (which must be a `TypedEvent`) to a specific `topic`. Events are sent to all active subscribers concurrently. If a subscriber is slow or the channel is full, the event is dropped to prevent blocking the publisher. The `context.Context` can be used for cancellation during the publish operation.
*   `Close()`: Gracefully closes the event bus, marking it as closed and preventing new subscriptions. All existing subscriptions are cleaned up, channels are closed, and subscribers are notified. Safe to call multiple times.

### 2.4. `bus` Struct
The `bus` struct is the concrete implementation of the `Bus` interface. It manages the mapping of topics to their respective subscribers using a `sync.RWMutex` for concurrency safety.

### 2.5. New Function
`New() Bus`
*   Returns a new instance of the event bus.

## 3. Usage Examples

### Defining a Custom Event
```go
package main

import "fmt"

// UserLoggedInEvent implements the events.TypedEvent interface
type UserLoggedInEvent struct {
	Username string
	Timestamp time.Time
}

func (e UserLoggedInEvent) EventType() string {
	return "UserLoggedIn"
}

// SystemStatusEvent implements the events.TypedEvent interface
type SystemStatusEvent struct {
	Status  string
	Message string
}

func (e SystemStatusEvent) EventType() string {
	return "SystemStatus"
}
```

### Creating and Using the Event Bus
```go
package main

import (
	"acacia/core/events"
	"context"
	"fmt"
	"time"
)

func main() {
	eventBus := events.New()
	defer eventBus.Close() // Ensure the bus is closed when main exits

	// Subscriber 1: Logs all messages on "user_events" topic
	userEventsCh1, unsubscribe1, err := eventBus.Subscribe("user_events")
	if err != nil {
		fmt.Println("Error subscribing:", err)
		return
	}
	defer unsubscribe1() // Ensure unsubscribe is called

	go func() {
		for event := range userEventsCh1 {
			// Type assertion to the concrete event type
			if userEvent, ok := event.(UserLoggedInEvent); ok {
				fmt.Printf("Subscriber 1 received user event: %s at %v\n", userEvent.Username, userEvent.Timestamp)
			} else {
				fmt.Printf("Subscriber 1 received unknown user event type: %T\n", event)
			}
		}
		fmt.Println("Subscriber 1 channel closed.")
	}()

	// Subscriber 2: Logs all messages on "system_events" topic
	systemEventsCh, unsubscribe2, err := eventBus.Subscribe("system_events")
	if err != nil {
		fmt.Println("Error subscribing:", err)
		return
	}
	defer unsubscribe2()

	go func() {
		for event := range systemEventsCh {
			if sysEvent, ok := event.(SystemStatusEvent); ok {
				fmt.Printf("Subscriber 2 received system event: %s - %s\n", sysEvent.Status, sysEvent.Message)
			} else {
				fmt.Printf("Subscriber 2 received unknown system event type: %T\n", event)
			}
		}
		fmt.Println("Subscriber 2 channel closed.")
	}()

	// Subscriber 3: Also logs "user_events", but will unsubscribe later
	userEventsCh2, unsubscribe3, err := eventBus.Subscribe("user_events")
	if err != nil {
		fmt.Println("Error subscribing:", err)
		return
	}
	go func() {
		for event := range userEventsCh2 {
			if userEvent, ok := event.(UserLoggedInEvent); ok {
				fmt.Printf("Subscriber 3 received user event: %s at %v\n", userEvent.Username, userEvent.Timestamp)
			}
		}
		fmt.Println("Subscriber 3 channel closed.")
	}()


	// Publish some events
	fmt.Println("Publishing events...")
	eventBus.Publish(context.Background(), "user_events", UserLoggedInEvent{Username: "Alice", Timestamp: time.Now()})
	eventBus.Publish(context.Background(), "system_events", SystemStatusEvent{Status: "OK", Message: "Health check passed"})
	eventBus.Publish(context.Background(), "user_events", UserLoggedInEvent{Username: "Bob", Timestamp: time.Now()})

	time.Sleep(100 * time.Millisecond) // Give goroutines time to process

	// Unsubscribe Subscriber 3
	fmt.Println("Unsubscribing Subscriber 3 from user_events.")
	unsubscribe3()

	// Publish more events after unsubscribe
	eventBus.Publish(context.Background(), "user_events", UserLoggedInEvent{Username: "Charlie", Timestamp: time.Now()})
	eventBus.Publish(context.Background(), "system_events", SystemStatusEvent{Status: "INFO", Message: "Database backup complete"})

	time.Sleep(100 * time.Millisecond) // Give goroutines time to process

	fmt.Println("Main function ending.")
}
```
