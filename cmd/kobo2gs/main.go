package main

import (
	"github.com/rostis232/kobo2googlesheet-db/internal/pkg/app"
	"log"

	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/spf13/viper"
)

func main() {
	if err := initConfig(); err != nil {
		log.Fatalf("Error while db config loading: %s\n", err)
	}
	dbconf := repository.Config{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetString("db.port"),
		Username: viper.GetString("db.username"),
		Password: viper.GetString("db.password"),
		DBName:   viper.GetString("db.dbname"),
	}

	a, err := app.NewApp(dbconf)
	if err != nil {
		log.Fatalf("Error while creating new app: %s\n", err)
	}
	if a.Run(viper.GetString("app.sleep-time")) != nil {
		log.Fatalf("Error while running app: %s\n", err)
	}

}

func initConfig() error {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
