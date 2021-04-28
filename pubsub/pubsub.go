package pubsub

// Publisher interface.
type Publisher interface {
	Pub(topic string, data []byte) error
}

// Subscriber interface.
type Subscriber interface {
	Sub(topic string, cb func(data []byte)) (unsub func() error, err error)
}

// PubSub interface.
type PubSub interface {
	Publisher
	Subscriber
}
