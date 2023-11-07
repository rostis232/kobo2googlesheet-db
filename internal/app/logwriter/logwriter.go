package logwriter

import (
	"fmt"
	"github.com/rostis232/kobo2googlesheet-db/config"
)

func WriteLogToFile(log any) error {
	switch log.(type) {
	case error:
		fmt.Println(log)
	case string:
		if config.LogLevel == 0 {
			fmt.Println(log)
		}
	default:
		fmt.Println("unknown type of log: ", log)
	}
	return nil
}
