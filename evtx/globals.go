package evtx

import (
	"encoding/binary"
	"errors"
	"math"
	"runtime"
)

var (
	ErrInvalidEvent = errors.New("invalid Event")
	MaxJobs         = int(math.Floor(float64(runtime.NumCPU()) / 2))
	Endianness      = binary.LittleEndian
)

const (
	EventHeaderSize    = 24
	ChunkSize          = 0x10000
	ChunkHeaderSize    = 0x80
	ChunkMagic         = "ElfChnk\x00"
	sizeStringBucket   = 0x40
	sizeTemplateBucket = 0x20
	DefaultNameOffset  = -1
	EventMagic         = "\x2a\x2a\x00\x00"
	MaxSliceSize       = ChunkSize
)

const (
	TokenEOF                  = 0x00
	TokenOpenStartElementTag1 = 0x01
	TokenOpenStartElementTag2 = 0x41
	TokenCloseStartElementTag = 0x02
	TokenCloseEmptyElementTag = 0x03
	TokenEndElementTag        = 0x04
	TokenValue1               = 0x05
	TokenValue2               = 0x45
	TokenAttribute1           = 0x06
	TokenAttribute2           = 0x46
	TokenCharRef1             = 0x08
	TokenEntityRef1           = 0x09
	TokenEntityRef2           = 0x49
	TokenTemplateInstance     = 0x0c
	TokenNormalSubstitution   = 0x0d
	TokenOptionalSubstitution = 0x0e
	FragmentHeaderToken       = 0x0f
)

const (
	NullType       = 0x00
	StringType     = 0x01
	AnsiStringType = 0x02
	Int8Type       = 0x03
	UInt8Type      = 0x04
	Int16Type      = 0x05
	UInt16Type     = 0x06
	Int32Type      = 0x07
	UInt32Type     = 0x08
	Int64Type      = 0x09
	UInt64Type     = 0x0a
	Real64Type     = 0x0c
	BoolType       = 0x0d
	BinaryType     = 0x0e
	GuidType       = 0x0f
	FileTimeType   = 0x11
	SysTimeType    = 0x12
	SidType        = 0x13
	HexInt32Type   = 0x14
	HexInt64Type   = 0x15
	BinXmlType     = 0x21
	ArrayType      = 0x80
)

var (
	PathSeparator = "/"
	XmlnsPath     = Path("/Event/xmlns")
	EventIDPath   = Path("/Event/System/EventID")
)
