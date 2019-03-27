package streams

type StreamPersister interface {
	Register(name string, objTemplate interface{}) error
	Persist(name string, stream Readable) error
}

type noOpStreamPersister struct {
}

func NewNoOpStreamPersister() StreamPersister {
	return &noOpStreamPersister{}
}

func (sp *noOpStreamPersister) Register(name string, objTemplate interface{}) error {
	return nil
}

func (sp *noOpStreamPersister) Persist(name string, stream Readable) error {
	stream.Drain()
	return nil
}
