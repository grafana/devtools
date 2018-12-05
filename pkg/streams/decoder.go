package ghevents

import (
	"encoding/json"
	"io"
)

// EventDecoder reads and decodes events from an input stream.
type EventDecoder interface {
	Decode() (*DecodedEvent, error)
}

type defaultDecoder struct {
	d *json.Decoder
}

// NewDecoder returns a new event decoder that reads from r.
var NewDecoder = func(r io.Reader) EventDecoder {
	return &defaultDecoder{
		d: json.NewDecoder(r),
	}
}

func (de *defaultDecoder) Decode() (*DecodedEvent, error) {
	var evt DecodedEvent

	if err := de.d.Decode(&evt); err != nil {
		return nil, err
	}

	return &evt, nil
}
