package attendance

import (
	"fmt"
	"log"

	attendancesvc "github.com/turfaa/apotek-hris/internal/attendance"
	"github.com/turfaa/apotek-hris/internal/config"
	"github.com/turfaa/apotek-hris/internal/hris"
	"github.com/turfaa/apotek-hris/pkg/database"

	"github.com/spf13/cobra"
)

var (
	attendanceTypeID int64
	quotaIncrement   int
)

var increaseQuotaCmd = &cobra.Command{
	Use:   "increase-quota",
	Short: "Increase attendance type quota for all employees",
	Long:  `Increases the remaining quota of a specific attendance type by a given amount for all employees. Employees without an existing quota record will have one created.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFiles, err := cmd.Root().Flags().GetStringSlice("config")
		if err != nil {
			log.Fatalf("Failed to get config flag: %v", err)
		}

		cfg, err := config.Load(configFiles...)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		ctx := cmd.Context()

		db, err := database.NewPostgresConnection(ctx, cfg.Database)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		hrisSvc := hris.NewService(db)
		employees, err := hrisSvc.GetEmployees(ctx)
		if err != nil {
			log.Fatalf("Failed to get employees: %v", err)
		}

		employeeIDs := make([]int64, len(employees))
		for i, e := range employees {
			employeeIDs[i] = e.ID
		}

		attendanceSvc := attendancesvc.NewService(db)
		affected, err := attendanceSvc.IncrementQuotaForEmployees(ctx, employeeIDs, attendanceTypeID, quotaIncrement)
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
}
