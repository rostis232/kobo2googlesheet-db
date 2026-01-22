package logwriter

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Log = logrus.New()

func init() {
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)
}

func Info(msg string, fields logrus.Fields) {
	Log.WithFields(fields).Info(msg)
}

func Error(err error, fields logrus.Fields) {
	Log.WithFields(fields).Error(err)
}

func Warn(msg string, fields logrus.Fields) {
	Log.WithFields(fields).Warn(msg)
}

func WriteLogToFile(logtext any) {
	switch v := logtext.(type) {
	case error:
		Log.Error(v)
	case string:
		Log.Info(v)
	default:
		Log.Printf("unknown type of log: %v", v)
	}
}
