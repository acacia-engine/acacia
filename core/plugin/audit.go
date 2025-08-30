package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// AuditLoggerImpl implements security event logging
type AuditLoggerImpl struct {
	mu        sync.RWMutex
	events    []SecurityEventData
	logPath   string
	maxEvents int
	flushSize int
	eventChan chan SecurityEventData
	shutdown  chan struct{}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string, maxEvents int) (*AuditLoggerImpl, error) {
	logger := &AuditLoggerImpl{
		events:    make([]SecurityEventData, 0),
		logPath:   logPath,
		maxEvents: maxEvents,
		flushSize: 100, // Flush every 100 events
		eventChan: make(chan SecurityEventData, 1000),
		shutdown:  make(chan struct{}),
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Start background logger
	go logger.backgroundLogger()

	return logger, nil
}

// LogEvent logs a security event
func (l *AuditLoggerImpl) LogEvent(event SecurityEventData) error {
	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Send to background logger
	select {
	case l.eventChan <- event:
		return nil
	case <-time.After(time.Second):
		// Timeout - log synchronously as fallback
		return l.logEventSync(event)
	}
}

// QueryEvents retrieves security events for a plugin
func (l *AuditLoggerImpl) QueryEvents(pluginName string, limit int) ([]SecurityEventData, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var matchingEvents []SecurityEventData
	for _, event := range l.events {
		if event.PluginName == pluginName {
			matchingEvents = append(matchingEvents, event)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(matchingEvents, func(i, j int) bool {
		return matchingEvents[i].Timestamp.After(matchingEvents[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(matchingEvents) > limit {
		matchingEvents = matchingEvents[:limit]
	}

	return matchingEvents, nil
}

// GetAllEvents returns all security events (admin function)
func (l *AuditLoggerImpl) GetAllEvents(limit int) ([]SecurityEventData, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	events := make([]SecurityEventData, len(l.events))
	copy(events, l.events)

	// Sort by timestamp (newest first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}

// GetEventsByType returns events of a specific type
func (l *AuditLoggerImpl) GetEventsByType(eventType SecurityEvent, limit int) ([]SecurityEventData, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var matchingEvents []SecurityEventData
	for _, event := range l.events {
		if event.EventType == eventType {
			matchingEvents = append(matchingEvents, event)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(matchingEvents, func(i, j int) bool {
		return matchingEvents[i].Timestamp.After(matchingEvents[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(matchingEvents) > limit {
		matchingEvents = matchingEvents[:limit]
	}

	return matchingEvents, nil
}

// GetEventsByTimeRange returns events within a time range
func (l *AuditLoggerImpl) GetEventsByTimeRange(start, end time.Time, limit int) ([]SecurityEventData, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var matchingEvents []SecurityEventData
	for _, event := range l.events {
		if event.Timestamp.After(start) && event.Timestamp.Before(end) {
			matchingEvents = append(matchingEvents, event)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(matchingEvents, func(i, j int) bool {
		return matchingEvents[i].Timestamp.After(matchingEvents[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(matchingEvents) > limit {
		matchingEvents = matchingEvents[:limit]
	}

	return matchingEvents, nil
}

// GetSecuritySummary returns a summary of security events
func (l *AuditLoggerImpl) GetSecuritySummary(hours int) (map[string]int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	summary := make(map[string]int)

	for _, event := range l.events {
		if event.Timestamp.After(since) {
			key := string(event.EventType)
			summary[key]++
		}
	}

	return summary, nil
}

// backgroundLogger processes events in the background
func (l *AuditLoggerImpl) backgroundLogger() {
	defer l.flushEvents()

	for {
		select {
		case event := <-l.eventChan:
			l.addEvent(event)
		case <-l.shutdown:
			return
		}
	}
}

// addEvent adds an event to the in-memory store
func (l *AuditLoggerImpl) addEvent(event SecurityEventData) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, event)

	// Maintain max events limit
	if len(l.events) > l.maxEvents {
		// Keep most recent events
		l.events = l.events[len(l.events)-l.maxEvents:]
	}

	// Flush periodically
	if len(l.events)%l.flushSize == 0 {
		l.flushEventsLocked()
	}
}

// flushEvents writes events to disk
func (l *AuditLoggerImpl) flushEvents() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushEventsLocked()
}

// flushEventsLocked writes events to disk (with lock held)
func (l *AuditLoggerImpl) flushEventsLocked() {
	if len(l.events) == 0 {
		return
	}

	// Prepare data for writing
	auditData := struct {
		Timestamp time.Time           `json:"timestamp"`
		Events    []SecurityEventData `json:"events"`
		Version   string              `json:"version"`
	}{
		Timestamp: time.Now(),
		Events:    l.events,
		Version:   "1.0",
	}

	data, err := json.MarshalIndent(auditData, "", "  ")
	if err != nil {
		// Log error but don't panic
		fmt.Printf("Audit log marshal error: %v\n", err)
		return
	}

	// Append to log file
	file, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Audit log file error: %v\n", err)
		return
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		fmt.Printf("Audit log write error: %v\n", err)
		return
	}

	if _, err := file.WriteString("\n---\n"); err != nil {
		fmt.Printf("Audit log separator error: %v\n", err)
		return
	}
}

// logEventSync synchronously logs an event (fallback)
func (l *AuditLoggerImpl) logEventSync(event SecurityEventData) error {
	l.addEvent(event)
	l.flushEvents()
	return nil
}

// Shutdown gracefully shuts down the audit logger
func (l *AuditLoggerImpl) Shutdown(ctx context.Context) error {
	close(l.shutdown)

	// Wait for background logger to finish
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Second):
		// Force final flush
		l.flushEvents()
		return nil
	}
}

// GetStats returns audit logger statistics
func (l *AuditLoggerImpl) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return map[string]interface{}{
		"total_events": len(l.events),
		"max_events":   l.maxEvents,
		"flush_size":   l.flushSize,
		"log_path":     l.logPath,
		"queue_size":   len(l.eventChan),
	}
}

// ClearEvents clears all events (admin function)
func (l *AuditLoggerImpl) ClearEvents() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = make([]SecurityEventData, 0)
	return nil
}
