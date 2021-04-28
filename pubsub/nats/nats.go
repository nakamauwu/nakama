package nats

import "github.com/nats-io/nats.go"

// PubSub implementation using NATS server.
type PubSub struct {
	Conn *nats.Conn
}

// Pub publishes some data to the given topic.
func (ps *PubSub) Pub(topic string, data []byte) error {
	return ps.Conn.Publish(topic, data)
}

// Sub subscribes the given callback function to the interested topic.
func (ps *PubSub) Sub(topic string, cb func(data []byte)) (unsub func() error, err error) {
	s, err := ps.Conn.Subscribe(topic, func(msg *nats.Msg) {
		cb(msg.Data)
	})
	if err != nil {
		return nil, err
	}
	return s.Unsubscribe, nil
}
