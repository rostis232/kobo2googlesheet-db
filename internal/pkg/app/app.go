package app

import (
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/service"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
	"log"
	"net/http"
	"sync"
	"time"
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

func (a *App) Run(sleepTime string) error {
	sleepTimeParsedDuration, err := time.ParseDuration(sleepTime)
	if err != nil {
		return err
	}
	for {
		log.Println("✔️ New iteration started")

		data, err := a.repo.GetAllData()
		if err != nil {
			log.Printf("🔴 Error while getting data from DB: %s.Waiting 10 minutes.\n", err)
			time.Sleep(10 * time.Minute)
			continue
		}

		log.Println("✔️ Data is successfully retrieved from DB.")

		sortedData := a.service.Sorter(data)

		log.Println("✔️ Data is successfully sorted.")

		wg := sync.WaitGroup{}

		for keyAPI, keyLinkMap := range sortedData {
			wg.Add(1)

			go func(keyAPI string, keyLinkMap map[string][]models.Data, wg *sync.WaitGroup) {
				defer wg.Done()
				log.Printf("✔️ Working with API-key`s set: %s.\n", string([]rune(keyAPI)[:100]))

				for keyKoboLink, dataSlice := range keyLinkMap {
					log.Printf("✔️ Working with Kobo-form`s set: %s.\n", keyKoboLink)

					var values [][]interface{}

					for _, data := range dataSlice {

						if data.Status == 0 {
							log.Printf("⚠️ %s -> %s - skipped\n", data.FormName, data.SpreadSheetName)
							continue
						}

						if len(values) == 0 {
							client := &http.Client{
								Timeout: 10 * time.Minute,
							}
							records, err := a.service.Export(data.CSVLink, data.KoboToken, client)
							if err != nil {
								log.Printf("🔴 Error while exporting from Kobo %s: %s\n", data.FormName, err)
								client.CloseIdleConnections()
								break
							}
							client.CloseIdleConnections()
							log.Printf("✔️ Info is obtained from form: %s successful.\n", data.FormName)

							values = a.service.Converter(records)
						}

						if len(values) == 0 {
							log.Printf("⚠️ No values (%s)\n", data.FormName)
							continue
						}

						err = a.service.Importer(data.APIKey, data.SpreadSheetID, data.SheetName, values)
						if err != nil {
							log.Printf("🔴 %s - > %s - Error while importing: %s\n", data.FormName, data.SpreadSheetName, err)
							continue
						}
						log.Printf("✔️ %s -> %s - success.\n", data.FormName, data.SpreadSheetName)
					}
				}

			}(keyAPI, keyLinkMap, &wg)

		}
		wg.Wait()

		log.Printf("✔️ Iteration completed. Waiting for next one after: %s\n", sleepTime)
		time.Sleep(sleepTimeParsedDuration)
	}
}
