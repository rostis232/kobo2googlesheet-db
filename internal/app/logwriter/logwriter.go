package logwriter

import (
	"github.com/fatih/color"
	"github.com/rostis232/kobo2googlesheet-db/config"
)

func WriteLogToFile(log any) {
	switch log.(type) {
	case error:
		color.Red("%s", log)
	case string:
		if config.LogLevel == 0 {
			color.Green("%s", log)
		}
	default:
		color.Red("unknown type of log: %s", log)
	}
	return
}
