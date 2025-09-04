package salary

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/turfaa/apotek-hris/internal/attendance"
	"github.com/turfaa/apotek-hris/internal/hris"
	"github.com/turfaa/apotek-hris/pkg/timex"
	"golang.org/x/sync/errgroup"
)

var (
	workUnitFee = decimal.NewFromInt(1_000)
	fixedBonus  = decimal.NewFromInt(200_000)
)

type Service struct {
	db                *DB
	hrisService       *hris.Service
	attendanceService *attendance.Service
}

func NewService(db *sqlx.DB, hrisService *hris.Service, attendanceService *attendance.Service) *Service {
	return &Service{
		db:                NewDB(db),
		hrisService:       hrisService,
		attendanceService: attendanceService,
	}
}

func (s *Service) GetSalary(ctx context.Context, employeeID int64, month timex.Month) (Salary, error) {
	monthDateFrom, monthDateTo, err := month.DateRange()
	if err != nil {
		return Salary{}, fmt.Errorf("get month date range: %w", err)
	}

	monthTimeFrom, err := timex.BeginningOfDate(monthDateFrom.String())
	if err != nil {
		return Salary{}, fmt.Errorf("get month time from: %w", err)
	}

	monthTimeTo, err := timex.EndOfDate(monthDateTo.String())
	if err != nil {
		return Salary{}, fmt.Errorf("get month time to: %w", err)
	}

	var (
		employee             hris.Employee
		attendances          []attendance.Attendance
		workLogs             []hris.WorkLog
		additionalComponents []AdditionalComponent
	)

	eg, gCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		var err error
		employee, err = s.hrisService.GetEmployee(gCtx, employeeID)
		if err != nil {
			return fmt.Errorf("get employee from hris service: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		var err error
		attendances, err = s.attendanceService.GetEmployeeAttendancesBetweenDates(gCtx, employeeID, monthDateFrom, monthDateTo)
		if err != nil {
			return fmt.Errorf("get employee attendances between dates from attendance service: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		var err error
		workLogs, err = s.hrisService.GetEmployeeWorkLogsBetween(gCtx, employeeID, monthTimeFrom, monthTimeTo)
		if err != nil {
			return fmt.Errorf("get employee work logs between dates from hris service: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		var err error
		additionalComponents, err = s.db.GetEmployeeAdditionalComponents(gCtx, employeeID, month)
		if err != nil {
			return fmt.Errorf("get employee additional components from db: %w", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return Salary{}, fmt.Errorf("wait for get salary: %w", err)
	}

	attendanceSummary := attendance.CreateEmployeeSummary(attendances)

	return s.calculateSalary(
		employee,
		attendanceSummary,
		workLogs,
		additionalComponents,
	), nil
}

func (s *Service) calculateSalary(
	employee hris.Employee,
	attendanceSummary attendance.EmployeeSummary,
	workLogs []hris.WorkLog,
	additionalComponents []AdditionalComponent,
) Salary {
	components := []Component{
		{
			Description: "Banyak Shift Jaga",
			Amount:      employee.ShiftFee,
			Multiplier:  decimal.NewFromInt(int64(attendanceSummary.WorkingDays)),
		},
		{
			Description: "Banyak Jam Lembur",
			Amount:      s.calculateHourlyOvertimeFee(employee.ShiftFee),
			Multiplier:  attendanceSummary.OvertimeHours,
		},
	}

	benefits := maps.Keys(attendanceSummary.DaysByBenefit)
	for _, benefit := range slices.Sorted(benefits) {
		components = append(components, Component{
			Description: benefit,
			Amount:      employee.ShiftFee,
			Multiplier:  decimal.NewFromInt(int64(attendanceSummary.DaysByBenefit[benefit])),
		})
	}

	totalWorkUnits := decimal.Zero
	for _, workLog := range workLogs {
		for _, unit := range workLog.Units {
			totalWorkUnits = totalWorkUnits.Add(unit.WorkMultiplier)
		}
	}

	components = append(components, Component{
		Description: "Tes dan Resep",
		Amount:      totalWorkUnits.Mul(workUnitFee),
		Multiplier:  decimal.NewFromInt(1),
	})

	return Salary{
		Components:           components,
		AdditionalComponents: additionalComponents,
	}
}

// calculateHourlyOvertimeFee returns (shiftFee + 10.000) / 7.
// This is an existing formula.
func (*Service) calculateHourlyOvertimeFee(shiftFee decimal.Decimal) decimal.Decimal {
	return shiftFee.Add(decimal.NewFromInt(10_000)).Div(decimal.NewFromInt(7)).RoundUp(0)
}

func (s *Service) GetEmployeeAdditionalComponents(ctx context.Context, employeeID int64, month timex.Month) ([]AdditionalComponent, error) {
	return s.db.GetEmployeeAdditionalComponents(ctx, employeeID, month)
}

func (s *Service) CreateAdditionalComponent(ctx context.Context, employeeID int64, month timex.Month, component Component) (AdditionalComponent, error) {
	return s.db.CreateAdditionalComponent(ctx, employeeID, month, component)
}

func (s *Service) DeleteAdditionalComponent(ctx context.Context, employeeID int64, month timex.Month, id int64) error {
	return s.db.DeleteAdditionalComponent(ctx, employeeID, month, id)
}
