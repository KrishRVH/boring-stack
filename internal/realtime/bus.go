package realtime

import "context"

// TopicTodosChanged broadcasts changes that require a fresh todo snapshot.
const TopicTodosChanged = "todos.changed"

// Event is an ephemeral message received from a Bus.
type Event struct {
	Topic string
	Data  []byte
}

// Bus publishes and subscribes to ephemeral application events.
type Bus interface {
	Name() string
	Publish(ctx context.Context, topic string, data []byte) error
	// Subscribe returns a channel owned by the bus. Consumers should stop by
	// canceling ctx or calling unsubscribe; implementations do not promise to
	// close the returned channel.
	Subscribe(ctx context.Context, topic string) (<-chan Event, func(), error)
	Close() error
}
