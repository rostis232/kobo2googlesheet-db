package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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
		fmt.Println(err)
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
			if err := logwriter.WriteLogToFile("Sleep time....\n"); err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Hour)
			continue
		}

		if err := logwriter.WriteLogToFile("✔️ New iteration started\n"); err != nil {
			fmt.Println(err)
		}

		data, err := a.repo.GetAllData()
		if err != nil {
			if err := logwriter.WriteLogToFile(fmt.Errorf("error while getting data from DB: %s. If there is previous data it will be used in this itereation", err)); err != nil {
				fmt.Println(err)
			}
			if len(dataCache) == 0 {
				if err := logwriter.WriteLogToFile(fmt.Errorf("previous data is empty. I will try to connect to DB in 10 minutes")); err != nil {
					fmt.Println(err)
				}
				time.Sleep(10 * time.Minute)
				continue
			}
		} else {
			dataCache = data
		}

		if err := logwriter.WriteLogToFile("✔️ Data is successfully retrieved from DB.\n"); err != nil {
			fmt.Println(err)
		}

		sortedData := a.service.Sorter(data)

		if err := logwriter.WriteLogToFile("✔️ Data is successfully sorted.\n"); err != nil {
			fmt.Println(err)
		}

		wg := sync.WaitGroup{}

		for keyAPI, keyLinkMap := range sortedData {
			wg.Add(1)

			go func(keyAPI string, keyLinkMap map[string][]models.Data, wg *sync.WaitGroup) {
				defer wg.Done()
				if err := logwriter.WriteLogToFile(fmt.Sprintf("✔️ Working with API-key`s set: %s.\n", string([]rune(keyAPI)[:100]))); err != nil {
					fmt.Println(err)
				}

				for keyKoboLink, dataSlice := range keyLinkMap {
					if err := logwriter.WriteLogToFile(fmt.Sprintf("✔️ Working with Kobo-form`s set: %s.\n", keyKoboLink)); err != nil {
						fmt.Println(err)
					}

					var values [][]interface{}

					for _, data := range dataSlice {

						if data.Status == 0 {
							if err := logwriter.WriteLogToFile(fmt.Sprintf("⚠️ %s -> %s - skipped\n", data.FormName, data.SpreadSheetName)); err != nil {
								fmt.Println(err)
							}
							continue
						}

						if len(values) == 0 {
							client := &http.Client{
								Timeout: 10 * time.Minute,
							}
							records, err := a.service.Export(data.CSVLink, data.KoboToken, client)
							if err != nil {
								if err := logwriter.WriteLogToFile(fmt.Errorf("error while exporting from Kobo %s (%d): %s", data.FormName, data.Id, err)); err != nil {
									fmt.Println(err)
								}
								if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", time.Now().Format(time.DateTime), fmt.Sprintf("Kobo: %s", err))); err != nil {
									if err := logwriter.WriteLogToFile(fmt.Errorf("error while updating db: %s", err)); err != nil {
										fmt.Println(err)
									}
								}
								client.CloseIdleConnections()
								break
							}
							client.CloseIdleConnections()
							if err := logwriter.WriteLogToFile(fmt.Sprintf("✔️ Info is obtained from form: %s successful.\n", data.FormName)); err != nil {
								fmt.Println(err)
							}

							if strings.Contains(data.SpreadSheetName, " -wot") {
								records = records[1:]
								fmt.Printf("%s: Founded -wot: deleted titles", data.SpreadSheetName)
							}

							if strings.Contains(data.SpreadSheetName, " -idx") {
								fmt.Printf("%s: Founded -idx: changing index", data.SpreadSheetName)
								records = changingIndex(records)
							}

							values = a.service.Converter(records)
						}

						if len(values) == 0 {
							if err := logwriter.WriteLogToFile(fmt.Sprintf("⚠️ No values (%s)\n", data.FormName)); err != nil {
								fmt.Println(err)
							}
							continue
						}

						err = a.service.Importer(data.APIKey, data.SpreadSheetID, data.SheetName, values)
						if err != nil {
							if err := logwriter.WriteLogToFile(fmt.Errorf("🔴 %s - > %s (%d)- Error while importing: %s", data.FormName, data.SpreadSheetName, data.Id, err)); err != nil {
								fmt.Println(err)
							}
							if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("ERROR; %s; %s", time.Now().Format(time.DateTime), fmt.Sprintf("GoogleSheets: %s", err))); err != nil {
								if err := logwriter.WriteLogToFile(fmt.Errorf("error while updating db: %s", err)); err != nil {
									fmt.Println(err)
								}
							}
							continue
						}
						if err := logwriter.WriteLogToFile(fmt.Sprintf("✔️ %s -> %s - success (id %d).\n", data.FormName, data.SpreadSheetName, data.Id)); err != nil {
							fmt.Println(err)
						}
						if err := a.repo.WriteInfo(data.Id, fmt.Sprintf("Ok; %s", time.Now().Format(time.DateTime))); err != nil {
							if err := logwriter.WriteLogToFile(fmt.Errorf("error while updating db: %s", err)); err != nil {
								fmt.Println(err)
							}
						}

					}
				}

			}(keyAPI, keyLinkMap, &wg)

		}
		wg.Wait()

		if err := logwriter.WriteLogToFile(fmt.Sprintf("✔️ Iteration completed. Waiting for next one after: %s\n", sleepTime)); err != nil {
			fmt.Println(err)
		}
		time.Sleep(sleepTimeParsedDuration)
	}
}

func changingIndex(input [][]string) [][]string {
	indexId := 0
	for rowId, cells := range input {
		for cellId, cellValue := range cells {
			if rowId == 0 {
				if cellValue == "_index" {
					indexId = cellId
				}
			} else {
				if indexId == 0 {
					fmt.Println("Index not found")
					return input
				} else {
					if cellId == indexId {
						input[rowId][cellId] = "i"+cellValue
					}
				}

			}

		}
	}
	return input
}