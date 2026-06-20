package realtime

import (
	"context"
	"sync"
	"time"
)

type MemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Event]struct{}
}

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{subscribers: make(map[string]map[chan Event]struct{})}
}

func (b *MemoryBus) Name() string { return "memory" }

func (b *MemoryBus) Publish(_ context.Context, topic string, data []byte) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := Event{Topic: topic, Data: append([]byte(nil), data...), At: time.Now()}
	for ch := range b.subscribers[topic] {
		select {
		case ch <- event:
		default:
			// Slow browser streams should not block the request path.
		}
	}
	return nil
}

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

func (b *MemoryBus) Close() error { return nil }
