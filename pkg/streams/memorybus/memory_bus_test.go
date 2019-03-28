package memorybus

import (
	"sync"
	"testing"
	"time"

	"github.com/grafana/devtools/pkg/streams"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryBus(t *testing.T) {
	Convey("Test in memory bus", t, func() {
		bus := New()

		Convey("Publish to topic with no subscriber should drain messages", func() {
			startBusAndRun(bus, func() {
				bus.Publish("stream-1", streams.NewFrom("msg"))
			})
		})

		Convey("Publish to topic with one subscriber should send all messages to subscriber", func() {
			receivedMessages := []streams.T{}
			bus.Subscribe([]string{"stream-1"}, func(p streams.Publisher, stream streams.Readable) {
				for msg := range stream {
					receivedMessages = append(receivedMessages, msg)
				}
			})

			startBusAndRun(bus, func() {
				bus.Publish("stream-1", streams.NewFromRange(0, 9))
			})

			So(receivedMessages, ShouldResemble, []streams.T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
		})

		Convey("Publish to topic with two subscribers should send all messages to both subscribers", func() {
			receivedMessages1 := []streams.T{}
			bus.Subscribe([]string{"stream-1"}, func(p streams.Publisher, stream streams.Readable) {
				for msg := range stream {
					receivedMessages1 = append(receivedMessages1, msg)
				}
			})

			receivedMessages2 := []streams.T{}
			bus.Subscribe([]string{"stream-1"}, func(p streams.Publisher, stream streams.Readable) {
				for msg := range stream {
					receivedMessages2 = append(receivedMessages2, msg)
				}
			})

			startBusAndRun(bus, func() {
				bus.Publish("stream-1", streams.NewFromRange(0, 9))
			})

			So(receivedMessages1, ShouldResemble, []streams.T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
			So(receivedMessages2, ShouldResemble, []streams.T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
		})

		Convey("Publish to topic with one subscriber should send messages to subscriber as soon as they're published", func() {
			receivedMessages := []streams.T{}
			bus.Subscribe([]string{"stream-1"}, func(p streams.Publisher, stream streams.Readable) {
				for msg := range stream {
					receivedMessages = append(receivedMessages, msg)
				}
			})

			in, out := streams.New()

			startBusAndRun(bus, func() {
				bus.Publish("stream-1", in)
				out <- 0
				out <- 1
				out <- 2
				out <- 3
				out <- 4
				<-time.Tick(10 * time.Millisecond)
				So(receivedMessages, ShouldResemble, []streams.T{0, 1, 2, 3, 4})
				out <- 5
				out <- 6
				out <- 7
				out <- 8
				out <- 9
				close(out)
			})

			So(receivedMessages, ShouldResemble, []streams.T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
		})
	})
}

func startBusAndRun(bus streams.Bus, fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-bus.Start()
		wg.Done()
	}()

	fn()
	wg.Wait()
}
