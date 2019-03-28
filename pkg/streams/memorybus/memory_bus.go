package memorybus

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/log"
)

type InMemoryBus struct {
	logger         log.Logger
	Subscriptions  StreamSubscriptionCollection
	subscriptionMu sync.RWMutex
	started        bool
}

func New() *InMemoryBus {
	return &InMemoryBus{
		logger:        log.New(),
		Subscriptions: StreamSubscriptionCollection{},
	}
}

func (bus *InMemoryBus) SetLogger(logger log.Logger) {
	bus.logger = logger.New("logger", "memory-bus")
}

func (bus *InMemoryBus) Subscribe(topics []string, fn streams.SubscribeFunc) error {
	if bus.started {
		return fmt.Errorf("you cannot subscribe after bus have been started")
	}

	bus.subscriptionMu.Lock()
	bus.Subscriptions = append(bus.Subscriptions, NewStreamSubscription(topics, fn))
	bus.subscriptionMu.Unlock()

	bus.logger.Debug("subscription added", "topics", strings.Join(topics, ","))

	return nil
}

func (bus *InMemoryBus) Publish(topic string, stream streams.Readable) error {
	bus.subscriptionMu.RLock()
	defer bus.subscriptionMu.RUnlock()
	subscriptionCount := bus.Subscriptions.countByTopic(topic)

	if subscriptionCount == 0 {
		bus.logger.Debug("no subscribers for published topic, draining messages in stream", "topic", topic)
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
			start := time.Now()
			bus.logger.Debug("starting to publish messages to subscriber...", "topics", strings.Join(cs.Topics, ","))
			in, out := streams.New()

			go func() {
				cs.SubscribeFn(bus, in)
				bus.logger.Debug("sending of messages to subscriber done", "topics", strings.Join(cs.Topics, ","), "took", time.Since(start))
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
