package log

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
)

const (
	LDebug    = 1
	LInfo     = 1 << 1
	LError    = 1 << 2
	LCritical = 1 << 3
)

var (
	gLogLevel = LInfo
)

func init() {
	InitLogger(LInfo)
}

func InitLogger(logLevel int) {
	SetLogLevel(logLevel)
	if logLevel <= LDebug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

func SetLogLevel(logLevel int) {

	switch logLevel {
	case LInfo:
		gLogLevel = logLevel
	case LDebug:
		gLogLevel = logLevel
	case LCritical:
		gLogLevel = logLevel
	case LError:
		gLogLevel = logLevel
	default:
		gLogLevel = LInfo
	}
}

func logMessage(prefix string, i ...interface{}) {
	format := fmt.Sprintf("%s%s", prefix, strings.Repeat("%v ", len(i)))
	msg := fmt.Sprintf(format, i...)
	_ = log.Output(3, msg)
}

func Error(i ...interface{}) {
	if gLogLevel <= LError {
		logMessage("ERROR - ", i...)
	}
}

func Errorf(format string, i ...interface{}) {
	if gLogLevel <= LError {
		logMessage("ERROR - ", fmt.Sprintf(format, i...))
	}
}

func DontPanicf(format string, i ...interface{}) {
	msg := fmt.Sprintf("%v\n %s", fmt.Sprintf(format, i...), debug.Stack())
	logMessage("PANIC - ", msg)
}
