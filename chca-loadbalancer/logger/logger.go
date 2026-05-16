package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Log is the package-level logger used throughout the application.
var Log *logrus.Logger

// Init initialises the global logger with the given level string.
func Init(level string) {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	Log.SetLevel(lvl)
}
