package hris

import (
	"context"
	"fmt"
	"time"

	"github.com/turfaa/apotek-hris/pkg/validatorx"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: &DB{db: db}}
}

func (s *Service) GetEmployees(ctx context.Context) ([]Employee, error) {
	employees, err := s.db.GetEmployees(ctx)
	if err != nil {
		return []Employee{}, fmt.Errorf("get employees from db: %w", err)
	}

	return employees, nil
}

func (s *Service) CreateEmployee(ctx context.Context, request CreateEmployeeRequest) (Employee, error) {
	if err := validatorx.Validate(request); err != nil {
		return Employee{}, fmt.Errorf("invalid request: %w", err)
	}

	employee, err := s.db.CreateEmployee(ctx, request)
	if err != nil {
		return Employee{}, fmt.Errorf("create employee in db: %w", err)
	}

	return employee, nil
}

func (s *Service) GetWorkTypes(ctx context.Context) ([]WorkType, error) {
	workTypes, err := s.db.GetWorkTypes(ctx)
	if err != nil {
		return []WorkType{}, fmt.Errorf("get work types from db: %w", err)
	}

	return workTypes, nil
}

func (s *Service) CreateWorkType(ctx context.Context, request CreateWorkTypeRequest) (WorkType, error) {
	if err := validatorx.Validate(request); err != nil {
		return WorkType{}, fmt.Errorf("invalid request: %w", err)
	}

	workType, err := s.db.CreateWorkType(ctx, request)
	if err != nil {
		return WorkType{}, fmt.Errorf("create work type in db: %w", err)
	}

	return workType, nil
}

func (s *Service) GetWorkLogsBetween(ctx context.Context, startDate time.Time, endDate time.Time) ([]WorkLog, error) {
	workLogs, err := s.db.GetWorkLogsBetween(ctx, startDate, endDate)
	if err != nil {
		return []WorkLog{}, fmt.Errorf("get work logs from db: %w", err)
	}

	return workLogs, nil
}

func (s *Service) GetWorkLog(ctx context.Context, workLogID int64) (WorkLog, error) {
	workLog, err := s.db.GetWorkLog(ctx, workLogID)
	if err != nil {
		return WorkLog{}, fmt.Errorf("get work log from db: %w", err)
	}

	return workLog, nil
}

func (s *Service) CreateWorkLog(ctx context.Context, request CreateWorkLogRequest) (WorkLog, error) {
	if err := validatorx.Validate(request); err != nil {
		return WorkLog{}, fmt.Errorf("invalid request: %w", err)
	}

	workLog, err := s.db.CreateWorkLog(ctx, request)
	if err != nil {
		return WorkLog{}, fmt.Errorf("create work log in db: %w", err)
	}

	return workLog, nil
}
