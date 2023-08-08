package app

import (
	"fmt"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/service"
	"log"
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
		for i1, d1 := range sortedData {

			fmt.Printf("Working with API-key`s set: %s.", string([]rune(i1)[:10]))
			for i2, d2 := range d1 {
				log.Printf("Working with Kobo-form`s set: %s.", i2)

				var values [][]interface{}

				for index, d3 := range d2 {

					if index == 0 {
						log.Printf("Getting information from form: %s", *d3.FormName)

						records, err := a.service.Export(*d3.CSVLink, *d3.KoboToken)
						if err != nil {
							log.Println(err)
							continue
						}
						log.Printf("Info is goten from form: %s successful.", *d3.FormName)

						values = a.service.Converter(records)
					}

					log.Printf("Exporting data into table: %s.", *d3.SpreadSheetName)
					err = a.service.Importer(*d3.APIKey, *d3.SpreadSheetID, *d3.SheetName, values)
					if err != nil {
						log.Println(err)
						continue

					}
					log.Printf("Exporting data into table: %s is successful.", *d3.SpreadSheetName)
				}
			}

		}

		log.Println("Iteration completed.")
		time.Sleep(1 * time.Hour)
	}
}
