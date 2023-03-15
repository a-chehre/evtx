package evtx

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf16"
)

func ToJSON(data interface{}) []byte {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return b
}

func BackupSeeker(seeker io.Seeker) int64 {
	backup, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}
	return backup
}

func GoToSeeker(seeker io.Seeker, offset int64) {
	_, _ = seeker.Seek(offset, io.SeekStart)
}

func RelGoToSeeker(seeker io.Seeker, offset int64) {
	_, _ = seeker.Seek(offset, io.SeekCurrent)
}

type UTF16String []uint16

var (
	UTF16EndOfString = uint16(0x0)
)

func (us *UTF16String) Len() int32 {
	return int32(len(*us)) * 2
}

func (us UTF16String) ToString() string {
	return strings.TrimRight(string(utf16.Decode(us)), "\u0000")
}

type UTCTime time.Time

func (u UTCTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", time.Time(u).UTC().Format(time.RFC3339Nano))), nil
}

type FileTime struct {
	Nanoseconds int64
}

func (v *FileTime) Convert() (sec int64, nsec int64) {
	nano := int64(10000000)
	sec = int64(float64(v.Nanoseconds)/float64(nano) - 11644473600.0)
	nsec = ((v.Nanoseconds - 11644473600*nano) - sec*nano) * 100
	return
}

func (v *FileTime) Time() UTCTime {
	sec, nsec := v.Convert()
	return UTCTime(time.Unix(sec, nsec))
}

func (v *FileTime) String() string {
	sec, nsec := v.Convert()
	return time.Unix(sec, nsec).Format(time.RFC3339Nano)
}
