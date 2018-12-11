package streams

import "sync"

type SubscribeFunc func(p Publisher, stream Readable)
type Subscriber interface {
	Subscribe(topics []string, fn SubscribeFunc)
}

type Publisher interface {
	Publish(topic string, stream Readable)
}

type Engine interface {
	Subscriber
	Publisher
	Start() <-chan bool
}

type streamSubscription struct {
	topics      []string
	ready       chan bool
	streams     []Readable
	subscribeFn SubscribeFunc
}

func newStreamSubscription(topics []string, subscribeFn SubscribeFunc) *streamSubscription {
	return &streamSubscription{
		topics:      topics,
		ready:       make(chan bool),
		streams:     []Readable{},
		subscribeFn: subscribeFn,
	}
}

type engine struct {
	subscriptions []*streamSubscription
}

func New() Engine {
	return &engine{
		subscriptions: []*streamSubscription{},
	}
}

func (e *engine) Subscribe(topics []string, fn SubscribeFunc) {
	e.subscriptions = append(e.subscriptions, newStreamSubscription(topics, fn))
}

func (e *engine) Publish(topic string, stream Readable) {
	totalStreamCount := 0

	for _, cs := range e.subscriptions {
		for _, t := range cs.topics {
			if t == topic {
				totalStreamCount++
				break
			}
		}
	}

	streams := Split(totalStreamCount, stream)
	streamIndex := 0
	subscriberExists := false

	for _, cs := range e.subscriptions {
		for _, t := range cs.topics {
			if t == topic {
				cs.streams = append(cs.streams, streams[streamIndex])
				subscriberExists = true
				if len(cs.streams) == len(cs.topics) {
					cs.ready <- true
					close(cs.ready)
				}
				streamIndex++
				break
			}
		}
	}

	if !subscriberExists {
		go func() {
			stream.Drain()
		}()
	}
}

func (e *engine) Start() <-chan bool {
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(len(e.subscriptions))

	for _, cs := range e.subscriptions {
		go func(cs *streamSubscription) {
			defer wg.Done()
			<-cs.ready
			combinedStream := Combine(cs.streams)
			cs.subscribeFn(e, combinedStream)
		}(cs)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}
