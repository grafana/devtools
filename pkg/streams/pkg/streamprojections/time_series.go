package streamprojections

import (
	"reflect"
	"sort"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/streams/pkg/streams"
)

type TimeSeriesProjectionState struct {
	ProjectionState
}

type TimeSeriesPartitionFunc func(msg interface{}) time.Time
type TimeSeriesWindowFunc func(t time.Time) (windowTime time.Time, windowKey string)

const timePartitionKey = "ts"

func (pk *partitionKey) AddTimestamp(t time.Time) {
	pk.Add(timePartitionKey, t)
}

func (pk *partitionKey) UpdateTimestamp(t time.Time) {
	pk.Update(timePartitionKey, t)
}

func (pk *partitionKey) GetTimestamp() (time.Time, bool) {
	if v, exists := pk.Get(timePartitionKey); exists {
		if ts, ok := v.(time.Time); ok {
			return ts, true
		}
	}

	return time.Time{}, false
}

type timeSeriesProjection struct {
	*partionedProjection
	windowKeys       map[string]*partitionKey
	windows          map[string][]interface{}
	tsPartitioner    TimePartitioner
	windowPreceeding int
	windowFollowing  int
	windowFormat     string
	windowApplyFn    TimeWindowApplyFunc
}

func newTimeSeriesProjection(pp *partionedProjection, tp TimePartitioner, windowPreceeding, windowFollowing int, windowFormat string, windowApplyFn TimeWindowApplyFunc) *timeSeriesProjection {
	return &timeSeriesProjection{
		partionedProjection: pp,
		windowKeys:          map[string]*partitionKey{},
		windows:             map[string][]interface{}{},
		tsPartitioner:       tp,
		windowPreceeding:    windowPreceeding,
		windowFollowing:     windowFollowing,
		windowFormat:        windowFormat,
		windowApplyFn:       windowApplyFn,
	}
}

type timeProjectionState struct {
	time  time.Time
	state ProjectionState
}

func (p *timeSeriesProjection) Run(in streams.Readable) []ProjectionState {
	result := p.partionedProjection.Run(in)

	if p.windowApplyFn == nil {
		return result
	}

	group := map[string][]*timeProjectionState{}
	groupKeys := map[string]*partitionKey{}

	sortedKeys := []string{}
	for key := range p.state {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		pk := p.partionedProjection.keys[key]

		ts, ok := pk.GetTimestamp()
		if !ok {
			continue
		}
		keys := pk.GetKeys()
		values := pk.GetValues()
		pkWithoutTs := newPartitionKey()
		for i, key := range keys {
			// skip timestamp and format
			if i == 0 || i == len(keys)-1 {
				continue
			}
			pkWithoutTs.Add(key, values[i])
		}
		pkWithoutTsKey := pkWithoutTs.FormatKey()

		if _, exists := group[pkWithoutTsKey]; !exists {
			group[pkWithoutTsKey] = []*timeProjectionState{}
		}
		group[pkWithoutTsKey] = append(group[pkWithoutTsKey], &timeProjectionState{time: ts, state: p.state[key]})
		groupKeys[pkWithoutTsKey] = pkWithoutTs
	}

	state := map[string]ProjectionState{}

	for pkWithoutTsKey, slice := range group {
		slice = p.tsPartitioner.FillMissingValues(slice)
		var interfaceSlice = make([]interface{}, len(slice))
		for i, d := range slice {
			interfaceSlice[i] = d
		}
		windowSlices := p.tsPartitioner.Window(p.windowPreceeding, p.windowFollowing, interfaceSlice)

		for _, windowSlice := range windowSlices {
			pkWithoutTs := groupKeys[pkWithoutTsKey]
			ts := slice[windowSlice.curIndex].time
			pk := newPartitionKey()
			pk.AddTimestamp(ts)
			values := pkWithoutTs.GetValues()
			for i, key := range pkWithoutTs.GetKeys() {
				pk.Add(key, values[i])
			}
			pk.Add("format", p.windowFormat)
			key := pk.FormatKey()

			s := p.callInit(pk.GetValues())
			for _, item := range slice[windowSlice.wStart:windowSlice.wEnd] {
				if item.state == nil {
					item.state = p.callInit(pk.GetValues())
				}
				p.callWindowApply(s, item.state, len(slice[windowSlice.wStart:windowSlice.wEnd]))
			}
			state[key] = s
		}
	}

	sortedKeys = []string{}
	for key := range state {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	stateArr := []ProjectionState{}
	for _, key := range sortedKeys {
		stateArr = append(stateArr, state[key])
	}

	return stateArr
}

func (p *timeSeriesProjection) callWindowApply(state ProjectionState, msg interface{}, windowSize int) {
	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(state))
	params = append(params, reflect.ValueOf(msg))
	params = append(params, reflect.ValueOf(windowSize))

	reflect.ValueOf(p.windowApplyFn).Call(params)
}

type TimeSeriesProjectionBuilder struct {
	*PartionedProjectionBuilder
	tsPartitioner    TimePartitioner
	windowPreceeding int
	windowFollowing  int
	windowFormat     string
	windowApplyFn    TimeWindowApplyFunc
}

func (b *StreamProjectionBuilder) TimeSeries(tsPartitioner TimePartitioner, fns ...PartitionFunc) *TimeSeriesProjectionBuilder {
	partitionFns := []PartitionFunc{}
	partitionFns = append(partitionFns, func(msg interface{}) (key string, value interface{}) {
		return timePartitionKey, tsPartitioner.Partition(msg)
	})
	partitionFns = append(partitionFns, fns...)
	partitionFns = append(partitionFns, func(msg interface{}) (key string, value interface{}) {
		return "tsFormat", tsPartitioner.GetFormat()
	})
	builder := &TimeSeriesProjectionBuilder{
		PartionedProjectionBuilder: b.PartitionBy(partitionFns...),
		tsPartitioner:              tsPartitioner,
	}

	return builder
}

func (b *StreamProjectionBuilder) Hourly(fn TimeSeriesPartitionFunc, fns ...PartitionFunc) *TimeSeriesProjectionBuilder {
	return b.TimeSeries(newHourlyTimeSeriesPartitioner(fn), fns...)
}

func (b *StreamProjectionBuilder) Daily(fn TimeSeriesPartitionFunc, fns ...PartitionFunc) *TimeSeriesProjectionBuilder {
	return b.TimeSeries(newDailyTimeSeriesPartitioner(fn), fns...)
}

func (b *StreamProjectionBuilder) Weekly(fn TimeSeriesPartitionFunc, fns ...PartitionFunc) *TimeSeriesProjectionBuilder {
	return b.TimeSeries(newWeeklyTimeSeriesPartitioner(fn), fns...)
}

func (b *StreamProjectionBuilder) Monthly(fn TimeSeriesPartitionFunc, fns ...PartitionFunc) *TimeSeriesProjectionBuilder {
	return b.TimeSeries(newMonthlyTimeSeriesPartitioner(fn), fns...)
}

func (b *StreamProjectionBuilder) Quarterly(fn TimeSeriesPartitionFunc, fns ...PartitionFunc) *TimeSeriesProjectionBuilder {
	return b.TimeSeries(newQuarterlyTimeSeriesPartitioner(fn), fns...)
}

func (b *StreamProjectionBuilder) Yearly(fn TimeSeriesPartitionFunc, fns ...PartitionFunc) *TimeSeriesProjectionBuilder {
	return b.TimeSeries(newYearlyTimeSeriesPartitioner(fn), fns...)
}

func (b *TimeSeriesProjectionBuilder) Init(fn InitFunc) *TimeSeriesProjectionBuilder {
	b.PartionedProjectionBuilder.Init(fn)
	return b
}

func (b *TimeSeriesProjectionBuilder) Apply(fn ApplyFunc) *TimeSeriesProjectionBuilder {
	b.PartionedProjectionBuilder.Apply(fn)
	return b
}

type TimeWindowApplyFunc interface{}

func (b *TimeSeriesProjectionBuilder) Window(preceeding, following int, format string, fn TimeWindowApplyFunc) *TimeSeriesProjectionBuilder {
	b.windowPreceeding = preceeding
	b.windowFollowing = following
	b.windowFormat = format
	b.windowApplyFn = fn
	return b
}

func (b *TimeSeriesProjectionBuilder) Done(fn DoneFunc) *TimeSeriesProjectionBuilder {
	b.PartionedProjectionBuilder.Done(fn)
	return b
}

func (b *TimeSeriesProjectionBuilder) ToStream(name string) *TimeSeriesProjectionBuilder {
	b.PartionedProjectionBuilder.ToStream(name)
	return b
}

func (b *TimeSeriesProjectionBuilder) ToStreams(fn SplitToStreamsFunc) *TimeSeriesProjectionBuilder {
	b.PartionedProjectionBuilder.ToStreams(fn)
	return b
}

func (b *TimeSeriesProjectionBuilder) Persist(name string, obj interface{}) *TimeSeriesProjectionBuilder {
	b.PartionedProjectionBuilder.Persist(name, obj)
	return b
}

func (b *TimeSeriesProjectionBuilder) Build() *StreamProjection {
	projection := newProjection(b.filterFn, b.reduceFn, b.reduceInitialValue, b.initFn, b.applyFn, b.doneFn)
	partionedProjection := newPartionedProjection(projection, b.partitionFns)
	tsProjection := newTimeSeriesProjection(partionedProjection, b.tsPartitioner, b.windowPreceeding, b.windowFollowing, b.windowFormat, b.windowApplyFn)
	return newStreamProjection(b.fromStreams, b.splitToStreamsFn, b.persistTo, b.persistObj, tsProjection)
}

type WindowSlice struct {
	curIndex int
	wStart   int
	wEnd     int
}

type TimePartitioner interface {
	Partition(msg interface{}) time.Time
	GetFormat() string
	FillMissingValues(data []*timeProjectionState) []*timeProjectionState
	Window(preceeding, following int, data []interface{}) []*WindowSlice
}

type TimePartitionerBase struct {
	ExtractTimeFn TimeSeriesPartitionFunc
	StepFn        func(time.Time) time.Time
	Format        string
}

func NewTimeSeriesPartitionerBase(extractTimeFn TimeSeriesPartitionFunc, stepFn func(time.Time) time.Time, format string) *TimePartitionerBase {
	return &TimePartitionerBase{
		ExtractTimeFn: extractTimeFn,
		StepFn:        stepFn,
		Format:        format,
	}
}

func (p *TimePartitionerBase) GetFormat() string {
	return p.Format
}

func (p *TimePartitionerBase) FillMissingValues(data []*timeProjectionState) []*timeProjectionState {
	if len(data) == 0 {
		return data
	}

	if len(data) == 1 {
		data = append(data, &timeProjectionState{
			time: p.StepFn(data[0].time),
		})
		return data
	}

	res := []*timeProjectionState{}
	prev := time.Time{}

	for n := 0; n < len(data); n++ {
		if prev.IsZero() {
			prev = data[n].time
			res = append(res, data[n])
			continue
		}

		for p.StepFn(prev).Before(data[n].time) {
			prev = p.StepFn(prev)
			res = append(res, &timeProjectionState{time: prev})
		}

		res = append(res, data[n])
		prev = data[n].time
	}

	return res
}

func (p *TimePartitionerBase) Window(preceeding, following int, data []interface{}) []*WindowSlice {
	res := []*WindowSlice{}
	allPreceeding := preceeding == -1

	for n := 0; n < len(data); n++ {
		if allPreceeding {
			preceeding = n
		}

		if (preceeding > 0 && n < preceeding) || (following > 0 && len(data[n:]) <= following) {
			continue
		}

		w := &WindowSlice{
			curIndex: n,
			wStart:   n,
			wEnd:     n + 1,
		}

		if preceeding > 0 {
			w.wStart = n - preceeding
		}

		if following > 0 && len(data[n:]) > following {
			w.wEnd = n + following + 1
		}

		res = append(res, w)
	}

	return res
}

type dailyTimeSeriesPartitioner struct {
	*TimePartitionerBase
}

func newDailyTimeSeriesPartitioner(extractTimeFn TimeSeriesPartitionFunc) TimePartitioner {
	stepFn := func(t time.Time) time.Time {
		return t.Add(24 * time.Hour)
	}

	return &dailyTimeSeriesPartitioner{
		TimePartitionerBase: NewTimeSeriesPartitionerBase(extractTimeFn, stepFn, "d"),
	}
}

func (p *dailyTimeSeriesPartitioner) Partition(msg interface{}) time.Time {
	t := p.ExtractTimeFn(msg)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

type weeklyTimeSeriesPartitioner struct {
	*TimePartitionerBase
}

func newWeeklyTimeSeriesPartitioner(extractTimeFn TimeSeriesPartitionFunc) TimePartitioner {
	stepFn := func(t time.Time) time.Time {
		return t.Add(7 * 24 * time.Hour)
	}

	return &weeklyTimeSeriesPartitioner{
		TimePartitionerBase: NewTimeSeriesPartitionerBase(extractTimeFn, stepFn, "w"),
	}
}

func (p *weeklyTimeSeriesPartitioner) Partition(msg interface{}) time.Time {
	t := p.ExtractTimeFn(msg)
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	if d.Weekday() == time.Sunday {
		return d.Add(-6 * 24 * time.Hour)
	}

	if d.Weekday() == time.Monday {
		return d
	}

	return d.Add(-time.Duration(int64(d.Weekday()-1)) * 24 * time.Hour)
}

type monthlyTimeSeriesPartitioner struct {
	*TimePartitionerBase
}

func newMonthlyTimeSeriesPartitioner(extractTimeFn TimeSeriesPartitionFunc) TimePartitioner {
	stepFn := func(t time.Time) time.Time {
		return t.AddDate(0, 1, 0)
	}

	return &monthlyTimeSeriesPartitioner{
		TimePartitionerBase: NewTimeSeriesPartitionerBase(extractTimeFn, stepFn, "m"),
	}
}

func (p *monthlyTimeSeriesPartitioner) Partition(msg interface{}) time.Time {
	t := p.ExtractTimeFn(msg)
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

type quarterlyTimeSeriesPartitioner struct {
	*TimePartitionerBase
}

func newQuarterlyTimeSeriesPartitioner(extractTimeFn TimeSeriesPartitionFunc) TimePartitioner {
	stepFn := func(t time.Time) time.Time {
		return t.AddDate(0, 3, 0)
	}

	return &quarterlyTimeSeriesPartitioner{
		TimePartitionerBase: NewTimeSeriesPartitionerBase(extractTimeFn, stepFn, "q"),
	}
}

func (p *quarterlyTimeSeriesPartitioner) Partition(msg interface{}) time.Time {
	t := p.ExtractTimeFn(msg)

	if t.Month() < time.April {
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	}

	if t.Month() < time.July {
		return time.Date(t.Year(), 4, 1, 0, 0, 0, 0, t.Location())
	}

	if t.Month() < time.October {
		return time.Date(t.Year(), 7, 1, 0, 0, 0, 0, t.Location())
	}

	return time.Date(t.Year(), 10, 1, 0, 0, 0, 0, t.Location())
}

type yearlyTimeSeriesPartitioner struct {
	*TimePartitionerBase
}

func newYearlyTimeSeriesPartitioner(extractTimeFn TimeSeriesPartitionFunc) TimePartitioner {
	stepFn := func(t time.Time) time.Time {
		return t.AddDate(1, 0, 0)
	}

	return &yearlyTimeSeriesPartitioner{
		TimePartitionerBase: NewTimeSeriesPartitionerBase(extractTimeFn, stepFn, "y"),
	}
}

func (p *yearlyTimeSeriesPartitioner) Partition(msg interface{}) time.Time {
	t := p.ExtractTimeFn(msg)
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

type hourlyTimeSeriesPartitioner struct {
	*TimePartitionerBase
}

func newHourlyTimeSeriesPartitioner(extractTimeFn TimeSeriesPartitionFunc) TimePartitioner {
	stepFn := func(t time.Time) time.Time {
		return t.Add(1 * time.Hour)
	}

	return &hourlyTimeSeriesPartitioner{
		TimePartitionerBase: NewTimeSeriesPartitionerBase(extractTimeFn, stepFn, "h"),
	}
}

func (p *hourlyTimeSeriesPartitioner) Partition(msg interface{}) time.Time {
	t := p.ExtractTimeFn(msg)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}
