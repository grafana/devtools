package stream

type Msg interface{}
type Readable <-chan Msg
type Writable chan<- Msg

func New() (Readable, Writable) {
	ch := make(chan Msg)
	return ch, ch
}

func NewBuffered(size int) (Readable, Writable) {
	ch := make(chan Msg, size)
	return ch, ch
}

func (in Readable) ReadAll() []Msg {
	res := []Msg{}
	for d := range in {
		res = append(res, d)
	}
	return res
}

func (in Readable) Read() Msg {
	return <-in
}

func (stream Writable) SendMany(messages ...Msg) {
	for _, msg := range messages {
		stream.Send(msg)
	}
}

func (stream Writable) Send(msg Msg) {
	stream <- msg
}

func (stream Writable) Close() {
	close(stream)
}
