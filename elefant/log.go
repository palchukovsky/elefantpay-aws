package elefant

// Log is global product log object.
var Log ProductLog

// ProductLog describes product log interface.
type ProductLog interface {
	Flush()

	Debug(format string, args ...interface{})

	Info(format string, args ...interface{})

	Warn(format string, args ...interface{})

	Error(format string, args ...interface{})
	Err(err error)

	Panicf(format string, args ...interface{})
}

var logService string // set by builder
