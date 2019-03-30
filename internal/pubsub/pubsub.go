package pubsub

// PubSub system.
type PubSub interface {
	// Pub publishes to the given topic.
	Pub(topic string, b []byte) error
	// Sub subscribes to the given topic.
	Sub(topic string, fn func([]byte)) (unsub func() error, err error)
}
