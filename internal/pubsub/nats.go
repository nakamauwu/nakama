package pubsub

import "github.com/nats-io/go-nats"

// Nats implementation.
type Nats struct {
	Conn *nats.Conn
}

// Pub publishes to the given topic.
func (ps *Nats) Pub(topic string, b []byte) error {
	return ps.Conn.Publish(topic, b)
}

// Sub subscribes to the given topic.
func (ps *Nats) Sub(topic string, fn func([]byte)) (func() error, error) {
	sub, err := ps.Conn.Subscribe(topic, func(m *nats.Msg) {
		fn(m.Data)
	})
	if err != nil {
		return nil, err
	}
	return sub.Unsubscribe, nil
}
