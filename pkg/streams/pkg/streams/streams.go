package streams

import (
	"context"
	"sync"
)

type T interface{}
type Readable <-chan T
type Writable chan<- T
type ReadableCollection []Readable
type WritableCollection []Writable

func NewReadableWritable() (Readable, Writable) {
	ch := make(chan T)
	return ch, ch
}

func NewReadableWritableCollection(size int) (ReadableCollection, WritableCollection) {
	readables := make(ReadableCollection, size)
	writables := make(WritableCollection, size)

	for n := 0; n < size; n++ {
		ch := make(chan T)
		readables[n] = ch
		writables[n] = ch
	}

	return readables, writables
}

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

type stream struct {
	name   string
	ready  chan bool
	stream Readable
}

func newStream(name string) *stream {
	return &stream{
		name:   name,
		ready:  make(chan bool),
		stream: make(Readable, 1),
	}
}

type combinedStream struct {
	topics      []string
	ready       chan bool
	streams     []Readable
	subscribeFn SubscribeFunc
}

func newCombinedStream(topics []string, subscribeFn SubscribeFunc) *combinedStream {
	return &combinedStream{
		topics:      topics,
		ready:       make(chan bool),
		streams:     []Readable{},
		subscribeFn: subscribeFn,
	}
}

type engine struct {
	ctx               context.Context
	streamSubscribers map[string][]SubscribeFunc
	streams           map[string]*stream
	combinedStreams   []*combinedStream
}

func NewWithContext(ctx context.Context) Engine {
	return &engine{
		streamSubscribers: make(map[string][]SubscribeFunc, 0),
		streams:           make(map[string]*stream),
		combinedStreams:   []*combinedStream{},
	}
}

func New() Engine {
	return NewWithContext(context.Background())
}

func (e *engine) Subscribe(topics []string, fn SubscribeFunc) {
	if len(topics) == 1 {
		topic := topics[0]
		if _, ok := e.streamSubscribers[topic]; !ok {
			e.streamSubscribers[topic] = make([]SubscribeFunc, 0)
		}

		e.streamSubscribers[topic] = append(e.streamSubscribers[topic], fn)

		if _, ok := e.streams[topic]; !ok {
			e.streams[topic] = newStream(topic)
		}
	} else {
		e.combinedStreams = append(e.combinedStreams, newCombinedStream(topics, fn))
	}
}

func (e *engine) Publish(topic string, stream Readable) {
	totalStreamCount := 0

	for _, cs := range e.combinedStreams {
		for _, t := range cs.topics {
			if t == topic {
				totalStreamCount++
				break
			}
		}
	}

	if _, ok := e.streams[topic]; ok {
		totalStreamCount++
	}

	streams := e.split(totalStreamCount, stream)

	n := 0
	subscriberExists := false
	if s, ok := e.streams[topic]; ok {
		s.stream = streams[0]
		s.ready <- true
		close(s.ready)
		subscriberExists = true
		n++
	}

	for _, cs := range e.combinedStreams {
		for _, t := range cs.topics {
			if t == topic {
				cs.streams = append(cs.streams, streams[n])
				subscriberExists = true
				if len(cs.streams) == len(cs.topics) {
					cs.ready <- true
					close(cs.ready)
				}
				n++
				break
			}
		}
	}

	if !subscriberExists {
		go func() {
			for range stream {
			}
		}()
	}
}

func (e *engine) Start() <-chan bool {
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(len(e.streamSubscribers) + len(e.combinedStreams))

	for k, v := range e.streams {
		go func(topic string, s *stream) {
			defer wg.Done()
			<-s.ready
			<-e.parallel(e.streamSubscribers[topic], s.stream)
		}(k, v)
	}

	for _, cs := range e.combinedStreams {
		go func(cs *combinedStream) {
			defer wg.Done()
			<-cs.ready
			combinedStream := e.combine(cs.streams)
			cs.subscribeFn(e, combinedStream)
		}(cs)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}

func (e *engine) combine(streams []Readable) Readable {
	out := make(chan T)
	var wg sync.WaitGroup
	wg.Add(len(streams))

	for _, stream := range streams {
		go func(s Readable) {
			for msg := range s {
				out <- msg
			}
			wg.Done()
		}(stream)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (e *engine) split(streams int, stream Readable) []chan T {
	out := make([]chan T, streams)
	for n := 0; n < streams; n++ {
		out[n] = make(chan T)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for evt := range stream {
			for n := 0; n < streams; n++ {
				c := out[n]
				c <- evt
			}
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()

		for n := 0; n < streams; n++ {
			close(out[n])
		}
	}()

	return out
}

func (e *engine) parallel(subscribers []SubscribeFunc, stream Readable) <-chan bool {
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(len(subscribers))

	streams := e.split(len(subscribers), stream)

	for n := 0; n < len(subscribers); n++ {
		go func(fn SubscribeFunc, stream Readable) {
			fn(e, stream)
			wg.Done()
		}(subscribers[n], streams[n])
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}
