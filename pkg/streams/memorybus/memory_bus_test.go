package memorybus

import (
	"sync"
	"testing"

	"github.com/grafana/devtools/pkg/streams"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryBus(t *testing.T) {
	Convey("Test in memory bus", t, func() {
		e := New()

		Convey("Publish to topic with no subscriber should drain messages", func() {
			startEngineAndRun(e, func() {
				e.Publish("stream-1", streams.NewFrom("msg"))
			})
		})

		Convey("Publish to topic with one subscriber should send all messages to subscriber", func() {
			receivedMessages := []streams.T{}
			e.Subscribe([]string{"stream-1"}, func(p streams.Publisher, stream streams.Readable) {
				for msg := range stream {
					receivedMessages = append(receivedMessages, msg)
				}
			})

			startEngineAndRun(e, func() {
				e.Publish("stream-1", streams.NewFromRange(0, 9))
			})

			So(receivedMessages, ShouldResemble, []streams.T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
		})

		Convey("Publish to topic with two subscribers should send all messages to both subscribers", func() {
			receivedMessages1 := []streams.T{}
			e.Subscribe([]string{"stream-1"}, func(p streams.Publisher, stream streams.Readable) {
				for msg := range stream {
					receivedMessages1 = append(receivedMessages1, msg)
				}
			})

			receivedMessages2 := []streams.T{}
			e.Subscribe([]string{"stream-1"}, func(p streams.Publisher, stream streams.Readable) {
				for msg := range stream {
					receivedMessages2 = append(receivedMessages2, msg)
				}
			})

			startEngineAndRun(e, func() {
				e.Publish("stream-1", streams.NewFromRange(0, 9))
			})

			So(receivedMessages1, ShouldResemble, []streams.T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
			So(receivedMessages2, ShouldResemble, []streams.T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
		})
	})
}

func startEngineAndRun(e streams.Bus, fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-e.Start()
		wg.Done()
	}()

	fn()
	wg.Wait()
}
