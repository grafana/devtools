package streams

import (
	"fmt"
	"sort"
	"sync"
)

type T interface{}
type Readable <-chan T
type Writable chan<- T
type ReadableCollection []Readable
type WritableCollection []Writable

func New() (Readable, Writable) {
	ch := make(chan T)
	return ch, ch
}

func NewFrom(slice ...interface{}) Readable {
	r, w := New()
	go func() {
		for _, v := range slice {
			w <- v
		}
		w.Close()
	}()

	return r
}

func NewFromValue(val interface{}) Readable {
	return NewFrom(val)
}

func NewFromRange(lowerBound, upperBound int) Readable {
	slice := []interface{}{}
	for n := lowerBound; n <= upperBound; n++ {
		slice = append(slice, n)
	}

	return NewFrom(slice...)
}

func (r Readable) Drain() {
	for range r {
	}
}

func (w Writable) Close() {
	close(w)
}

func NewCollection(size int) (ReadableCollection, WritableCollection) {
	rc := make(ReadableCollection, size)
	wc := make(WritableCollection, size)

	for n := 0; n < size; n++ {
		ch := make(chan T)
		rc[n] = ch
		wc[n] = ch
	}

	return rc, wc
}

func (wc WritableCollection) Close() {
	for _, w := range wc {
		w.Close()
	}
}

type TransformFunc func(msg T, out Writable)

func Transform(in Readable, fn TransformFunc) Readable {
	r, w := New()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for msg := range in {
			fn(msg, w)
		}
	}()

	go func() {
		wg.Wait()
		w.Close()
	}()

	return r
}

func (r Readable) Transform(fn TransformFunc) Readable {
	return Transform(r, fn)
}

type FilterFunc func(msg T) bool

func Filter(in Readable, fn FilterFunc) Readable {
	return in.Transform(func(msg T, out Writable) {
		if fn(msg) {
			out <- msg
		}
	})
}

func (r Readable) Filter(fn FilterFunc) Readable {
	return Filter(r, fn)
}

type MapFunc func(msg T) T

func Map(in Readable, fn MapFunc) Readable {
	return in.Transform(func(msg T, out Writable) {
		out <- fn(msg)
	})
}

func (r Readable) Map(fn MapFunc) Readable {
	return Map(r, fn)
}

type ReduceFunc func(accumulator T, msg T) T

func Reduce(in Readable, fn ReduceFunc, accumulator T) Readable {
	r, w := New()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for msg := range in {
			accumulator = fn(accumulator, msg)
		}
		w <- accumulator
	}()

	go func() {
		wg.Wait()
		w.Close()
	}()

	return r
}

func (r Readable) Reduce(fn ReduceFunc, accumulator T) Readable {
	return Reduce(r, fn, accumulator)
}

type FlatMapFunc func(msg T) []interface{}

func FlatMap(in Readable, fn FlatMapFunc) Readable {
	return in.Transform(func(msg T, out Writable) {
		slice := fn(msg)
		for _, value := range slice {
			out <- value
		}
	})
}

func (r Readable) FlatMap(fn FlatMapFunc) Readable {
	return FlatMap(r, fn)
}

func Split(streamCount int, stream Readable) ReadableCollection {
	rc, wc := NewCollection(streamCount)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for evt := range stream {
			for n := 0; n < streamCount; n++ {
				c := wc[n]
				c <- evt
			}
		}
	}()

	go func() {
		wg.Wait()
		wc.Close()
	}()

	return rc
}

func (r Readable) Split(streams int) ReadableCollection {
	return Split(streams, r)
}

func Combine(streams ReadableCollection) Readable {
	r, w := New()
	var wg sync.WaitGroup
	wg.Add(len(streams))

	for _, stream := range streams {
		go func(s Readable) {
			defer wg.Done()
			for msg := range s {
				w <- msg
			}
		}(stream)
	}

	go func() {
		wg.Wait()
		w.Close()
	}()

	return r
}

func (rc ReadableCollection) Combine() Readable {
	return Combine(rc)
}

type GroupedT struct {
	PartitionKey *PartitionKey
	Stream       Readable
	values       []interface{}
}

type GroupedReadable <-chan *GroupedT
type GroupedWritable chan<- *GroupedT

func NewGrouped() (GroupedReadable, GroupedWritable) {
	ch := make(chan *GroupedT)
	return ch, ch
}

func (w GroupedWritable) Close() {
	close(w)
}

type GroupByFunc func(msg T) ([]string, []interface{})

func GroupBy(in Readable, fn GroupByFunc) GroupedReadable {
	gr, gw := NewGrouped()
	groupedStreams := map[string]*GroupedT{}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for msg := range in {
			keys, values := fn(msg)
			pKey := NewPartitionKey()
			for k, v := range keys {
				pKey.Add(v, values[k])
			}
			key := pKey.FormatKey()
			if groupedStream, exists := groupedStreams[key]; !exists {
				groupedStreams[key] = &GroupedT{
					PartitionKey: pKey,
					values:       []interface{}{msg},
				}
			} else {
				groupedStream.values = append(groupedStream.values, msg)
				groupedStreams[key] = groupedStream
			}
		}

		sortedKeys := []string{}
		for k := range groupedStreams {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			groupedStream := groupedStreams[key]
			groupedStream.Stream = NewFrom(groupedStream.values...)
			gw <- groupedStream
		}

		wg.Done()
	}()

	go func() {
		wg.Wait()
		gw.Close()
	}()

	return gr
}

func (r Readable) GroupBy(fn GroupByFunc) GroupedReadable {
	return GroupBy(r, fn)
}

func (gr GroupedReadable) Reduce(fn ReduceFunc, accumulator T) Readable {
	r, w := New()

	go func() {
		for grouped := range gr {
			reduced := grouped.Stream.Reduce(fn, accumulator)
			for r := range reduced {
				w <- r
			}
		}

		w.Close()
	}()

	return r
}

type PartitionKey struct {
	sortedKeys []string
	values     map[string]interface{}
}

func NewPartitionKey() *PartitionKey {
	return &PartitionKey{
		sortedKeys: []string{},
		values:     map[string]interface{}{},
	}
}

func (pk *PartitionKey) Add(key string, value interface{}) {
	pk.sortedKeys = append(pk.sortedKeys, key)
	pk.values[key] = value
}

func (pk *PartitionKey) Update(key string, value interface{}) {
	pk.values[key] = value
}

func (pk *PartitionKey) Get(key string) (interface{}, bool) {
	if v, exists := pk.values[key]; exists {
		return v, true
	}

	return nil, false
}

func (pk *PartitionKey) GetKeys() []string {
	keys := []string{}
	for _, key := range pk.sortedKeys {
		keys = append(keys, key)
	}
	return keys
}

func (pk *PartitionKey) GetValues() []interface{} {
	values := []interface{}{}
	for _, key := range pk.sortedKeys {
		values = append(values, pk.values[key])
	}
	return values
}

func (pk *PartitionKey) FormatKey() string {
	str := ""
	for _, key := range pk.sortedKeys {
		if len(str) > 0 {
			str = str + "|"
		}
		str = str + fmt.Sprintf("%s", pk.values[key])
	}
	return str
}
