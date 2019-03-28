package memorybus

import "github.com/grafana/devtools/pkg/streams"

type StreamSubscription struct {
	Topics      []string
	Ready       chan bool
	Streams     streams.ReadableCollection
	SubscribeFn streams.SubscribeFunc
}

func NewStreamSubscription(topics []string, subscribeFn streams.SubscribeFunc) *StreamSubscription {
	return &StreamSubscription{
		Topics:      topics,
		Ready:       make(chan bool),
		Streams:     streams.ReadableCollection{},
		SubscribeFn: subscribeFn,
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
	ss.Streams = append(ss.Streams, stream)
	ss.tryMarkAsReady()
}

func (ss *StreamSubscription) tryMarkAsReady() {
	if len(ss.Streams) == len(ss.Topics) {
		ss.Ready <- true
		close(ss.Ready)
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
