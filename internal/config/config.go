package config

import (
	"fmt"

	"github.com/turfaa/apotek-hris/internal/salary"
	"github.com/turfaa/apotek-hris/internal/salary/payslip"
	"github.com/turfaa/apotek-hris/pkg/database"
	"github.com/turfaa/apotek-hris/pkg/server"
	"github.com/turfaa/apotek-hris/pkg/validatorx"
	"github.com/turfaa/apotek-hris/pkg/vmedisproxy"

	"github.com/spf13/viper"
)

type Config struct {
	Database database.Config `mapstructure:"database" validate:"required"`
	Server   server.Config   `mapstructure:"server" validate:"required"`

	// Pharmacy is printed on the letterhead of pay slips. It is optional:
	// HRIS starts without it, and pay slip PDFs are refused until it is set.
	Pharmacy payslip.Letterhead `mapstructure:"pharmacy"`

	// VmedisProxy is the vmedis-proxy-api that salary journal entries are
	// posted to Vmedis through.
	VmedisProxy vmedisproxy.Config `mapstructure:"vmedis_proxy"`

	// Journal configures posting salary snapshots to the Vmedis journal.
	Journal salary.JournalConfig `mapstructure:"journal"`
}

// defaults are the values of the keys config files don't set.
var defaults = map[string]any{
	"vmedis_proxy.base_url":                "http://backend-server.vmedis/api",
	"journal.accounts.bank":                "111004", // Bank OCBC
	"journal.accounts.salary_expense":      "610002", // Gaji Karyawan
	"journal.accounts.employee_receivable": "171001", // Piutang Pegawai
}

func Load(configPaths ...string) (Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	for key, value := range defaults {
		v.SetDefault(key, value)
	}

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
