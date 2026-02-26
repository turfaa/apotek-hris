package attendance

import (
	"context"
	"fmt"
	"slices"

	"github.com/jmoiron/sqlx"

	"github.com/turfaa/apotek-hris/pkg/validatorx"
	"github.com/turfaa/go-date"
)

type Service struct {
	db *DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: &DB{db: db}}
}

func (s *Service) GetAttendancesBetweenDates(ctx context.Context, from date.Date, to date.Date) ([]Attendance, error) {
	attendances, err := s.db.GetAttendancesBetweenDates(ctx, from, to)
	if err != nil {
		return []Attendance{}, fmt.Errorf("get attendances between dates from db: %w", err)
	}

	return attendances, nil
}

func (s *Service) GetEmployeeAttendancesBetweenDates(ctx context.Context, employeeID int64, from date.Date, to date.Date) ([]Attendance, error) {
	attendances, err := s.db.GetEmployeeAttendancesBetweenDates(ctx, employeeID, from, to)
	if err != nil {
		return []Attendance{}, fmt.Errorf("get employee attendances between dates from db: %w", err)
	}

	return attendances, nil
}

func (s *Service) UpsertAttendance(ctx context.Context, request UpsertAttendanceRequest) (Attendance, error) {
	if err := validatorx.Validate(request); err != nil {
		return Attendance{}, fmt.Errorf("invalid request: %w", err)
	}

	attendance, err := s.db.UpsertAttendance(
		ctx,
		request.EmployeeID,
		request.Date,
		request.TypeID,
		request.OvertimeHours,
	)
	if err != nil {
		return Attendance{}, fmt.Errorf("upsert attendance in db: %w", err)
	}

	return attendance, nil
}

func (s *Service) GetAttendanceTypes(ctx context.Context) ([]Type, error) {
	attendanceTypes, err := s.db.GetAttendanceTypes(ctx)
	if err != nil {
		return []Type{}, fmt.Errorf("get attendance types from db: %w", err)
	}

	return attendanceTypes, nil
}

func (s *Service) CreateAttendanceType(ctx context.Context, request CreateAttendanceTypeRequest) (Type, error) {
	if err := validatorx.Validate(request); err != nil {
		return Type{}, fmt.Errorf("invalid request: %w", err)
	}

	if !slices.Contains(PayableTypes(), request.PayableType) {
		return Type{}, fmt.Errorf("invalid payable type: %s", request.PayableType)
	}

	attendanceType, err := s.db.CreateAttendanceType(ctx, request.Name, request.PayableType, request.HasQuota)
	if err != nil {
		return Type{}, fmt.Errorf("create attendance type in db: %w", err)
	}

	return attendanceType, nil
}

func (s *Service) GetAllQuotas(ctx context.Context) ([]EmployeeAttendanceQuota, error) {
	quotas, err := s.db.GetAllQuotas(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all quotas from db: %w", err)
	}

	return quotas, nil
}

func (s *Service) GetEmployeeQuotas(ctx context.Context, employeeID int64) ([]EmployeeAttendanceQuota, error) {
	quotas, err := s.db.GetEmployeeQuotas(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get employee quotas from db: %w", err)
	}

	return quotas, nil
}

func (s *Service) EnableAttendanceTypeQuota(ctx context.Context, typeID int64) (Type, error) {
	t, err := s.db.EnableAttendanceTypeQuota(ctx, typeID)
	if err != nil {
		return Type{}, fmt.Errorf("enable attendance type quota in db: %w", err)
	}

	return t, nil
}

func (s *Service) SetEmployeeQuota(ctx context.Context, request SetEmployeeAttendanceQuotaRequest) (EmployeeAttendanceQuota, error) {
	if err := validatorx.Validate(request); err != nil {
		return EmployeeAttendanceQuota{}, fmt.Errorf("invalid request: %w", err)
	}

	quota, err := s.db.UpsertEmployeeQuota(ctx, request.EmployeeID, request.AttendanceTypeID, request.RemainingQuota)
	if err != nil {
		return EmployeeAttendanceQuota{}, fmt.Errorf("upsert employee quota in db: %w", err)
	}

	return quota, nil
}
