package app

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/fatih/color"
	"github.com/rostis232/kobo2googlesheet-db/config"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/logwriter"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/service"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
)

type App struct {
	service *service.Service
	repo    *repository.Repository
}

func NewApp(dbconf repository.Config) (*App, error) {
	a := &App{}
	db, err := repository.NewMariaDB(dbconf)
	if err != nil {
		return nil, err
	}
	a.repo = repository.NewRepository(db)
	a.service = service.NewService(*a.repo)

	return a, err
}

func (a *App) Run(sleepTime string, logLevel string) error {
	logLevelInt, err := strconv.Atoi(logLevel)
	if err != nil {
		color.Red("error while converting value of logging level from string to int :%s. Logging level is setted to 0.", err)
		logLevelInt = 0
	}
	config.SetLogLevel(logLevelInt)
	var dataCache []models.Data
	sleepTimeParsedDuration, err := time.ParseDuration(sleepTime)
	if err != nil {
		return err
	}
	for {

		logwriter.WriteLogToFile("✔️ New iteration started")

		data, err := a.repo.GetAllData()
		if err != nil {
			logwriter.WriteLogToFile(fmt.Errorf("error while getting data from DB: %s. If there is previous data it will be used in this itereation", err))
			if len(dataCache) == 0 {
				logwriter.WriteLogToFile(fmt.Errorf("previous data is empty. I will try to connect to DB in 10 minutes"))
				time.Sleep(10 * time.Minute)
				continue
			}
		} else {
			dataCache = data
		}

		logwriter.WriteLogToFile("✔️ Data is successfully retrieved from DB.")

		sortedData := a.service.Sorter(data)

		logwriter.WriteLogToFile("✔️ Data is successfully sorted.")

		wg := sync.WaitGroup{}

		for keyAPI, dataSlice := range sortedData {
			wg.Add(1)

			go func(keyAPI string, dataSlice []models.Data, wg *sync.WaitGroup) {

				defer wg.Done()

				shortKeyAPI := []rune(keyAPI)
				if len([]rune(keyAPI)) > 20 {
					shortKeyAPI = shortKeyAPI[:20]
				}
				logwriter.WriteLogToFile(fmt.Sprintf("✔️ Working with API-key`s set: %s.\n", string(shortKeyAPI)))

				for _, data := range dataSlice {
					switch {
					case strings.HasSuffix(data.CSVLink, ".csv"):
						a.processCSV(data)
					case strings.HasSuffix(data.CSVLink, ".xls") || strings.HasSuffix(data.CSVLink, ".xlsx"):
						a.processXLS(data)
					default:
						logwriter.WriteLogToFile(errors.New("wrong kobo link"))
					}

				}

			}(keyAPI, dataSlice, &wg)

		}
		wg.Wait()

		logwriter.WriteLogToFile(fmt.Sprintf("✔️ Iteration completed. Waiting for next one after: %s\n", sleepTime))
		time.Sleep(sleepTimeParsedDuration)
	}
}

func (a *App) processCSV(data models.Data) {
	logwriter.WriteLogToFile(fmt.Sprintf("✔️ Working with Kobo-form`s set: %s.\n", data.CSVLink))

	if data.Status == 0 {
		logwriter.WriteLogToFile(fmt.Sprintf("⚠️ %s -> %s - skipped (%d)\n", data.FormName, data.SpreadSheetName, data.Id))
		return
	}

	var records [][]string
	var err error
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}
	for i := 0; i < 3; i++ {
		records, err = a.service.Export(data.CSVLink, data.KoboToken, client)
		if err == nil {
			break
		}
		logwriter.WriteLogToFile(fmt.Errorf("attempt %d failed: error while exporting from Kobo %s (%d): %s", i+1, data.FormName, data.Id, err))
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.WriteLogToFile(fmt.Errorf("error while exporting from Kobo %s (%d): %s", data.FormName, data.Id, err))
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("Kobo: %s", err))); err != nil {
			logwriter.WriteLogToFile(fmt.Errorf("error while updating db (%d): %s", data.Id, err))
		}
		client.CloseIdleConnections()
		return
	}
	client.CloseIdleConnections()
	logwriter.WriteLogToFile(fmt.Sprintf("✔️ Info is obtained from form: %s successful (%d)", data.FormName, data.Id))

	if len(records) == 0 {
		logwriter.WriteLogToFile(fmt.Sprintf("⚠️ No values (%s) (%d)", data.FormName, data.Id))
		return
	}

	for i := 0; i < 3; i++ {
		err = a.service.Importer(data.APIKey, data.SpreadSheetName, data.SpreadSheetID, data.SheetName, records)
		if err == nil {
			break
		}
		logwriter.WriteLogToFile(fmt.Errorf("attempt %d failed: %s - > %s (%d)- Error while importing: %s", i+1, data.FormName, data.SpreadSheetName, data.Id, err))
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.WriteLogToFile(fmt.Errorf("🔴 %s - > %s (%d)- Error while importing: %s", data.FormName, data.SpreadSheetName, data.Id, err))
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
			logwriter.WriteLogToFile(fmt.Errorf("error while updating db (%d): %s", data.Id, err))
		}
		return
	}

	logwriter.WriteLogToFile(fmt.Sprintf("✔️ %s -> %s - success (id %d).\n", data.FormName, data.SpreadSheetName, data.Id))

	if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", GetTime())); err != nil {
		logwriter.WriteLogToFile(fmt.Errorf("error while updating db (%d): %s", data.Id, err))
	}
}

func (a *App) processXLS(data models.Data) {
	logwriter.WriteLogToFile(fmt.Sprintf("✔️ Working with Kobo-form`s set: %s.\n", data.CSVLink))

	if data.Status == 0 {
		logwriter.WriteLogToFile(fmt.Sprintf("⚠️ %s -> %s - skipped (%d)\n", data.FormName, data.SpreadSheetName, data.Id))
		return
	}

	var records map[string][][]string
	var err error
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}
	for i := 0; i < 3; i++ {
		records, err = a.service.ExportXLS(data.CSVLink, data.KoboToken, client)
		if err == nil {
			break
		}
		logwriter.WriteLogToFile(fmt.Errorf("attempt %d failed: error while exporting from Kobo %s (%d): %s", i+1, data.FormName, data.Id, err))
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.WriteLogToFile(fmt.Errorf("error while exporting from Kobo %s (%d): %s", data.FormName, data.Id, err))
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("Kobo: %s", err))); err != nil {
			logwriter.WriteLogToFile(fmt.Errorf("error while updating db (%d): %s", data.Id, err))
		}
		client.CloseIdleConnections()
		return
	}
	client.CloseIdleConnections()
	logwriter.WriteLogToFile(fmt.Sprintf("✔️ Info is obtained from form: %s successful.", data.FormName))

	for i := 0; i < 3; i++ {
		err = a.service.ImporterXLS(data.APIKey, data.SpreadSheetID, records)
		if err == nil {
			break
		}
		logwriter.WriteLogToFile(fmt.Errorf("attempt %d failed: %s - > %s (%d)- Error while importing: %s", i+1, data.FormName, data.SpreadSheetName, data.Id, err))
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logwriter.WriteLogToFile(fmt.Errorf("🔴 %s - > %s (%d)- Error while importing: %s", data.FormName, data.SpreadSheetName, data.Id, err))
		if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", GetTime(), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
			logwriter.WriteLogToFile(fmt.Errorf("error while updating db (%d): %s", data.Id, err))
		}
		return
	}
	logwriter.WriteLogToFile(fmt.Sprintf("✔️ %s -> %s - success (id %d).\n", data.FormName, data.SpreadSheetName, data.Id))
	if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", GetTime())); err != nil {
		logwriter.WriteLogToFile(fmt.Errorf("error while updating db (%d): %s", data.Id, err))
	}
}

func GetTime() string {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		return time.Now().Format(time.DateTime)
	}
	return time.Now().In(loc).Format(time.DateTime)
}
