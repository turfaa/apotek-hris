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

func Load(configPaths ...string) (Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	for _, configPath := range configPaths {
		v.SetConfigFile(configPath)
		if err := v.MergeInConfig(); err != nil {
			return Config{}, fmt.Errorf("error reading config file %s: %w", configPath, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := validatorx.Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("error validating config: %w", err)
	}

	return cfg, nil
}
