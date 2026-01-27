package config

import "github.com/sirupsen/logrus"

var LogLevel logrus.Level

func SetLogLevel(level logrus.Level) {
	LogLevel = level
}
