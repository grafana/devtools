package memorybus

import (
	"fmt"
	"sync"

	"github.com/grafana/devtools/pkg/streams"
)

type InMemoryBus struct {
	streams.Bus
	Subscriptions  StreamSubscriptionCollection
	subscriptionMu sync.RWMutex
	started        bool
}

func New() *InMemoryBus {
	return &InMemoryBus{
		Subscriptions: StreamSubscriptionCollection{},
	}
}

func (bus *InMemoryBus) Subscribe(topics []string, fn streams.SubscribeFunc) error {
	if bus.started {
		return fmt.Errorf("you cannot subscribe after bus have been started")
	}

	bus.subscriptionMu.Lock()
	bus.Subscriptions = append(bus.Subscriptions, NewStreamSubscription(topics, fn))
	bus.subscriptionMu.Unlock()

	return nil
}

func (bus *InMemoryBus) Publish(topic string, stream streams.Readable) error {
	bus.subscriptionMu.RLock()
	defer bus.subscriptionMu.RUnlock()
	subscriptionCount := bus.Subscriptions.countByTopic(topic)

	if subscriptionCount == 0 {
		go func() {
			stream.Drain()
		}()
		return nil
	}

	streams := stream.Split(subscriptionCount)
	streamIndex := 0

	for _, subscription := range bus.Subscriptions {
		if subscription.hasTopic(topic) {
			subscription.addReadyStream(streams[streamIndex])
			streamIndex++
		}
	}

	return nil
}

func (bus *InMemoryBus) Start() <-chan bool {
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(len(bus.Subscriptions))

	for _, cs := range bus.Subscriptions {
		go func(cs *StreamSubscription) {
			<-cs.Ready
			in, out := streams.New()

			go func() {
				cs.SubscribeFn(bus, in)
				wg.Done()
			}()

			var wg2 sync.WaitGroup
			wg2.Add(len(cs.Topics))

			go func() {
				for publishedStream := range cs.PublishedStreams {
					go func(publishedStream streams.Readable) {
						for msg := range publishedStream {
							out <- msg
						}
						wg2.Done()
					}(publishedStream)
				}
			}()

			wg2.Wait()
			close(out)
		}(cs)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}
