package config

import (
	"fmt"

	"github.com/turfaa/apotek-hris/pkg/database"
	"github.com/turfaa/apotek-hris/pkg/server"
	"github.com/turfaa/apotek-hris/pkg/validatorx"

	"github.com/spf13/viper"
)

type Config struct {
	Database database.Config `mapstructure:"database" validate:"required"`
	Server   server.Config   `mapstructure:"server" validate:"required"`
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

	if err := validatorx.Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("error validating config: %w", err)
	}

	return cfg, nil
}
