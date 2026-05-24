package config

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func ReadConfig(path string) {
	viper.SetConfigName("config")
	viper.AddConfigPath(path)
	// VFEEG_BACKEND_DATABASE_PASSWORD -> "database.password" etc.
	// Without these two lines viper.AutomaticEnv() silently fails to
	// resolve env-var overrides because nested keys contain dots that
	// shell env-var names can't carry.
	viper.SetEnvPrefix("VFEEG_BACKEND")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
}
