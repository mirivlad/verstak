// Package events provides an in-process event bus for plugin and core communication.
package events

import (
	"sync"
)

// Handler is a function that processes an event.
type Handler func(event Event)

// Event represents a named event with a payload.
type Event struct {
	Name      string      `json:"name"`
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Bus is a simple in-process event bus.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for an event name.
func (b *Bus) Subscribe(event string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[event] = append(b.handlers[event], handler)
}

// HasSubscribers reports whether an event has at least one registered handler.
func (b *Bus) HasSubscribers(event string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[event]) > 0
}

// Unsubscribe removes all handlers for a plugin (matched by prefix or exact).
// For now, a simple version: clear all handlers for a given event name.
func (b *Bus) Unsubscribe(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.handlers, event)
}

// Publish dispatches an event to all registered handlers.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers, ok := b.handlers[event.Name]
	b.mu.RUnlock()

	if !ok {
		return
	}

	for _, handler := range handlers {
		handler(event)
	}
}
