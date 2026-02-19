package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func IsDebug() bool {
	if log.Level == logrus.InfoLevel {
		return false
	}
	return true
}

func ConfigureLog(jsonLog bool, logLevel int) {
	// logging
	if jsonLog {
		log.Formatter = &logrus.JSONFormatter{}
	}

	log.Out = os.Stdout

	switch {
	case logLevel == 1:
		log.Level = logrus.DebugLevel
	case logLevel >= 2:
		log.Level = logrus.TraceLevel
	default:
		log.Level = logrus.InfoLevel
	}
}

func Infof(format string, v ...any) {
	log.Infof(format, v...)
}

func Warnf(format string, v ...any) {
	log.Warnf(format, v...)
}

func Debugf(format string, v ...any) {
	log.Debugf(format, v...)
}

func Tracef(format string, v ...any) {
	log.Tracef(format, v...)
}

func Errorf(format string, v ...any) {
	log.Errorf(format, v...)
}

func Fatalf(format string, v ...any) {
	log.Fatalf(format, v...)
}
