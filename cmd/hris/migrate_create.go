package hris

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var migrateCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		timestamp := time.Now().UTC().Format("20060102150405")

		// Clean and format the migration name
		name = strings.ToLower(name)
		name = strings.ReplaceAll(name, " ", "_")

		// Create migrations directory if it doesn't exist
		if err := os.MkdirAll("migrations", 0755); err != nil {
			log.Fatalf("Failed to create migrations directory: %v", err)
		}

		// Create up migration
		upFile := filepath.Join("migrations", fmt.Sprintf("%s_%s.up.sql", timestamp, name))
		if err := os.WriteFile(upFile, []byte("-- Write your UP migration SQL here\n"), 0644); err != nil {
			log.Fatalf("Failed to create up migration file: %v", err)
		}

		// Create down migration
		downFile := filepath.Join("migrations", fmt.Sprintf("%s_%s.down.sql", timestamp, name))
		if err := os.WriteFile(downFile, []byte("-- Write your DOWN migration SQL here\n"), 0644); err != nil {
			log.Fatalf("Failed to create down migration file: %v", err)
		}

		fmt.Printf("Created migration files:\n%s\n%s\n", upFile, downFile)
	},
}

func init() {
	migrateCmd.AddCommand(migrateCreateCmd)
}
