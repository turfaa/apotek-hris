package hris

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/turfaa/apotek-hris/internal/config"
	"github.com/turfaa/apotek-hris/pkg/database"
	"github.com/turfaa/apotek-hris/pkg/server"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(configFiles...)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		db, err := database.NewPostgresConnection(cmd.Context(), cfg.Database)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		srv := server.New(cfg.Server, db)

		// Handle graceful shutdown
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		log.Printf("Server is running on %s:%d", cfg.Server.Host, cfg.Server.Port)

		<-done
		log.Print("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}

		fmt.Println("Server exited properly")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
