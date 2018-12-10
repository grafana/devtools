package streamprojections

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/grafana/devtools/pkg/streams/pkg/streams"
)

type PartitionFunc func(msg interface{}) (key string, value interface{})

type partionedProjection struct {
	*projection
	keys         map[string]*partitionKey
	state        map[string]ProjectionState
	partitionFns []PartitionFunc
}

func newPartionedProjection(projection *projection, partitionFns []PartitionFunc) *partionedProjection {
	return &partionedProjection{
		projection:   projection,
		keys:         map[string]*partitionKey{},
		state:        map[string]ProjectionState{},
		partitionFns: partitionFns,
	}
}

func (p *partionedProjection) Run(in streams.Readable) []ProjectionState {
	for msg := range in {
		if p.filterFn != nil && !p.callFilter(msg) {
			continue
		}

		pk := p.callPartition(msg)
		key := pk.FormatKey()

		state, exists := p.state[key]
		if !exists {
			state = p.callInit(pk.GetValues())
			p.keys[key] = pk
			p.state[key] = state
		}

		p.callApply(state, msg)
	}

	sortedKeys := []string{}
	for key := range p.state {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	stateArr := []ProjectionState{}
	for _, key := range sortedKeys {
		stateArr = append(stateArr, p.state[key])
	}

	if p.doneFn != nil {
		p.callDone(stateArr)
	}

	return stateArr
}

func (p *partionedProjection) callPartition(msg interface{}) *partitionKey {
	pKey := newPartitionKey()
	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(msg))

	for _, partitionFn := range p.partitionFns {
		ret := reflect.ValueOf(partitionFn).Call(params)
		pKey.Add(ret[0].Interface().(string), ret[1].Interface())
	}

	return pKey
}

type PartionedProjectionBuilder struct {
	*StreamProjectionBuilder
	partitionFns []PartitionFunc
}

func newPartionedProjectionBuilder(baseBuilder *StreamProjectionBuilder) *PartionedProjectionBuilder {
	return &PartionedProjectionBuilder{
		StreamProjectionBuilder: baseBuilder,
		partitionFns:            []PartitionFunc{},
	}
}

func (b *StreamProjectionBuilder) PartitionBy(fns ...PartitionFunc) *PartionedProjectionBuilder {
	partionedBuilder := newPartionedProjectionBuilder(b)
	for _, fn := range fns {
		partionedBuilder.partitionFns = append(partionedBuilder.partitionFns, fn)
	}
	return partionedBuilder
}

func (b *PartionedProjectionBuilder) Init(fn InitFunc) *PartionedProjectionBuilder {
	b.StreamProjectionBuilder.Init(fn)
	return b
}

func (b *PartionedProjectionBuilder) Apply(fn ApplyFunc) *PartionedProjectionBuilder {
	b.StreamProjectionBuilder.Apply(fn)
	return b
}

func (b *PartionedProjectionBuilder) Done(fn DoneFunc) *PartionedProjectionBuilder {
	b.StreamProjectionBuilder.Done(fn)
	return b
}

func (b *PartionedProjectionBuilder) ToStream(name string) *PartionedProjectionBuilder {
	b.StreamProjectionBuilder.ToStream(name)
	return b
}

func (b *PartionedProjectionBuilder) ToStreams(fn SplitToStreamsFunc) *PartionedProjectionBuilder {
	b.StreamProjectionBuilder.ToStreams(fn)
	return b
}

func (b *PartionedProjectionBuilder) Persist(name string, obj interface{}) *PartionedProjectionBuilder {
	b.StreamProjectionBuilder.Persist(name, obj)
	return b
}

func (b *PartionedProjectionBuilder) Build() *StreamProjection {
	projection := newProjection(b.filterFn, b.reduceFn, b.reduceInitialValue, b.initFn, b.applyFn, b.doneFn)
	return newStreamProjection(b.fromStreams, b.splitToStreamsFn, b.persistTo, b.persistObj, newPartionedProjection(projection, b.partitionFns))
}

type partitionKey struct {
	sortedKeys []string
	values     map[string]interface{}
}

func newPartitionKey() *partitionKey {
	return &partitionKey{
		sortedKeys: []string{},
		values:     map[string]interface{}{},
	}
}

func (pk *partitionKey) Add(key string, value interface{}) {
	pk.sortedKeys = append(pk.sortedKeys, key)
	pk.values[key] = value
}

func (pk *partitionKey) Update(key string, value interface{}) {
	pk.values[key] = value
}

func (pk *partitionKey) Get(key string) (interface{}, bool) {
	if v, exists := pk.values[key]; exists {
		return v, true
	}

	return nil, false
}

func (pk *partitionKey) GetKeys() []string {
	keys := []string{}
	for _, key := range pk.sortedKeys {
		keys = append(keys, key)
	}
	return keys
}

func (pk *partitionKey) GetValues() []interface{} {
	values := []interface{}{}
	for _, key := range pk.sortedKeys {
		values = append(values, pk.values[key])
	}
	return values
}

func (pk *partitionKey) FormatKey() string {
	str := ""
	for _, key := range pk.sortedKeys {
		if len(str) > 0 {
			str = str + "|"
		}
		str = str + fmt.Sprintf("%s", pk.values[key])
	}
	return str
}
