package logwriter

import (
	"fmt"
	"github.com/rostis232/kobo2googlesheet-db/config"
	"os"
	"time"
)

func WriteLogToFile(log any) error {
	switch log.(type) {
	case error:
		if err := writeToFile(log); err != nil {
			return err
		}
	case string:
		if config.LogLevel == 0 {
			if err := writeToFile(log); err != nil {
				return err
			}
		}
	default:
		fmt.Println("unknown type of log")
	}
	return nil
}

func writeToFile(log any) error {
	logTime := time.Now()
	fileName := logTime.Month().String()
	file, err := os.OpenFile("./logs/"+fileName+".log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	_, err = file.WriteString(logTime.Format(time.DateTime)+": "+fmt.Sprint(log))
	if err != nil {
		return err
	}
	file.Close()
	return nil
}
