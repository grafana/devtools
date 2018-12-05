package stream

import (
	"context"
)

type Emitter interface {
	Emit(msg Msg) error
}

type emitter struct {
	ctx    context.Context
	output Writable
}

func (e *emitter) Emit(msg Msg) error {
	select {
	case <-e.ctx.Done():
		return e.ctx.Err()
	default:
		e.output <- msg
		return nil
	}
}

func NewEmitter(ctx context.Context, output Writable) Emitter {
	return &emitter{
		ctx:    ctx,
		output: output,
	}
}

type Producer interface {
	Produce(context.Context) Readable
}

type ProduceFunc func(emitter Emitter) error

type defaultProducer struct {
	ctx       context.Context
	produceFn ProduceFunc
	capacity  int
}

func NewProducer(produceFn ProduceFunc) Producer {
	return NewBufferedProducer(0, produceFn)
}

func NewBufferedProducer(capacity int, produceFn ProduceFunc) Producer {
	return &defaultProducer{
		produceFn: produceFn,
		capacity:  capacity,
	}
}

func (p *defaultProducer) Produce(ctx context.Context) Readable {
	r, w := NewBuffered(p.capacity)

	go func() {
		defer close(w)
		p.produceFn(NewEmitter(ctx, w))
	}()

	return r
}
