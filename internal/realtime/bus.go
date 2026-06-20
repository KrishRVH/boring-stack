package realtime

import (
	"context"
	"time"
)

const TopicTodosChanged = "todos.changed"

type Event struct {
	Topic string
	Data  []byte
	At    time.Time
}

type Bus interface {
	Name() string
	Publish(ctx context.Context, topic string, data []byte) error
	// Subscribe returns a channel owned by the bus. Consumers should stop by
	// canceling ctx or calling unsubscribe; implementations do not promise to
	// close the returned channel.
	Subscribe(ctx context.Context, topic string) (<-chan Event, func(), error)
	Close() error
}
