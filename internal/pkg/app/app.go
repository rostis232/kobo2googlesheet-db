package app

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

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

		iterationStartTime := time.Now()

		if iterationStartTime.Hour() >= 1 && iterationStartTime.Hour() < 7 {
			logwriter.WriteLogToFile("Sleep time...")
			time.Sleep(time.Hour)
			continue
		}

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

		for keyAPI, keyLinkMap := range sortedData {
			wg.Add(1)

			go func(keyAPI string, keyLinkMap map[string][]models.Data, wg *sync.WaitGroup) {


				defer wg.Done()
				logwriter.WriteLogToFile(fmt.Sprintf("✔️ Working with API-key`s set: %s.\n", string([]rune(keyAPI)[:100])))

				for keyKoboLink, dataSlice := range keyLinkMap {
					logwriter.WriteLogToFile(fmt.Sprintf("✔️ Working with Kobo-form`s set: %s.\n", keyKoboLink))

					var records [][]string

					for _, data := range dataSlice {
						// //test
						// if data.SpreadSheetName != "UHF PROT CASE Харків -wot -idx" && data.SpreadSheetName != "UHF PROT CRISIS Харків -wot -idx" && data.SpreadSheetName != "UHF PROT LAW Харків -wot -idx"{
						// 	continue
						// }
						// //test

						if data.Status == 0 {
							logwriter.WriteLogToFile(fmt.Sprintf("⚠️ %s -> %s - skipped\n", data.FormName, data.SpreadSheetName))
							continue
						}

						if len(records) == 0 {
							var err error
							client := &http.Client{
								Timeout: 10 * time.Minute,
							}
							records, err = a.service.Export(data.CSVLink, data.KoboToken, client)
							if err != nil {
								logwriter.WriteLogToFile(fmt.Errorf("error while exporting from Kobo %s (%d): %s", data.FormName, data.Id, err))
								if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", time.Now().Format(time.DateTime), fmt.Sprintf("Kobo: %s", err))); err != nil {
									logwriter.WriteLogToFile(fmt.Errorf("error while updating db: %s", err))
								}
								client.CloseIdleConnections()
								break
							}
							client.CloseIdleConnections()
							logwriter.WriteLogToFile(fmt.Sprintf("✔️ Info is obtained from form: %s successful.", data.FormName))
						}

						if len(records) == 0 {
							logwriter.WriteLogToFile(fmt.Sprintf("⚠️ No values (%s)", data.FormName))
							continue
						}

						err = a.service.Importer(data.APIKey, data.SpreadSheetName, data.SpreadSheetID, data.SheetName, records)
						if err != nil {
							logwriter.WriteLogToFile(fmt.Errorf("🔴 %s - > %s (%d)- Error while importing: %s", data.FormName, data.SpreadSheetName, data.Id, err))
							if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", time.Now().Format(time.DateTime), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
								logwriter.WriteLogToFile(fmt.Errorf("error while updating db: %s", err))
							}
							continue
						}
						logwriter.WriteLogToFile(fmt.Sprintf("✔️ %s -> %s - success (id %d).\n", data.FormName, data.SpreadSheetName, data.Id))
						if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", time.Now().Format(time.DateTime))); err != nil {
							logwriter.WriteLogToFile(fmt.Errorf("error while updating db: %s", err))
						}

					}
				}

			}(keyAPI, keyLinkMap, &wg)

		}
		wg.Wait()

		logwriter.WriteLogToFile(fmt.Sprintf("✔️ Iteration completed. Waiting for next one after: %s\n", sleepTime))
		time.Sleep(sleepTimeParsedDuration)
	}
}

