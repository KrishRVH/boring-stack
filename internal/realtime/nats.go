package realtime

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	natsDrainTimeout = 2 * time.Second
	natsCloseTimeout = natsDrainTimeout + 6*time.Second
)

type NATSBus struct {
	conn   *nats.Conn
	closed <-chan struct{}
}

func NewNATSBus(url string) (*NATSBus, error) {
	closed := make(chan struct{})
	var closeOnce sync.Once
	conn, err := nats.Connect(
		url,
		nats.Name("boring-stack"),
		nats.Timeout(3*time.Second),
		nats.DrainTimeout(natsDrainTimeout),
		nats.ClosedHandler(func(*nats.Conn) {
			closeOnce.Do(func() { close(closed) })
		}),
	)
	if err != nil {
		return nil, err
	}
	return &NATSBus{conn: conn, closed: closed}, nil
}

func (b *NATSBus) Name() string { return "nats" }

func (b *NATSBus) Publish(ctx context.Context, topic string, data []byte) error {
	if err := b.conn.Publish(topic, data); err != nil {
		return err
	}
	return b.flush(ctx)
}

func (b *NATSBus) Subscribe(ctx context.Context, topic string) (<-chan Event, func(), error) {
	ch := make(chan Event, 16)
	done := make(chan struct{})
	sub, err := b.conn.Subscribe(topic, func(msg *nats.Msg) {
		event := Event{Topic: msg.Subject, Data: append([]byte(nil), msg.Data...), At: time.Now()}
		select {
		case <-done:
			return
		case ch <- event:
		default:
			// Drop stale events for slow browser streams. The next broadcast will patch a full snapshot.
		}
	})
	if err != nil {
		return nil, nil, err
	}
	if err := b.flush(ctx); err != nil {
		close(done)
		_ = sub.Unsubscribe()
		return nil, nil, err
	}

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			close(done)
			_ = sub.Unsubscribe()
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

func (b *NATSBus) Close() error {
	if err := b.conn.Drain(); err != nil {
		if errors.Is(err, nats.ErrConnectionClosed) {
			return nil
		}
		return err
	}

	timer := time.NewTimer(natsCloseTimeout)
	defer timer.Stop()
	select {
	case <-b.closed:
		return b.conn.LastError()
	case <-timer.C:
		b.conn.Close()
		return errors.New("nats drain timed out")
	}
}

func (b *NATSBus) flush(ctx context.Context) error {
	flushCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return b.conn.FlushWithContext(flushCtx)
}
