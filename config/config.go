package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func ReadConfig(path string) {
	viper.SetConfigName("config")
	viper.AddConfigPath(path)
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
}
