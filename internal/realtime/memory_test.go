package realtime

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBusPublishSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := NewMemoryBus()
	ch, unsubscribe, err := bus.Subscribe(ctx, "todos")
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()

	if err := bus.Publish(ctx, "todos", []byte("changed")); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-ch:
		if event.Topic != "todos" || string(event.Data) != "changed" {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
