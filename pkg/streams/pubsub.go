package streams

type SubscribeFunc func(p Publisher, stream Readable)

type Subscriber interface {
	Subscribe(topics []string, fn SubscribeFunc) error
}

type Publisher interface {
	Publish(topic string, stream Readable) error
}

type Bus interface {
	Subscriber
	Publisher
	Start() <-chan bool
}
