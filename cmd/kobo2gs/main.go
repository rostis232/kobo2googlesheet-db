package main

import (
	"github.com/rostis232/kobo2googlesheet-db/internal/pkg/app"
	"github.com/sirupsen/logrus"
	"os"

	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/spf13/viper"
)

func initLogger() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetOutput(os.Stdout)
}

func main() {
	initLogger()
	if err := initConfig(); err != nil {
		logrus.Fatalf("Error while db config loading: %s\n", err)
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
		logrus.Fatalf("Error while creating new app: %s\n", err)
	}
	logrus.Info("Підключено до бази даних!")
	if err := a.Run(viper.GetString("app.sleep-time"), viper.GetString("app.log-level")); err != nil {
		logrus.Fatalf("Error while running app: %s\n", err)
	}

}

func initConfig() error {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
