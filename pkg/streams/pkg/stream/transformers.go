package stream

import "reflect"

type Transformer interface {
	Transform(in Readable) (out Readable)
}

type TransformOnNextFunc func(msg Msg, out Writable)
type TransformOnCompletedFunc func(out Writable)
type FilterFunc func(msg Msg) bool
type MapFunc func(msg Msg) Msg
type PartitionFunc func(msg Msg) (string, interface{})
type ReduceFunc func(accumulator Msg, current Msg) Msg

type DefaultTransformer struct {
	OnNext      TransformOnNextFunc
	OnCompleted TransformOnCompletedFunc
}

func (t *DefaultTransformer) Transform(in Readable) Readable {
	r, w := New()

	go func() {
		defer w.Close()
		for msg := range in {
			t.OnNext(msg, w)
		}
		if t.OnCompleted != nil {
			t.OnCompleted(w)
		}
	}()

	return r
}

func Filter(predicateFn FilterFunc) Transformer {
	return &DefaultTransformer{
		OnNext: func(msg Msg, out Writable) {
			if predicateFn(msg) {
				out <- msg
			}
		},
	}
}

func Map(fn MapFunc) Transformer {
	return &DefaultTransformer{
		OnNext: func(msg Msg, out Writable) {
			out <- fn(msg)
		},
	}
}

func Partition(fns ...PartitionFunc) Transformer {
	sortedKeys := []string{}
	messagePartitions := map[string]*MessagePartition{}

	return &DefaultTransformer{
		OnNext: func(msg Msg, out Writable) {
			pk := newPartitionKey()
			for _, fn := range fns {
				pk.Add(fn(msg))
			}
			key := pk.GetCompositeKey()
			if _, exists := messagePartitions[key]; !exists {
				sortedKeys = append(sortedKeys, key)
				messagePartitions[key] = NewMessagePartition(pk)
			}
			messagePartitions[key].Add(msg)
		},
		OnCompleted: func(out Writable) {
			for _, key := range sortedKeys {
				out <- messagePartitions[key]
			}
		},
	}
}

func Reduce(fn ReduceFunc, accumulator Msg) Transformer {
	return &DefaultTransformer{
		OnNext: func(msg Msg, out Writable) {
			accumulator = fn(accumulator, msg)
		},
		OnCompleted: func(out Writable) {
			out <- accumulator
		},
	}
}

func Flatten() Transformer {
	return &DefaultTransformer{
		OnNext: func(msg Msg, out Writable) {
			dv := reflect.ValueOf(msg)
			if dv.Kind() == reflect.Slice || dv.Kind() == reflect.Ptr && dv.Elem().Kind() == reflect.Slice {
				for i := 0; i < dv.Len(); i++ {
					out <- dv.Index(i).Interface()
				}
			} else {
				out <- msg
			}
		},
	}
}

type flatMapTransformer struct {
	mapFn MapFunc
}

func FlatMap(fn MapFunc) Transformer {
	return &flatMapTransformer{
		mapFn: fn,
	}
}

func (t *flatMapTransformer) Transform(in Readable) Readable {
	mapped := Map(t.mapFn).Transform(in)
	return Flatten().Transform(mapped)
}
