package projections

import (
	"strings"
	"time"

	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/log"
)

type FilterFunc func(msg interface{}) bool
type ReduceFunc func(accumulator interface{}, currentValue interface{}) interface{}
type SplitToStreamsFunc func(s []ProjectionState) map[string][]ProjectionState

type StreamProjectionEngine interface {
	SetLogger(logger log.Logger)
	Register(streamProjection *StreamProjection)
}

type streamProjectionEngine struct {
	logger      log.Logger
	bus         streams.Bus
	persister   streams.StreamPersister
	projections map[string]Projection
}

func New(bus streams.Bus, persister streams.StreamPersister) StreamProjectionEngine {
	return &streamProjectionEngine{
		logger:      log.New(),
		bus:         bus,
		persister:   persister,
		projections: map[string]Projection{},
	}
}

func (e *streamProjectionEngine) SetLogger(logger log.Logger) {
	e.logger = logger.New("logger", "stream-projections")
}

func (e *streamProjectionEngine) Register(streamProjection *StreamProjection) {
	e.logger.Debug("registering stream...", "fromStreams", strings.Join(streamProjection.FromStreams, ","))

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
		e.bus.Subscribe([]string{topic}, func(p streams.Publisher, stream streams.Readable) {
			e.logger.Debug("persisting projection stream", "name", streamProjection.PersistTo)
			if err := e.persister.Persist(streamProjection.PersistTo, stream); err != nil {
				e.logger.Error("failed to persist projection stream", "error", err)
			}

			e.logger.Debug("projection stream persisted", "name", streamProjection.PersistTo)
		})
	}
	e.bus.Subscribe(streamProjection.createSubscriber(e.logger))
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

func (sp *StreamProjection) createSubscriber(logger log.Logger) ([]string, streams.SubscribeFunc) {
	subscribeFn := func(publisher streams.Publisher, stream streams.Readable) {
		fromStreams := strings.Join(sp.FromStreams, ", ")
		start := time.Now()
		logger.Debug("running stream projection...", "fromStreams", fromStreams)
		state := sp.Projection.Run(stream)
		logger.Debug("stream projection done", "fromStreams", fromStreams, "took", time.Since(start))

		if sp.ToStreams != nil {
			outputStreams := sp.ToStreams(state)

			for topic, items := range outputStreams {
				go func(topic string, items []ProjectionState) {
					out := make(chan streams.T, 1)
					logger.Debug("publishing stream projection state...", "topic", topic, "fromStreams", fromStreams)
					publisher.Publish(topic, out)

					msgCount := int64(0)
					for _, item := range items {
						msgCount++
						out <- item
					}

					logger.Debug("stream projection state published", "topic", topic, "fromStreams", fromStreams, "messages", msgCount)

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
