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
		request.OperatorEmployeeID,
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

	attendanceType, err := s.db.CreateAttendanceType(ctx, request.Name, request.PayableType)
	if err != nil {
		return Type{}, fmt.Errorf("create attendance type in db: %w", err)
	}

	return attendanceType, nil
}
