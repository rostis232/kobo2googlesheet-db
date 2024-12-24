package logwriter

import (
	"github.com/rostis232/kobo2googlesheet-db/config"
	"log"
)

func WriteLogToFile(logtext any) {
	switch logtext.(type) {
	case error:
		log.Printf("%s", logtext)
	case string:
		if config.LogLevel == 0 {
			log.Printf("%s", logtext)
		}
	default:
		log.Printf("unknown type of log: %s", logtext)
	}
	return
}
