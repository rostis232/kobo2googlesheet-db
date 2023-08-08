package main

import (
	"github.com/rostis232/kobo2googlesheet-db/internal/pkg/app"
	"log"

	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/spf13/viper"
)

func main() {
	if err := initConfig(); err != nil {
		log.Fatalln("error while db config loading")
	}
	dbconf := repository.Config{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetString("db.port"),
		Username: viper.GetString("db.username"),
		Password: viper.GetString("db.password"),
		DBName:   viper.GetString("db.dbname"),
	}

	a := app.NewApp(dbconf)
	a.Run()

}

func initConfig() error {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
