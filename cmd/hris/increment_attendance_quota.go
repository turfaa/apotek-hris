package hris

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/turfaa/apotek-hris/internal/attendance"
	"github.com/turfaa/apotek-hris/internal/config"
	"github.com/turfaa/apotek-hris/pkg/database"
)

var (
	attendanceTypeID int64
	quotaIncrement   int
)

var incrementAttendanceQuotaCmd = &cobra.Command{
	Use:   "increment-attendance-quota",
	Short: "Increment attendance quota for all available employees",
	Long:  `Increases an attendance type's quota by a specified number for all employees with show_in_attendances enabled.`,
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

		svc := attendance.NewService(db)

		count, err := svc.IncrementQuotaForAllEmployees(cmd.Context(), attendanceTypeID, quotaIncrement)
		if err != nil {
			log.Fatalf("Failed to increment attendance quota: %v", err)
		}

		log.Printf("Successfully incremented quota by %d for %d employees (attendance type ID: %d)", quotaIncrement, count, attendanceTypeID)
	},
}

func init() {
	incrementAttendanceQuotaCmd.Flags().Int64Var(&attendanceTypeID, "type-id", 0, "Attendance type ID")
	incrementAttendanceQuotaCmd.Flags().IntVar(&quotaIncrement, "increment", 0, "Number to increment the quota by")
	incrementAttendanceQuotaCmd.MarkFlagRequired("type-id")
	incrementAttendanceQuotaCmd.MarkFlagRequired("increment")
	rootCmd.AddCommand(incrementAttendanceQuotaCmd)
}
