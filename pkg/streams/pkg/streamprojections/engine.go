package streamprojections

import (
	"github.com/grafana/devtools/pkg/streams/pkg/streams"
)

type FilterFunc func(msg interface{}) bool
type ReduceFunc func(accumulator interface{}, currentValue interface{}) interface{}
type SplitToStreamsFunc func(s []ProjectionState) map[string][]ProjectionState

type StreamProjectionEngine interface {
	Register(streamProjection *StreamProjection)
}

type streamProjectionEngine struct {
	streamingEngine streams.Engine
	persister       StreamPersister
	projections     map[string]Projection
}

func New(streamingEngine streams.Engine, persister StreamPersister) StreamProjectionEngine {
	return &streamProjectionEngine{
		streamingEngine: streamingEngine,
		persister:       persister,
		projections:     map[string]Projection{},
	}
}

func (e *streamProjectionEngine) Register(streamProjection *StreamProjection) {
	if streamProjection.PersistTo != "" {
		topic := "persist_to_" + streamProjection.PersistTo
		if streamProjection.ToStreams != nil {
			oldToStreams := streamProjection.ToStreams
			streamProjection.ToStreams = func(state []ProjectionState) map[string][]ProjectionState {
				streams := oldToStreams(state)
				streams[topic] = state
				return streams
			}
		} else {
			streamProjection.ToStreams = func(state []ProjectionState) map[string][]ProjectionState {
				return map[string][]ProjectionState{
					topic: state,
				}
			}
		}
		e.persister.Register(streamProjection.PersistTo, streamProjection.PersistObject)
		e.streamingEngine.Subscribe([]string{topic}, func(p streams.Publisher, stream streams.Readable) {
			e.persister.Persist(streamProjection.PersistTo, stream)
		})
	}
	e.streamingEngine.Subscribe(streamProjection.createSubscriber())
}

func FromStream(name string) *StreamProjectionBuilder {
	return newStreamProjectionBuilder().fromStream(name)
}

func FromStreams(names ...string) *StreamProjectionBuilder {
	b := newStreamProjectionBuilder()
	b.fromStreams = names
	return b
}

type StreamProjection struct {
	FromStreams   []string
	ToStreams     SplitToStreamsFunc
	Projection    Projection
	PersistTo     string
	PersistObject interface{}
}

func (sp *StreamProjection) createSubscriber() ([]string, streams.SubscribeFunc) {
	subscribeFn := func(publisher streams.Publisher, stream streams.Readable) {
		state := sp.Projection.Run(stream)

		if sp.ToStreams != nil {
			outputStreams := sp.ToStreams(state)

			for topic, items := range outputStreams {
				go func(topic string, items []ProjectionState) {
					out := make(chan streams.T, 1)
					publisher.Publish(topic, out)

					for _, item := range items {
						out <- item
					}

					close(out)
				}(topic, items)
			}
		}
	}
	return sp.FromStreams, subscribeFn
}

func newStreamProjection(fromStreams []string, toStreamsFn SplitToStreamsFunc, persistTo string, persistObj interface{}, p Projection) *StreamProjection {
	return &StreamProjection{
		FromStreams:   fromStreams,
		ToStreams:     toStreamsFn,
		Projection:    p,
		PersistTo:     persistTo,
		PersistObject: persistObj,
	}
}
