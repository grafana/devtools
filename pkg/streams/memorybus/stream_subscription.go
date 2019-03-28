package memorybus

import "github.com/grafana/devtools/pkg/streams"

type StreamSubscription struct {
	Topics           []string
	Ready            chan bool
	SubscribeFn      streams.SubscribeFunc
	PublishedStreams chan streams.Readable
	ReadyStreams     int
}

func NewStreamSubscription(topics []string, subscribeFn streams.SubscribeFunc) *StreamSubscription {
	return &StreamSubscription{
		Topics:           topics,
		Ready:            make(chan bool),
		SubscribeFn:      subscribeFn,
		PublishedStreams: make(chan streams.Readable),
	}
}

func (ss *StreamSubscription) hasTopic(topic string) bool {
	for _, t := range ss.Topics {
		if t == topic {
			return true
		}
	}

	return false
}

func (ss *StreamSubscription) addReadyStream(stream streams.Readable) {
	ss.ReadyStreams++
	if ss.ReadyStreams == 1 {
		ss.Ready <- true
		close(ss.Ready)
	}

	ss.PublishedStreams <- stream

	if ss.ReadyStreams == len(ss.Topics) {
		close(ss.PublishedStreams)
	}
}

type StreamSubscriptionCollection []*StreamSubscription

func (subscriptions StreamSubscriptionCollection) countByTopic(topic string) int {
	count := 0

	for _, subscription := range subscriptions {
		if subscription.hasTopic(topic) {
			count++
		}
	}

	return count
}
