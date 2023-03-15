package evtx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"rawsec-evtx/encoding"
	"regexp"
	"sync"
)

type ChunkSorter []Chunk

func (cs ChunkSorter) Len() int {
	return len(cs)
}

func (cs ChunkSorter) Less(i, j int) bool {
	return cs[i].Header.NumFirstRecLog < cs[j].Header.NumFirstRecLog
}

func (cs ChunkSorter) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

var (
	ErrCorruptedHeader = fmt.Errorf("corrupted header")
	ErrDirtyFile       = fmt.Errorf("file is flagged as dirty")
	ErrRepairFailed    = fmt.Errorf("file header could not be repaired")
)

type FileHeader struct {
	Magic           [8]byte
	FirstChunkNum   uint64
	LastChunkNum    uint64
	NextRecordID    uint64
	HeaderSpace     uint32
	MinVersion      uint16
	MajVersion      uint16
	ChunkDataOffset uint16
	ChunkCount      uint16
	Unknown         [76]byte
	Flags           uint32
	CheckSum        uint32
}

func (f *FileHeader) Verify() error {
	if !bytes.Equal(f.Magic[:], []byte("ElfFile\x00")) {
		return ErrCorruptedHeader
	}
	if f.Flags == 1 {
		return ErrDirtyFile
	}
	return nil
}

func (f *FileHeader) Repair(r io.ReadSeeker) error {
	chunkHeaderRE := regexp.MustCompile(ChunkMagic)
	rr := bufio.NewReader(r)
	cc := uint16(0)
	for loc := chunkHeaderRE.FindReaderIndex(rr); loc != nil; loc = chunkHeaderRE.FindReaderIndex(rr) {
		cc++
	}

	if f.ChunkCount > cc {
		return ErrRepairFailed
	}

	f.ChunkCount = cc
	f.LastChunkNum = uint64(f.ChunkCount - 1)
	f.Flags = 0
	return nil
}

type File struct {
	sync.Mutex
	Header          FileHeader
	file            io.ReadSeeker
	monitorExisting bool
}

func New(r io.ReadSeeker) (ef File, err error) {
	ef.file = r
	ef.ParseFileHeader()
	return
}

func Open(filepath string) (ef File, err error) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}

	ef, err = New(file)
	if err != nil {
		return
	}

	err = ef.Header.Verify()

	return
}

func OpenDirty(filepath string) (ef File, err error) {
	if ef, err = Open(filepath); err == ErrDirtyFile {
		err = ef.Header.Repair(ef.file)
	}
	return
}

func (ef *File) ParseFileHeader() {
	ef.Lock()
	defer ef.Unlock()

	GoToSeeker(ef.file, 0)
	err := encoding.Unmarshal(ef.file, &ef.Header, Endianness)
	if err != nil {
		panic(err)
	}
}

func (f FileHeader) String() string {
	return fmt.Sprintf(
		"Magic: %q\n"+
			"FirstChunkNum: %d\n"+
			"LastChunkNum: %d\n"+
			"NumNextRecord: %d\n"+
			"HeaderSpace: %d\n"+
			"MinVersion: 0x%04x\n"+
			"MaxVersion: 0x%04x\n"+
			"SizeHeader: %d\n"+
			"ChunkCount: %d\n"+
			"Flags: 0x%08x\n"+
			"CheckSum: 0x%08x\n",
		f.Magic,
		f.FirstChunkNum,
		f.LastChunkNum,
		f.NextRecordID,
		f.HeaderSpace,
		f.MinVersion,
		f.MajVersion,
		f.ChunkDataOffset,
		f.ChunkCount,
		f.Flags,
		f.CheckSum)

}

func (ef *File) FetchRawChunk(offset int64) (Chunk, error) {
	ef.Lock()
	defer ef.Unlock()
	c := NewChunk()
	GoToSeeker(ef.file, offset)
	c.Offset = offset
	c.Data = make([]byte, ChunkHeaderSize)
	if _, err := ef.file.Read(c.Data); err != nil {
		return c, err
	}
	reader := bytes.NewReader(c.Data)
	c.ParseChunkHeader(reader)
	return c, nil
}

func (ef *File) FetchChunk(offset int64) (Chunk, error) {
	ef.Lock()
	defer ef.Unlock()
	c := NewChunk()
	GoToSeeker(ef.file, offset)
	c.Offset = offset
	c.Data = make([]byte, ChunkSize)
	if _, err := ef.file.Read(c.Data); err != nil {
		return c, err
	}
	reader := bytes.NewReader(c.Data)
	c.ParseChunkHeader(reader)
	GoToSeeker(reader, int64(c.Header.SizeHeader))
	c.ParseStringTable(reader)
	if err := c.ParseTemplateTable(reader); err != nil {
		return c, err
	}
	if err := c.ParseEventOffsets(reader); err != nil {
		return c, err
	}
	return c, nil
}

func (ef *File) UnorderedChunks() (cc chan Chunk) {
	cc = make(chan Chunk)
	go func() {
		defer close(cc)
		for i := uint16(0); i < ef.Header.ChunkCount; i++ {
			offsetChunk := int64(ef.Header.ChunkDataOffset) + int64(ChunkSize)*int64(i)
			chunk, err := ef.FetchRawChunk(offsetChunk)
			switch {
			case err != nil && err != io.EOF:
				panic(err)
			case err == nil:
				cc <- chunk
			}
		}
	}()
	return
}

func (ef *File) UnorderedEvents() (cgem chan *GoEvtxMap) {
	cgem = make(chan *GoEvtxMap, 42)
	go func() {
		defer close(cgem)
		chanQueue := make(chan (chan *GoEvtxMap), MaxJobs)
		go func() {
			defer close(chanQueue)
			for pc := range ef.UnorderedChunks() {
				cpc, err := ef.FetchChunk(pc.Offset)
				switch {
				case err != nil && err != io.EOF:
					panic(err)
				case err == nil:
					ev := cpc.Events()
					chanQueue <- ev
				}
			}
		}()
		for ec := range chanQueue {
			for event := range ec {
				cgem <- event
			}
		}
	}()
	return
}

func (ef *File) Close() error {
	if f, ok := ef.file.(io.Closer); ok {
		return f.Close()
	}

	return nil
}
