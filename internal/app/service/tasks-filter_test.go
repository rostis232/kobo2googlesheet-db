package service

import (
	"database/sql"
	"testing"
	"time"

	"github.com/rostis232/kobo2googlesheet-db/internal/models"
)

func TestGetPeriod(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{
			name:  "valid 1h",
			input: "some string -period=1h",
			want:  1 * time.Hour,
		},
		{
			name:  "valid 30m",
			input: "task name -period=30m",
			want:  30 * time.Minute,
		},
		{
			name:  "valid 24h",
			input: "daily report -period=24h",
			want:  24 * time.Hour,
		},
		{
			name:  "no period flag",
			input: "just a string",
			want:  3 * time.Hour,
		},
		{
			name:  "invalid duration value",
			input: "string -period=invalid",
			want:  3 * time.Hour,
		},
		{
			name:  "missing space before hyphen",
			input: "string-period=1h",
			want:  3 * time.Hour,
		},
		{
			name:  "period in the middle",
			input: "prefix -period=2h suffix",
			want:  2 * time.Hour,
		},
		{
			name:  "empty string",
			input: "",
			want:  3 * time.Hour,
		},
		{
			name:  "multiple spaces",
			input: "task  -period=5h",
			want:  5 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPeriod(tt.input)
			if got != tt.want {
				t.Errorf("getPeriod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTimeFromLastResult(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Kyiv")
	tests := []struct {
		name       string
		lastResult sql.NullString
		want       time.Time
	}{
		{
			name:       "Ok result",
			lastResult: sql.NullString{String: "Ok; 2024-01-11 11:46:21", Valid: true},
			want:       time.Date(2024, 1, 11, 11, 46, 21, 0, loc),
		},
		{
			name:       "Error result",
			lastResult: sql.NullString{String: "ERROR; 2025-02-18 11:02:54; GoogleSheets: googleapi: got HTTP response code 502", Valid: true},
			want:       time.Date(2025, 2, 18, 11, 2, 54, 0, loc),
		},
		{
			name:       "Invalid format",
			lastResult: sql.NullString{String: "Invalid string", Valid: true},
			want:       time.Time{},
		},
		{
			name:       "Empty string",
			lastResult: sql.NullString{String: "", Valid: true},
			want:       time.Time{},
		},
		{
			name:       "Null result",
			lastResult: sql.NullString{Valid: false},
			want:       time.Time{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := getTimeFromLastResult(tt.lastResult)
			if !got.Equal(tt.want) {
				t.Errorf("getTimeFromLastResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterTask(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Kyiv")
	now := time.Now().In(loc)

	tasks := []models.Data{
		{
			SpreadSheetName: "Task 1 -period=1h",
			LastResult:      sql.NullString{String: "Ok; " + now.Add(-2*time.Hour).Format(time.DateTime), Valid: true},
		},
		{
			SpreadSheetName: "Task 2 -period=1h",
			LastResult:      sql.NullString{String: "Ok; " + now.Add(-30*time.Minute).Format(time.DateTime), Valid: true},
		},
		{
			SpreadSheetName: "Task 3", // default 3h
			LastResult:      sql.NullString{String: "Ok; " + now.Add(-4*time.Hour).Format(time.DateTime), Valid: true},
		},
		{
			SpreadSheetName: "Task 4",
			LastResult:      sql.NullString{String: "Invalid format", Valid: true}, // Should be included due to error
		},
		{
			SpreadSheetName: "Task 5",
			LastResult:      sql.NullString{Valid: false}, // Should be included due to null
		},
	}

	got := FilterTask(tasks)

	if len(got) != 4 {
		t.Errorf("FilterTask() returned %d tasks, want 4", len(got))
	}

	// Check if correct tasks are included
	foundTask1 := false
	foundTask3 := false
	foundTask4 := false
	foundTask5 := false

	for _, task := range got {
		if task.SpreadSheetName == "Task 1 -period=1h" {
			foundTask1 = true
		}
		if task.SpreadSheetName == "Task 3" {
			foundTask3 = true
		}
		if task.SpreadSheetName == "Task 4" {
			foundTask4 = true
		}
		if task.SpreadSheetName == "Task 5" {
			foundTask5 = true
		}
	}

	if !foundTask1 {
		t.Error("Task 1 should be in result")
	}
	if !foundTask3 {
		t.Error("Task 3 should be in result")
	}
	if !foundTask4 {
		t.Error("Task 4 should be in result")
	}
	if !foundTask5 {
		t.Error("Task 5 should be in result")
	}
}
