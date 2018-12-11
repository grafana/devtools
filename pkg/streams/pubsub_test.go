package streams

import (
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPubSub(t *testing.T) {
	Convey("Test pub/sub", t, func() {
		e := New()

		Convey("Publish to topic with no subscriber should drain messages", func() {
			in, out := NewReadableWritable()
			go func() {
				for n := 0; n < 10; n++ {
					out <- n
				}
				close(out)
			}()

			startEngineAndRun(e, func() {
				e.Publish("stream-1", in)
			})
		})

		Convey("Publish to topic with one subscriber should send all messages to subscriber", func() {
			receivedMessages := []T{}
			e.Subscribe([]string{"stream-1"}, func(p Publisher, stream Readable) {
				for msg := range stream {
					receivedMessages = append(receivedMessages, msg)
				}
			})

			in, out := NewReadableWritable()
			go func() {
				for n := 0; n < 10; n++ {
					out <- n
				}
				close(out)
			}()

			startEngineAndRun(e, func() {
				e.Publish("stream-1", in)
			})

			So(receivedMessages, ShouldResemble, []T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
		})

		Convey("Publish to topic with two subscribers should send all messages to both subscribers", func() {
			receivedMessages1 := []T{}
			e.Subscribe([]string{"stream-1"}, func(p Publisher, stream Readable) {
				for msg := range stream {
					receivedMessages1 = append(receivedMessages1, msg)
				}
			})

			receivedMessages2 := []T{}
			e.Subscribe([]string{"stream-1"}, func(p Publisher, stream Readable) {
				for msg := range stream {
					receivedMessages2 = append(receivedMessages2, msg)
				}
			})

			in, out := NewReadableWritable()
			go func() {
				for n := 0; n < 10; n++ {
					out <- n
				}
				close(out)
			}()

			startEngineAndRun(e, func() {
				e.Publish("stream-1", in)
			})

			So(receivedMessages1, ShouldResemble, []T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
			So(receivedMessages2, ShouldResemble, []T{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
		})
	})
}

func startEngineAndRun(e Engine, fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-e.Start()
		wg.Done()
	}()

	fn()
	wg.Wait()
}
