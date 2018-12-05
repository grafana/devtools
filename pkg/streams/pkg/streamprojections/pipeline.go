package streamprojections

import (
	"github.com/grafana/github-repo-metrics/pkg/streams/pkg/streams"
)

type TransformFunc func(msg streams.T, out streams.Writable)

func Transform(in streams.Readable, fn TransformFunc) streams.Readable {
	out := make(chan streams.T)

	go func() {
		for msg := range in {
			fn(msg, out)
		}
		close(out)
	}()

	return out
}

func Filter(fn FilterFunc) TransformFunc {
	return func(msg streams.T, out streams.Writable) {
		if fn(msg) {
			out <- msg
		}
	}
}

type ReduceFunc2 func(msg streams.T, out streams.Writable)

func Reduce(fn ReduceFunc2) TransformFunc {
	return func(msg streams.T, out streams.Writable) {
		fn(msg, out)
	}
}
