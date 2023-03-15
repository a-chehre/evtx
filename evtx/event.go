package evtx

import (
	"bytes"
	"fmt"
	"io"

	"rawsec-evtx/log"
)

type EventHeader struct {
	Magic     [4]byte
	Size      int32
	ID        int64
	Timestamp FileTime
}

func (h *EventHeader) Validate() error {
	if string(h.Magic[:]) != EventMagic {
		return fmt.Errorf("bad event magic %q", h.Magic)
	}
	if h.Size >= ChunkSize {
		return fmt.Errorf("too big event")
	}
	if h.Size < EventHeaderSize {
		return fmt.Errorf("too small event")
	}
	return nil
}

type Event struct {
	Offset int64
	Header EventHeader
}

func (e *Event) IsValid() bool {
	return e.Header.Validate() == nil
}

func (e Event) GoEvtxMap(c *Chunk) (pge *GoEvtxMap, err error) {
	if !e.IsValid() {
		err = ErrInvalidEvent
		return
	}
	reader := bytes.NewReader(c.Data)
	GoToSeeker(reader, e.Offset+EventHeaderSize)
	element, err := Parse(reader, c, false)
	if err != nil && err != io.EOF {
		log.Error(err)
	}
	fragment, ok := element.(*Fragment)
	if !ok {
		_ = element.(*Fragment)
	}
	return fragment.GoEvtxMap(), err
}

func (e Event) String() string {
	return fmt.Sprintf(
		"Magic: %s\n"+
			"Size: %d\n"+
			"ID: %d\n"+
			"Timestamp: %d\n",
		e.Header.Magic,
		e.Header.Size,
		e.Header.ID,
		e.Header.Timestamp)
}
