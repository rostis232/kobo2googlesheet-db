package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/rostis232/kobo2googlesheet-db/config"
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
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Errorf("error while parsing logging level :%s. Logging level is set to info.", err)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	config.SetLogLevel(level)
	sleepTimeParsedDuration, err := time.ParseDuration(sleepTime)
	if err != nil {
		return err
	}
	for {

		logrus.Info("New iteration started")

		data, err := a.repo.GetAllData()
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("error while getting data from DB")
			continue
		}

		filteredData := service.FilterTask(data)

		logrus.Info("Data is successfully retrieved from DB")

		sortedData := a.service.Sorter(filteredData)

		logrus.Info("Data is successfully sorted")

		for keyAPI, dataSlice := range sortedData {
			shortKeyAPI := []rune(keyAPI)
			if len([]rune(keyAPI)) > 20 {
				shortKeyAPI = shortKeyAPI[:20]
			}
			logrus.WithFields(logrus.Fields{"api_key": string(shortKeyAPI)}).Info("Working with API-key`s set")

			for _, data := range dataSlice {
				switch {
				case strings.HasSuffix(data.CSVLink, ".csv"):
					a.processCSV(data)
				case strings.HasSuffix(data.CSVLink, ".xls") || strings.HasSuffix(data.CSVLink, ".xlsx"):
					a.processXLS(data)
				default:
					logrus.WithFields(logrus.Fields{"csv_link": data.CSVLink, "form_id": data.Id}).Error("wrong kobo link")
				}

			}

		}

		logrus.WithFields(logrus.Fields{"wait_time": sleepTime}).Info("Iteration completed")
		time.Sleep(sleepTimeParsedDuration)
	}
}

func (a *App) processCSV(data models.Data) {
	startTime := time.Now()
	logrus.WithFields(logrus.Fields{"csv_link": data.CSVLink, "form_id": data.Id}).Info("Working with Kobo-form`s set")

	if data.Status == 0 {
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id}).Warn("Skipped form")
		return
	}

	var records [][]string
	var err error
	for i := 0; i < 3; i++ {
		records, err = a.service.Export(data.CSVLink, data.KoboToken, a.client)
		if err == nil {
			break
		}
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err}).Errorf("attempt %d failed: error while exporting from Kobo", i+1)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err}).Error("error while exporting from Kobo")
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("Kobo: %s", err))); err != nil {
			logrus.WithFields(logrus.Fields{"form_id": data.Id, "error": err}).Error("error while updating db")
		}
		return
	}
	logrus.WithFields(logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "duration": time.Since(startTime).String()}).Info("Info is obtained from form successful")

	if len(records) == 0 {
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "form_id": data.Id}).Warn("No values")
		return
	}

	importStartTime := time.Now()
	for i := 0; i < 3; i++ {
		err = a.service.Importer(data.APIKey, data.SpreadSheetName, data.SpreadSheetID, data.SheetName, records)
		if err == nil {
			break
		}
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err}).Errorf("attempt %d failed: Error while importing", i+1)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err}).Error("Error while importing")
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
			logrus.WithFields(logrus.Fields{"form_id": data.Id, "error": err}).Error("error while updating db")
		}
		return
	}

	logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "duration": time.Since(importStartTime).String(), "total_duration": time.Since(startTime).String()}).Info("Success")

	if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", GetTime())); err != nil {
		logrus.WithFields(logrus.Fields{"form_id": data.Id, "error": err}).Error("error while updating db")
	}
}

func (a *App) processXLS(data models.Data) {
	startTime := time.Now()
	logrus.WithFields(logrus.Fields{"csv_link": data.CSVLink, "form_id": data.Id}).Info("Working with Kobo-form`s set")

	if data.Status == 0 {
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id}).Warn("Skipped form")
		return
	}

	var records map[string][][]string
	var err error
	for i := 0; i < 3; i++ {
		records, err = a.service.ExportXLS(data.CSVLink, data.KoboToken, a.client)
		if err == nil {
			break
		}
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err}).Errorf("attempt %d failed: error while exporting from Kobo", i+1)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "error": err}).Error("error while exporting from Kobo")
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("Kobo: %s", err))); err != nil {
			logrus.WithFields(logrus.Fields{"form_id": data.Id, "error": err}).Error("error while updating db")
		}
		return
	}
	logrus.WithFields(logrus.Fields{"form_name": data.FormName, "form_id": data.Id, "duration": time.Since(startTime).String()}).Info("Info is obtained from form successful")

	importStartTime := time.Now()
	for i := 0; i < 3; i++ {
		err = a.service.ImporterXLS(data.APIKey, data.SpreadSheetID, records)
		if err == nil {
			break
		}
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err}).Errorf("attempt %d failed: Error while importing", i+1)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "error": err}).Error("Error while importing")
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
			logrus.WithFields(logrus.Fields{"form_id": data.Id, "error": err}).Error("error while updating db")
		}
		return
	}
	logrus.WithFields(logrus.Fields{"form_name": data.FormName, "spreadsheet_name": data.SpreadSheetName, "form_id": data.Id, "duration": time.Since(importStartTime).String(), "total_duration": time.Since(startTime).String()}).Info("Success")
	if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", GetTime())); err != nil {
		logrus.WithFields(logrus.Fields{"form_id": data.Id, "error": err}).Error("error while updating db")
	}
}

func GetTime() string {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		return time.Now().Format(time.DateTime)
	}
	return time.Now().In(loc).Format(time.DateTime)
}
