package hris

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFiles []string

	rootCmd = &cobra.Command{
		Use:   "apotek-hris",
		Short: "Apotek HRIS - Human Resource Information System",
		Long:  `A comprehensive HRIS system for managing pharmacy staff and operations.`,
	}
)

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&configFiles, "config", "c", []string{"config/config.yaml", "config/secret.yaml"}, "config file paths")
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
