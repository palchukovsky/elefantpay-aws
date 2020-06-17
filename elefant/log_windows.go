// +build windows

package elefant

import (
	"fmt"
	"log"
	"strings"
)

type productLog struct{}

// InitProductLog inits global product log.
func InitProductLog(project, projectPackage, module string) {
	Log = &productLog{}
}

func (productLog *productLog) Flush() {}

func (productLog *productLog) Debug(format string, args ...interface{}) {
	log.Printf("Debug: "+format+"\n", args...)
}

func (productLog *productLog) Info(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (productLog *productLog) Warn(format string, args ...interface{}) {
	log.Printf("Warn: "+format+"\n", args...)
}

func (productLog *productLog) Error(format string, args ...interface{}) {
	log.Printf("Error: "+format+"\n", args...)
}

func (productLog *productLog) Err(err error) {
	message := fmt.Sprintf("%v", err)
	if len(message) > 0 {
		message = strings.ToUpper(string(message[0])) + message[1:]
	}
	productLog.Error(message)
}

func (productLog *productLog) Panicf(format string, args ...interface{}) {
	log.Panicln(fmt.Sprintf(format, args...))
}
