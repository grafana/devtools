package streams

import "sync"

type TransformFunc func(msg T, out Writable)

func Transform(in Readable, fn TransformFunc) Readable {
	out := make(chan T)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for msg := range in {
			fn(msg, out)
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

type FilterFunc func(msg interface{}) bool

func Filter(fn FilterFunc) TransformFunc {
	return func(msg T, out Writable) {
		if fn(msg) {
			out <- msg
		}
	}
}

type ReduceFunc func(msg T, out Writable)

func Reduce(fn ReduceFunc) TransformFunc {
	return func(msg T, out Writable) {
		fn(msg, out)
	}
}

func Combine(streams []Readable) Readable {
	r, w := NewReadableWritable()
	var wg sync.WaitGroup
	wg.Add(len(streams))

	for _, stream := range streams {
		go func(s Readable) {
			for msg := range s {
				w <- msg
			}
			wg.Done()
		}(stream)
	}

	go func() {
		wg.Wait()
		close(w)
	}()

	return r
}

func Split(streamCount int, stream Readable) ReadableCollection {
	readables, writables := NewReadableWritableCollection(streamCount)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for evt := range stream {
			for n := 0; n < streamCount; n++ {
				c := writables[n]
				c <- evt
			}
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()

		for n := 0; n < streamCount; n++ {
			close(writables[n])
		}
	}()

	return readables
}
