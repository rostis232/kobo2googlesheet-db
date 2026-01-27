package app

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/fatih/color"
	"github.com/rostis232/kobo2googlesheet-db/config"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/logwriter"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/service"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
	"github.com/sirupsen/logrus"
)

type App struct {
	service *service.Service
	repo    *repository.Repository
	client  *http.Client
}

func NewApp(dbconf repository.Config) (*App, error) {
	a := &App{}
	db, err := repository.NewMariaDB(dbconf)
	if err != nil {
		return nil, err
	}
	a.repo = repository.NewRepository(db)
	a.service = service.NewService(*a.repo)
	a.client = &http.Client{
		Timeout: 10 * time.Minute,
	}

	return a, err
}

func (a *App) Run(sleepTime string, logLevel string) error {
	logLevelInt, err := strconv.Atoi(logLevel)
	if err != nil {
		color.Red("error while converting value of logging level from string to int :%s. Logging level is setted to 0.", err)
		logLevelInt = 0
	}
	config.SetLogLevel(logLevelInt)
	sleepTimeParsedDuration, err := time.ParseDuration(sleepTime)
	if err != nil {
		return err
	}
	for {

		logwriter.Info("New iteration started", nil)

		data, err := a.repo.GetAllData()
		if err != nil {
			logwriter.Error(fmt.Errorf("error while getting data from DB"), logrus.Fields{"error": err})
			continue
		}

		filteredData := service.FilterTask(data)

		logwriter.Info("Data is successfully retrieved from DB", nil)

		sortedData := a.service.Sorter(filteredData)

		logwriter.Info("Data is successfully sorted", nil)

		for keyAPI, dataSlice := range sortedData {
			shortKeyAPI := []rune(keyAPI)
			if len([]rune(keyAPI)) > 20 {
				shortKeyAPI = shortKeyAPI[:20]
			}
			logwriter.Info("Working with API-key`s set", logrus.Fields{"api_key": string(shortKeyAPI)})

			for _, data := range dataSlice {
				switch {
				case strings.HasSuffix(data.CSVLink, ".csv"):
					a.processCSV(data)
				case strings.HasSuffix(data.CSVLink, ".xls") || strings.HasSuffix(data.CSVLink, ".xlsx"):
					a.processXLS(data)
				default:
					logwriter.Error(errors.New("wrong kobo link"), logrus.Fields{"csv_link": data.CSVLink, "form_id": data.Id})
				}

			}

		}

		logwriter.Info("Iteration completed", logrus.Fields{"wait_time": sleepTime})
		time.Sleep(sleepTimeParsedDuration)
	}
}

func (a *App) processCSV(data models.Data) {
	startTime := time.Now()
	logwriter.Info("Working with Kobo-form`s set", logrus.Fields{"csv_link": data.CSVLink, "form_id": data.Id})

	if data.Status == 0 {
		logwriter.Warn("Skipped form", logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id})
		return
	}

	var records [][]string
	var err error
	for i := 0; i < 3; i++ {
		records, err = a.service.Export(data.CSVLink, data.KoboToken, a.client)
		if err == nil {
			break
		}
		logwriter.Error(fmt.Errorf("attempt %d failed: error while exporting from Kobo", i+1), logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err})
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.Error(fmt.Errorf("error while exporting from Kobo"), logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err})
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("Kobo: %s", err))); err != nil {
			logwriter.Error(fmt.Errorf("error while updating db"), logrus.Fields{"form_id": data.Id, "error": err})
		}
		return
	}
	logwriter.Info("Info is obtained from form successful", logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "duration": time.Since(startTime).String()})

	if len(records) == 0 {
		logwriter.Warn("No values", logrus.Fields{"form_name": data.FormName, "form_id": data.Id})
		return
	}

	importStartTime := time.Now()
	for i := 0; i < 3; i++ {
		err = a.service.Importer(data.APIKey, data.SpreadSheetName, data.SpreadSheetID, data.SheetName, records)
		if err == nil {
			break
		}
		logwriter.Error(fmt.Errorf("attempt %d failed: Error while importing", i+1), logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err})
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.Error(fmt.Errorf("Error while importing"), logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err})
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
			logwriter.Error(fmt.Errorf("error while updating db"), logrus.Fields{"form_id": data.Id, "error": err})
		}
		return
	}

	logwriter.Info("Success", logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "duration": time.Since(importStartTime).String(), "total_duration": time.Since(startTime).String()})

	if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", GetTime())); err != nil {
		logwriter.Error(fmt.Errorf("error while updating db"), logrus.Fields{"form_id": data.Id, "error": err})
	}
}

func (a *App) processXLS(data models.Data) {
	startTime := time.Now()
	logwriter.Info("Working with Kobo-form`s set", logrus.Fields{"csv_link": data.CSVLink, "form_id": data.Id})

	if data.Status == 0 {
		logwriter.Warn("Skipped form", logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id})
		return
	}

	var records map[string][][]string
	var err error
	for i := 0; i < 3; i++ {
		records, err = a.service.ExportXLS(data.CSVLink, data.KoboToken, a.client)
		if err == nil {
			break
		}
		logwriter.Error(fmt.Errorf("attempt %d failed: error while exporting from Kobo", i+1), logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err})
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.Error(fmt.Errorf("error while exporting from Kobo"), logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err})
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("Kobo: %s", err))); err != nil {
			logwriter.Error(fmt.Errorf("error while updating db"), logrus.Fields{"form_id": data.Id, "error": err})
		}
		return
	}
	logwriter.Info("Info is obtained from form successful", logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "duration": time.Since(startTime).String()})

	importStartTime := time.Now()
	for i := 0; i < 3; i++ {
		err = a.service.ImporterXLS(data.APIKey, data.SpreadSheetID, records)
		if err == nil {
			break
		}
		logwriter.Error(fmt.Errorf("attempt %d failed: Error while importing", i+1), logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err})
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.Error(fmt.Errorf("Error while importing"), logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err})
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
			logwriter.Error(fmt.Errorf("error while updating db"), logrus.Fields{"form_id": data.Id, "error": err})
		}
		return
	}
	logwriter.Info("Success", logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "duration": time.Since(importStartTime).String(), "total_duration": time.Since(startTime).String()})
	if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", GetTime())); err != nil {
		logwriter.Error(fmt.Errorf("error while updating db"), logrus.Fields{"form_id": data.Id, "error": err})
	}
}

func GetTime() string {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		return time.Now().Format(time.DateTime)
	}
	return time.Now().In(loc).Format(time.DateTime)
}
