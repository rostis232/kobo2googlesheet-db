package service

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rostis232/kobo2googlesheet-db/internal/models"
	"github.com/sirupsen/logrus"
)

func FilterTask(tasks []models.Data) []models.Data {
	result := make([]models.Data, 0)
	for _, task := range tasks {
		period := getPeriod(task.SpreadSheetName)
		lastUpdate, err := getTimeFromLastResult(task.LastResult)
		if err != nil {
			result = append(result, task)
			continue
		}

		if time.Since(lastUpdate) >= period {
			result = append(result, task)
		}
	}

	logrus.WithFields(logrus.Fields{"count": len(result)}).Info("Filtered tasks")
	return result
}

func getPeriod(gsName string) time.Duration {
	defaultDuration := 3 * time.Hour
	re := regexp.MustCompile(` -period=([^ ]+)`)
	matches := re.FindStringSubmatch(gsName)
	if len(matches) < 2 {
		return defaultDuration
	}

	duration, err := time.ParseDuration(matches[1])
	if err != nil {
		return defaultDuration
	}

	return duration
}

func getTimeFromLastResult(lastResult sql.NullString) (time.Time, error) {
	if !lastResult.Valid {
		return time.Time{}, errors.New("last result is null")
	}
	parts := strings.Split(lastResult.String, ";")
	if len(parts) < 2 {
		return time.Time{}, errors.New("invalid last result format")
	}

	timeStr := strings.TrimSpace(parts[1])
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t, parseErr := time.ParseInLocation(time.DateTime, timeStr, time.UTC)
		if parseErr != nil {
			return time.Time{}, fmt.Errorf("error while parsing time: %w", parseErr)
		}
		// Return time and error about location
		return t, fmt.Errorf("error while loading location: %w", err)
	}

	t, err := time.ParseInLocation(time.DateTime, timeStr, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("error while parsing time: %w", err)
	}

	return t, nil
}
