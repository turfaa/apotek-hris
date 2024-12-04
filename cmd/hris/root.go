package hris

import (
	"fmt"
	"os"

	"github.com/turfaa/apotek-hris/internal/config"

	"github.com/spf13/cobra"
)

var (
	configFiles []string
	cfg         config.Config

	rootCmd = &cobra.Command{
		Use:   "apotek-hris",
		Short: "Apotek HRIS - Human Resource Information System",
		Long:  `A comprehensive HRIS system for managing pharmacy staff and operations.`,
	}
)

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&configFiles, "config", "c", []string{"config/config.yaml", "config/secret.yaml"}, "config file paths")

	var err error
	cfg, err = config.Load(configFiles...)
	if err != nil {
		panic(err)
	}
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
