package app

import (
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
		iterationStartTime := time.Now()
		log.Println("Getting data from DB")
		data, err := a.repo.GetAllData()
		if err != nil {
			log.Println(err)
			time.Sleep(30 * time.Minute)
			continue
		}
		log.Println("Data from DB got successful")

		for i, d := range data {
			startTime := time.Now()
			log.Printf("Working on %d task", i)

			records, err := a.service.Export(*d.CSVLink, *d.KoboToken)
			if err != nil {
				log.Println(err)
				continue
			}

			values := a.service.Converter(records)

			err = a.service.Importer(*d.APIKey, *d.SpreadSheetID, *d.SheetName, values)
			if err != nil {
				log.Println(err)
				continue
			}
			endTime := time.Now()
			totalTime := endTime.Sub(startTime)
			log.Printf("Task %d completed. Time spent %g sec", i, totalTime.Seconds())
		}
		iterationEndTime := time.Now()
		iterationTime := iterationEndTime.Sub(iterationStartTime)
		log.Printf("Iteration completed. Time spent %g sec", iterationTime.Seconds())
		time.Sleep(1 * time.Hour)
	}
}
