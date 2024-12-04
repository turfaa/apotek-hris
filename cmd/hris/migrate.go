package hris

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/turfaa/apotek-hris/internal/config"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  `Manage database migrations (up/down)`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run migrations up",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(configFiles...)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		m, err := getMigrator(cfg)
		if err != nil {
			log.Fatal(err)
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("Migration completed successfully")
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback migrations",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(configFiles...)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		m, err := getMigrator(cfg)
		if err != nil {
			log.Fatal(err)
		}

		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("Rollback completed successfully")
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	rootCmd.AddCommand(migrateCmd)
}

func getMigrator(cfg config.Config) (*migrate.Migrate, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode)

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error creating migrator: %w", err)
	}
	return m, nil
}
