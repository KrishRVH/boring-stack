package realtime

import (
	"context"
	"sync"
)

// MemoryBus provides in-process ephemeral event fanout.
type MemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Event]struct{}
}

// NewMemoryBus builds an empty MemoryBus.
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{subscribers: make(map[string]map[chan Event]struct{})}
}

// Name returns the bus implementation name.
func (b *MemoryBus) Name() string { return "memory" }

// Publish sends an event to the current in-process subscribers.
func (b *MemoryBus) Publish(_ context.Context, topic string, data []byte) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := Event{Topic: topic, Data: append([]byte(nil), data...)}
	for ch := range b.subscribers[topic] {
		select {
		case ch <- event:
		default:
			// Slow browser streams should not block the request path.
		}
	}
	return nil
}

// Subscribe registers an in-process event subscriber.
func (b *MemoryBus) Subscribe(ctx context.Context, topic string) (<-chan Event, func(), error) {
	ch := make(chan Event, 16)
	done := make(chan struct{})

	b.mu.Lock()
	if b.subscribers[topic] == nil {
		b.subscribers[topic] = make(map[chan Event]struct{})
	}
	b.subscribers[topic][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			close(done)
			b.mu.Lock()
			delete(b.subscribers[topic], ch)
			if len(b.subscribers[topic]) == 0 {
				delete(b.subscribers, topic)
			}
			b.mu.Unlock()
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			unsubscribe()
		case <-done:
		}
	}()

	return ch, unsubscribe, nil
}

// Close releases bus resources.
func (b *MemoryBus) Close() error { return nil }
