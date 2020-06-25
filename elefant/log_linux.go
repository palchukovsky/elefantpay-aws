// +build linux

package elefant

import (
	"fmt"
	"log"
	"log/syslog"
	"strings"
)

type productLog struct{ writer *syslog.Writer }

// InitProductLog inits global product log.
func InitProductLog(project, projectPackage, module string) {
	writer, err := syslog.Dial("udp", logService,
		syslog.LOG_EMERG|syslog.LOG_KERN,
		fmt.Sprintf("%s/%s/%s@%s", project, projectPackage, module, Version))
	if err != nil {
		log.Panicf(`Failed to dial syslog: "%v".`, err)
	}
	Log = &productLog{writer: writer}
}

func (productLog *productLog) Flush() {}

func (productLog *productLog) Debug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Println("Debug: " + message)
	productLog.check(productLog.writer.Debug(message))
}

func (productLog *productLog) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Println(message)
	productLog.check(productLog.writer.Info(message))
}

func (productLog *productLog) Warn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Println("Warn: " + message)
	productLog.check(productLog.writer.Warning(message))
}

func (productLog *productLog) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Println("Error: " + message)
	productLog.check(productLog.writer.Err(message))
}

func (productLog *productLog) Err(err error) {
	message := fmt.Sprintf("%v", err)
	if len(message) > 0 {
		message = strings.ToUpper(string(message[0])) + message[1:]
	}
	productLog.Error(message)
}

func (productLog *productLog) Panicf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	productLog.check(productLog.writer.Emerg(message))
	log.Panicln(message)
}

func (*productLog) check(err error) {
	if err == nil {
		return
	}
	log.Printf("Error: Failed to write log record: \"%v\"\n", err)
}
