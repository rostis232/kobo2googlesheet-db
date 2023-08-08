package app

import (
	"fmt"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/service"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
	"log"
	"sync"
	"time"
)

type App struct {
	service *service.Service
	repo    *repository.Repository
}

func NewApp(dbconf repository.Config) *App {
	a := &App{}
	db, err := repository.NewMariaDB(dbconf)
	if err != nil {
		log.Fatalln(err)
	}
	a.repo = repository.NewRepository(db)
	a.service = service.NewService(*a.repo)

	return a
}

func (a *App) Run() {
	for {
		log.Println("New iteration started")
		log.Println("Getting data from DB")

		data, err := a.repo.GetAllData()
		if err != nil {
			log.Printf("Error while getting data from DB: %s.Waiting 10 minutes.", err)
			time.Sleep(10 * time.Minute)
			continue
		}

		log.Println("Data from DB got successful")

		sortedData := a.service.Sorter(data)

		log.Println("Data sorted successful")

		wg := sync.WaitGroup{}

		for keyAPI, keyLinkMap := range sortedData {

			wg.Add(1)

			go func(keyAPI string, keyLinkMap map[string][]models.Data, wg *sync.WaitGroup) {
				fmt.Printf("Working with API-key`s set: %s.\n", string([]rune(keyAPI)[:10]))
				for keyKoboLink, dataSlice := range keyLinkMap {
					log.Printf("Working with Kobo-form`s set: %s.\n", keyKoboLink)

					var values [][]interface{}

					for index, data := range dataSlice {

						if index == 0 {
							log.Printf("Getting information from form: %s\n", *data.FormName)

							records, err := a.service.Export(*data.CSVLink, *data.KoboToken)
							if err != nil {
								log.Println(err)
								continue
							}
							log.Printf("Info is goten from form: %s successful.\n", *data.FormName)

							values = a.service.Converter(records)
						}

						log.Printf("Exporting data into table: %s.", *data.SpreadSheetName)
						err = a.service.Importer(*data.APIKey, *data.SpreadSheetID, *data.SheetName, values)
						if err != nil {
							log.Println(err)
							continue

						}
						log.Printf("Exporting data into table: %s is successful.\n", *data.SpreadSheetName)
					}
				}

				wg.Done()
			}(keyAPI, keyLinkMap, &wg)

			wg.Wait()

		}

		log.Println("Iteration completed.")
		time.Sleep(1 * time.Hour)
	}
}
