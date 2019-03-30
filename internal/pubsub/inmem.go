package pubsub

import "sync"

type client struct {
	topic string
	fn    func([]byte)
}

// Inmem implementation.
type Inmem struct {
	clients sync.Map
}

// Pub publishes to the given topic.
func (ps *Inmem) Pub(topic string, b []byte) error {
	ps.clients.Range(func(key, _ interface{}) bool {
		client := key.(*client)
		if client.topic == topic {
			go client.fn(b)
		}
		return true
	})
	return nil
}

// Sub subscribes to the given topic.
func (ps *Inmem) Sub(topic string, fn func([]byte)) (unsub func() error, err error) {
	c := &client{topic, fn}
	ps.clients.Store(c, nil)
	return func() error {
		ps.clients.Delete(c)
		return nil
	}, nil
}
