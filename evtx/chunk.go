package evtx

import (
	"bytes"
	"fmt"
	"io"

	"rawsec-evtx/encoding"
)

type ChunkString struct {
	Name
}

func StringAt(reader io.ReadSeeker, offset int64) (cs ChunkString, err error) {
	backup := BackupSeeker(reader)
	GoToSeeker(reader, offset)
	err = cs.Parse(reader)
	GoToSeeker(reader, backup)
	return
}

type ChunkStringTable map[int32]ChunkString

type TemplateTable map[int32]TemplateDefinitionData

type ChunkHeader struct {
	Magic           [8]byte
	NumFirstRecLog  int64
	NumLastRecLog   int64
	FirstEventRecID int64
	LastEventRecID  int64
	SizeHeader      int32
	OffsetLastRec   int32
	Freespace       int32
	CheckSum        uint32
}

func (ch ChunkHeader) String() string {
	return fmt.Sprintf(
		"\tMagic: %s\n"+
			"\tNumFirstRecLog: %d\n"+
			"\tNumLastRecLog: %d\n"+
			"\tNumFirstRecFile: %d\n"+
			"\tNumLastRecFile: %d\n"+
			"\tSizeHeader: %d\n"+
			"\tOffsetLastRec: %d\n"+
			"\tFreespace: %d\n"+
			"\tCheckSum: 0x%08x\n",
		ch.Magic,
		ch.NumFirstRecLog,
		ch.NumLastRecLog,
		ch.FirstEventRecID,
		ch.LastEventRecID,
		ch.SizeHeader,
		ch.OffsetLastRec,
		ch.Freespace,
		ch.CheckSum)
}

type Chunk struct {
	Offset        int64
	Header        ChunkHeader
	StringTable   ChunkStringTable
	TemplateTable TemplateTable
	EventOffsets  []int32
	Data          []byte
}

func NewChunk() Chunk {
	return Chunk{StringTable: make(ChunkStringTable, 0), TemplateTable: make(TemplateTable, 0)}
}

func (c *Chunk) ParseChunkHeader(reader io.ReadSeeker) {
	err := encoding.Unmarshal(reader, &c.Header, Endianness)
	if err != nil {
		panic(err)
	}
}

func (c *Chunk) ParseStringTable(reader io.ReadSeeker) {
	strOffset := int32(0)
	for i := int64(0); i < sizeStringBucket*4; i += 4 {
		_ = encoding.Unmarshal(reader, &strOffset, Endianness)
		if strOffset > 0 {
			cs, err := StringAt(reader, int64(strOffset))
			if err != nil {
				panic(err)
			}
			c.StringTable[strOffset] = cs
		}
	}
	return
}

func (c *Chunk) ParseTemplateTable(reader io.ReadSeeker) error {
	templateDataOffset := int32(0)
	for i := int32(0); i < sizeTemplateBucket*4; i = i + 4 {
		err := encoding.Unmarshal(reader, &templateDataOffset, Endianness)
		if err != nil {
			return err
		}
		if templateDataOffset > 0 {
			backup := BackupSeeker(reader)
			GoToSeeker(reader, int64(templateDataOffset))
			tdd := TemplateDefinitionData{}
			err := tdd.Parse(reader)
			if err != nil {
				return err
			}
			c.TemplateTable[templateDataOffset] = tdd
			GoToSeeker(reader, backup)
		}
	}
	return nil
}

func (c *Chunk) ParseEventOffsets(reader io.ReadSeeker) (err error) {
	c.EventOffsets = make([]int32, 0)
	offsetEvent := int32(BackupSeeker(reader))
	c.EventOffsets = append(c.EventOffsets, offsetEvent)
	for offsetEvent <= c.Header.OffsetLastRec {
		eh := EventHeader{}
		GoToSeeker(reader, int64(offsetEvent))
		if err = encoding.Unmarshal(reader, &eh, Endianness); err != nil {
			return err
		}
		if err = eh.Validate(); err != nil {
			return err
		}
		offsetEvent += eh.Size
		c.EventOffsets = append(c.EventOffsets, offsetEvent)
	}
	return nil
}

func (c *Chunk) ParseEvent(offset int64) (e Event) {
	if int64(c.Header.OffsetLastRec) < offset {
		return
	}
	reader := bytes.NewReader(c.Data)
	GoToSeeker(reader, offset)
	e.Offset = offset
	err := encoding.Unmarshal(reader, &e.Header, Endianness)
	if err != nil {
		panic(err)
	}
	return e
}

func (c *Chunk) Events() (cgem chan *GoEvtxMap) {
	cgem = make(chan *GoEvtxMap, len(c.EventOffsets))
	go func() {
		defer close(cgem)
		for _, eo := range c.EventOffsets {
			event := c.ParseEvent(int64(eo))
			gem, err := event.GoEvtxMap(c)
			if err == nil {
				cgem <- gem
			}
		}
	}()
	return
}

func (c Chunk) String() string {
	templateOffsets := make([]int32, len(c.TemplateTable))
	i := 0
	for to := range c.TemplateTable {
		templateOffsets[i] = to
		i++
	}
	return fmt.Sprintf(
		"Header: %v\n"+
			"StringTable: %v\n"+
			"TemplateTable: %v\n"+
			"EventOffsets: %v\n"+
			"TemplatesOffsets (for debug): %v\n", c.Header, c.StringTable, c.TemplateTable, c.EventOffsets, templateOffsets)
}
