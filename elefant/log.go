package elefant

import (
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
)

// Log is global product log object.
var Log = &LogService{}

// Set by builder.
var logServiceDSN = ""

// LogService describes product log interface.
type LogService struct{}

// Init inits global product log.
func (productLog *LogService) Init(project, projectPackage, module string) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:     logServiceDSN,
		Release: fmt.Sprintf("%s_%s@%s", project, projectPackage, Version),
	})
	if err != nil {
		log.Fatalf(`Failed to init Sentry: "%v".`, err)
	}
	Log = &LogService{}
}

// Flush flushes log records.
func (productLog *LogService) Flush() {
	if !sentry.Flush(2 * time.Second) {
		productLog.Error("Error: failed to flush Sentry.")
	}
}

// Debug formats and logs debug message.
func (productLog *LogService) Debug(format string, args ...interface{}) {
	productLog.Info(format, args...)
}

// Info formats and logs information message.
func (*LogService) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Println(message)
	sentry.CaptureMessage(message)
}

// Error formats and logs error message.
func (productLog *LogService) Error(format string, args ...interface{}) {
	productLog.Err(fmt.Errorf(format, args...))
}

// Err logs error object.
func (*LogService) Err(err error) {
	log.Printf("Error: %v.\n", err)
	sentry.CaptureException(err)
}

// Panicf formats message, logs it and panics.
func (productLog *LogService) Panicf(format string, args ...interface{}) {
	productLog.Error(format, args...)
	productLog.Flush()
	log.Panicf("Panic: \"%s\".", args...)
}
