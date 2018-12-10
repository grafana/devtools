package projections

import (
	"reflect"

	"github.com/grafana/devtools/pkg/streams"
)

type Projection interface {
	Run(in streams.Readable) []ProjectionState
}

type ProjectionState interface{}

type InitFunc interface{}
type ApplyFunc interface{}
type DoneFunc interface{}

type projection struct {
	state              ProjectionState
	filterFn           FilterFunc
	reduceFn           ReduceFunc
	reduceInitialValue interface{}
	initFn             InitFunc
	applyFn            ApplyFunc
	doneFn             DoneFunc
}

func newProjection(filterFn FilterFunc, reduceFn ReduceFunc, reduceInitialValue interface{}, initFn InitFunc, applyFn ApplyFunc, doneFn DoneFunc) *projection {
	return &projection{
		filterFn:           filterFn,
		reduceFn:           reduceFn,
		reduceInitialValue: reduceInitialValue,
		applyFn:            applyFn,
		initFn:             initFn,
		doneFn:             doneFn,
	}
}

func (p *projection) callFilter(msg interface{}) bool {
	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(msg))

	ret := reflect.ValueOf(p.filterFn).Call(params)
	return ret[0].Interface().(bool)
}

func (p *projection) callInit(values []interface{}) interface{} {
	var params = []reflect.Value{}
	for _, val := range values {
		params = append(params, reflect.ValueOf(val))
	}

	ret := reflect.ValueOf(p.initFn).Call(params)
	return ret[0].Interface()
}

func (p *projection) callApply(state ProjectionState, msg interface{}, additionalArgs ...interface{}) {
	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(state))
	params = append(params, reflect.ValueOf(msg))

	for _, arg := range additionalArgs {
		params = append(params, reflect.ValueOf(arg))
	}

	reflect.ValueOf(p.applyFn).Call(params)
}

func (p *projection) callDone(state ProjectionState) {
	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(state))

	reflect.ValueOf(p.doneFn).Call(params)
}

func (p *projection) Run(in streams.Readable) []ProjectionState {
	if p.initFn != nil {
		p.state = p.callInit([]interface{}{})
	}

	lastAccumulatedValue := p.reduceInitialValue

	for msg := range in {
		if p.filterFn != nil && !p.callFilter(msg) {
			continue
		}

		if p.reduceFn != nil {
			lastAccumulatedValue = p.reduceFn(lastAccumulatedValue, msg)
		}

		if p.initFn == nil {
			if p.state == nil {
				p.state = []interface{}{}
			}
			arr := p.state.([]interface{})
			arr = append(arr, msg)
			p.state = arr
		}

		if p.applyFn != nil {
			p.callApply(p.state, msg)
		}
	}

	if arr, ok := p.state.([]interface{}); ok {
		stateArr := []ProjectionState{}
		for _, item := range arr {
			stateArr = append(stateArr, item)
		}
		if p.doneFn != nil {
			p.callDone(stateArr)
		}
		return stateArr
	}

	if p.doneFn != nil {
		p.callDone(p.state)
	}

	return []ProjectionState{p.state}
}

type StreamProjectionBuilder struct {
	fromStreams        []string
	filterFn           FilterFunc
	reduceFn           ReduceFunc
	reduceInitialValue interface{}
	initFn             InitFunc
	applyFn            ApplyFunc
	splitToStreamsFn   SplitToStreamsFunc
	doneFn             DoneFunc
	persistTo          string
	persistObj         interface{}
}

func newStreamProjectionBuilder() *StreamProjectionBuilder {
	return &StreamProjectionBuilder{
		fromStreams: []string{},
	}
}

func (b *StreamProjectionBuilder) fromStream(name string) *StreamProjectionBuilder {
	b.fromStreams = append(b.fromStreams, name)
	return b
}

func (b *StreamProjectionBuilder) Filter(fn FilterFunc) *StreamProjectionBuilder {
	b.filterFn = fn
	return b
}

func (b *StreamProjectionBuilder) Reduce(fn ReduceFunc, initialValue interface{}) *StreamProjectionBuilder {
	b.reduceFn = fn
	b.reduceInitialValue = initialValue
	return b
}

func (b *StreamProjectionBuilder) Init(fn InitFunc) *StreamProjectionBuilder {
	b.initFn = fn
	return b
}

func (b *StreamProjectionBuilder) Apply(fn ApplyFunc) *StreamProjectionBuilder {
	b.applyFn = fn
	return b
}

func (b *StreamProjectionBuilder) Done(fn DoneFunc) *StreamProjectionBuilder {
	b.doneFn = fn
	return b
}

func (b *StreamProjectionBuilder) ToStream(name string) *StreamProjectionBuilder {
	b.splitToStreamsFn = func(state []ProjectionState) map[string][]ProjectionState {
		return map[string][]ProjectionState{
			name: state,
		}
	}
	return b
}

func (b *StreamProjectionBuilder) ToStreams(fn SplitToStreamsFunc) *StreamProjectionBuilder {
	b.splitToStreamsFn = fn
	return b
}

func (b *StreamProjectionBuilder) Persist(name string, obj interface{}) *StreamProjectionBuilder {
	b.persistTo = name
	b.persistObj = obj
	return b
}

func (b *StreamProjectionBuilder) Build() *StreamProjection {
	return newStreamProjection(b.fromStreams, b.splitToStreamsFn, b.persistTo, b.persistObj, newProjection(b.filterFn, b.reduceFn, b.reduceInitialValue, b.initFn, b.applyFn, b.doneFn))
}
