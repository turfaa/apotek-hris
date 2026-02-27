package hris

import (
	"fmt"
	"log"

	"github.com/turfaa/apotek-hris/internal/attendance"
	"github.com/turfaa/apotek-hris/internal/config"
	"github.com/turfaa/apotek-hris/pkg/database"

	"github.com/spf13/cobra"
)

var (
	attendanceTypeID int64
	quotaIncrement   int
)

var attendanceCmd = &cobra.Command{
	Use:   "attendance",
	Short: "Attendance management commands",
}

var increaseQuotaCmd = &cobra.Command{
	Use:   "increase-quota",
	Short: "Increase attendance type quota for all employees",
	Long:  `Increases the remaining quota of a specific attendance type by a given amount for all employees. Employees without an existing quota record will have one created.`,
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

		affected, err := svc.IncrementQuotaForAllEmployees(cmd.Context(), attendanceTypeID, quotaIncrement)
		if err != nil {
			log.Fatalf("Failed to increase quota: %v", err)
		}

		fmt.Printf("Successfully increased quota by %d for attendance type %d. Affected %d employees.\n", quotaIncrement, attendanceTypeID, affected)
	},
}

func init() {
	increaseQuotaCmd.Flags().Int64Var(&attendanceTypeID, "type-id", 0, "Attendance type ID")
	increaseQuotaCmd.Flags().IntVar(&quotaIncrement, "increment", 0, "Amount to increase quota by")
	_ = increaseQuotaCmd.MarkFlagRequired("type-id")
	_ = increaseQuotaCmd.MarkFlagRequired("increment")

	attendanceCmd.AddCommand(increaseQuotaCmd)
	rootCmd.AddCommand(attendanceCmd)
}
