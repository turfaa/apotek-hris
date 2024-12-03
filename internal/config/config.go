package config

import (
	"fmt"

	"github.com/turfaa/apotek-hris/pkg/database"
	"github.com/turfaa/apotek-hris/pkg/server"

	"github.com/spf13/viper"
)

type Config struct {
	Database database.Config
	Server   server.Config
}

func Load(configPath string) (Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return cfg, nil
}
