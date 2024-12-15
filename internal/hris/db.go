package hris

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type DB struct {
	db *sqlx.DB
}

type Queryer interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Rebind(query string) string
}

func New(db *sqlx.DB) *DB {
	return &DB{db: db}
}

func (d *DB) GetEmployees(ctx context.Context) ([]Employee, error) {
	query := `
	SELECT id, name, shift_fee, created_at, updated_at 
	FROM employees`
	query = d.db.Rebind(query)

	var employees []Employee
	if err := d.db.SelectContext(ctx, &employees, query); err != nil {
		return []Employee{}, fmt.Errorf("select context from db: %w", err)
	}

	return employees, nil
}

func (d *DB) GetEmployee(ctx context.Context, id int64) (Employee, error) {
	query := `
	SELECT id, name, shift_fee, created_at, updated_at 
	FROM employees 
	WHERE id = ?`
	query = d.db.Rebind(query)
	args := []any{id}

	var employee Employee
	if err := d.db.GetContext(ctx, &employee, query, args...); err != nil {
		return Employee{}, fmt.Errorf("get context from db: %w", err)
	}

	return employee, nil
}

func (d *DB) CreateEmployee(ctx context.Context, request CreateEmployeeRequest) (Employee, error) {
	query := `
	INSERT INTO employees (name, shift_fee) 
	VALUES (?, ?) 
	RETURNING id, name, shift_fee, created_at, updated_at`
	query = d.db.Rebind(query)
	args := []any{request.Name, request.ShiftFee}

	var employee Employee
	if err := d.db.GetContext(ctx, &employee, query, args...); err != nil {
		return Employee{}, fmt.Errorf("get context from db: %w", err)
	}

	return employee, nil
}

func (d *DB) UpdateEmployeeShiftFee(ctx context.Context, id int64, shiftFee decimal.Decimal) (Employee, error) {
	query := `
	UPDATE employees 
	SET shift_fee = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ? 
	RETURNING id, name, shift_fee, created_at, updated_at`
	query = d.db.Rebind(query)
	args := []any{shiftFee, id}

	var employee Employee
	if err := d.db.GetContext(ctx, &employee, query, args...); err != nil {
		return Employee{}, fmt.Errorf("get context from db: %w", err)
	}

	return employee, nil
}

func (d *DB) CreateLeaveBalanceChange(ctx context.Context, employeeID int64, changeAmount int, description string) error {
	query := `
	INSERT INTO leave_balance_changes (employee_id, change_amount, description)
	VALUES (?, ?, ?)`
	query = d.db.Rebind(query)
	args := []any{employeeID, changeAmount, description}

	if _, err := d.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec context to db: %w", err)
	}

	return nil
}

func (d *DB) GetWorkLogsBetween(ctx context.Context, startDate time.Time, endDate time.Time) ([]WorkLog, error) {
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	query := `
	SELECT 
		wl.id, wl.patient_name, wl.created_at, wl.deleted_at, wl.deleted_by,
		e.id AS "employee.id",
		e.name AS "employee.name",
		e.shift_fee AS "employee.shift_fee",
		e.created_at AS "employee.created_at",
		e.updated_at AS "employee.updated_at"
	FROM work_logs wl
	JOIN employees e ON wl.employee_id = e.id
	WHERE wl.deleted_at IS NULL
	AND wl.created_at BETWEEN ? AND ?`
	query = d.db.Rebind(query)
	args := []any{startDate, endDate}

	var workLogs []WorkLog
	if err := d.db.SelectContext(ctx, &workLogs, query, args...); err != nil {
		return []WorkLog{}, fmt.Errorf("select context from db: %w", err)
	}

	if len(workLogs) == 0 {
		return nil, nil
	}

	workLogIDs := make([]int64, len(workLogs))
	for i, workLog := range workLogs {
		workLogIDs[i] = workLog.ID
	}

	workLogUnitsByWorkLogID, err := d.GetWorkLogUnitsByWorkLogIDs(ctx, workLogIDs)
	if err != nil {
		return []WorkLog{}, fmt.Errorf("get work log units by work log ids: %w", err)
	}

	for i, workLog := range workLogs {
		workLog.Units = workLogUnitsByWorkLogID[workLog.ID]
		workLogs[i] = workLog
	}

	return workLogs, nil
}

func (d *DB) GetWorkLog(ctx context.Context, id int64) (WorkLog, error) {
	query := `
	SELECT 
		wl.id, wl.patient_name, wl.created_at, wl.deleted_at, wl.deleted_by,
		e.id AS "employee.id",
		e.name AS "employee.name",
		e.shift_fee AS "employee.shift_fee",
		e.created_at AS "employee.created_at",
		e.updated_at AS "employee.updated_at"
	FROM work_logs wl
	JOIN employees e ON wl.employee_id = e.id
	WHERE wl.deleted_at IS NULL AND wl.id = ?`
	query = d.db.Rebind(query)
	args := []any{id}

	var workLog WorkLog
	if err := d.db.GetContext(ctx, &workLog, query, args...); err != nil {
		return WorkLog{}, fmt.Errorf("get context from db: %w", err)
	}

	workLogUnits, err := d.GetWorkLogUnitsByWorkLogIDs(ctx, []int64{id})
	if err != nil {
		return WorkLog{}, fmt.Errorf("get work log units by work log ids: %w", err)
	}
	workLog.Units = workLogUnits[id]

	return workLog, nil
}

func (d *DB) CreateWorkLog(ctx context.Context, request CreateWorkLogRequest) (workLog WorkLog, returnedErr error) {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return WorkLog{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if returnedErr != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			returnedErr = fmt.Errorf("panic: %v", r)
		}
	}()

	query := `
	WITH inserted_work_log AS (
		INSERT INTO work_logs (employee_id, patient_name)
		VALUES (?, ?)
		RETURNING id, employee_id, patient_name, created_at
	)
	SELECT 
		iwl.id AS "id",
		iwl.patient_name AS "patient_name",
		iwl.created_at AS "created_at",
		e.id AS "employee.id",
		e.name AS "employee.name",
		e.shift_fee AS "employee.shift_fee",
		e.created_at AS "employee.created_at",
		e.updated_at AS "employee.updated_at"
	FROM inserted_work_log iwl
	JOIN employees e ON iwl.employee_id = e.id`
	query = d.db.Rebind(query)
	args := []any{request.EmployeeID, request.PatientName}

	if err := tx.GetContext(ctx, &workLog, query, args...); err != nil {
		return WorkLog{}, fmt.Errorf("get context from db: %w", err)
	}

	workLog.Units, err = d.CreateWorkLogUnitsWithQueryer(ctx, tx, workLog.ID, request.Units)
	if err != nil {
		return WorkLog{}, fmt.Errorf("create work log units: %w", err)
	}

	return workLog, nil
}

func (d *DB) CreateWorkLogUnitsWithQueryer(ctx context.Context, queryer Queryer, workLogID int64, units []CreateWorkLogUnitRequest) ([]WorkLogUnit, error) {
	query := `
	WITH inserted_work_log_units AS (
		INSERT INTO work_log_units (work_log_id, work_type_id, work_outcome, work_multiplier)
		SELECT 
			?, -- work_log_id
			t.work_type_id,
			t.work_outcome,
			wt.multiplier -- Set work_multiplier from work type's multiplier
		FROM unnest(?::bigint[], ?::text[]) AS t(work_type_id, work_outcome)
		JOIN work_types wt ON t.work_type_id = wt.id
		RETURNING id, work_type_id, work_outcome, work_multiplier
	)
	SELECT 
		iwl.id AS "id", 
		iwl.work_outcome AS "work_outcome",
		iwl.work_multiplier AS "work_multiplier",
		wt.id AS "work_type.id",
		wt.name AS "work_type.name",
		wt.outcome_unit AS "work_type.outcome_unit",
		wt.multiplier AS "work_type.multiplier",
		wt.notes AS "work_type.notes"
	FROM inserted_work_log_units iwl
	JOIN work_types wt ON iwl.work_type_id = wt.id`
	query = queryer.Rebind(query)

	workTypeIDs := make([]int64, len(units))
	workOutcomes := make([]string, len(units))
	for i, unit := range units {
		workTypeIDs[i] = unit.WorkTypeID
		workOutcomes[i] = unit.WorkOutcome
	}

	args := []any{workLogID, workTypeIDs, workOutcomes}

	var workLogUnits []WorkLogUnit
	if err := queryer.SelectContext(ctx, &workLogUnits, query, args...); err != nil {
		return []WorkLogUnit{}, fmt.Errorf("select context from db: %w", err)
	}

	return workLogUnits, nil
}

func (d *DB) GetWorkLogUnitsByWorkLogID(ctx context.Context, workLogID int64) ([]WorkLogUnit, error) {
	workLogUnits, err := d.GetWorkLogUnitsByWorkLogIDs(ctx, []int64{workLogID})
	if err != nil {
		return []WorkLogUnit{}, fmt.Errorf("get work log units by work log ids: %w", err)
	}

	return workLogUnits[workLogID], nil
}

func (d *DB) GetWorkLogUnitsByWorkLogIDs(ctx context.Context, workLogIDs []int64) (map[int64][]WorkLogUnit, error) {
	query := `
	SELECT 
		wlu.id AS "id",
		wlu.work_outcome AS "work_outcome",
		wlu.work_log_id,
		wlu.work_multiplier,
		wlu.deleted_at,
		wlu.deleted_by,
		wt.id AS "work_type.id",
		wt.name AS "work_type.name",
		wt.outcome_unit AS "work_type.outcome_unit",
		wt.multiplier AS "work_type.multiplier",
		wt.notes AS "work_type.notes"
	FROM work_log_units wlu
	JOIN work_types wt ON wlu.work_type_id = wt.id
	WHERE wlu.deleted_at IS NULL AND wlu.work_log_id = ANY(?)`
	query = d.db.Rebind(query)
	args := []any{workLogIDs}

	type workLogUnitWithWorkLogID struct {
		WorkLogUnit
		WorkLogID int64 `db:"work_log_id"`
	}

	var workLogUnits []workLogUnitWithWorkLogID
	if err := d.db.SelectContext(ctx, &workLogUnits, query, args...); err != nil {
		return nil, fmt.Errorf("select context from db: %w", err)
	}

	result := make(map[int64][]WorkLogUnit)
	for _, workLogUnit := range workLogUnits {
		result[workLogUnit.WorkLogID] = append(result[workLogUnit.WorkLogID], workLogUnit.WorkLogUnit)
	}

	return result, nil
}

func (d *DB) GetWorkTypes(ctx context.Context) ([]WorkType, error) {
	query := `
	SELECT id, name, outcome_unit, multiplier, notes
	FROM work_types`
	query = d.db.Rebind(query)

	var workTypes []WorkType
	if err := d.db.SelectContext(ctx, &workTypes, query); err != nil {
		return []WorkType{}, fmt.Errorf("select context from db: %w", err)
	}

	return workTypes, nil
}

func (d *DB) GetWorkType(ctx context.Context, id int64) (WorkType, error) {
	return d.GetWorkTypeQueryer(ctx, d.db, id)
}

func (d *DB) GetWorkTypeQueryer(ctx context.Context, queryer Queryer, id int64) (WorkType, error) {
	query := `
	SELECT id, name, outcome_unit, multiplier
	FROM work_types
	WHERE id = ?`
	query = queryer.Rebind(query)
	args := []any{id}

	var workType WorkType
	if err := queryer.GetContext(ctx, &workType, query, args...); err != nil {
		return WorkType{}, fmt.Errorf("get context from db: %w", err)
	}

	return workType, nil
}

func (d *DB) CreateWorkType(ctx context.Context, request CreateWorkTypeRequest) (WorkType, error) {
	query := `
	INSERT INTO work_types (name, outcome_unit, multiplier, notes)
	VALUES (?, ?, ?, ?)
	RETURNING id, name, outcome_unit, multiplier, notes`
	query = d.db.Rebind(query)
	args := []any{request.Name, request.OutcomeUnit, request.Multiplier, request.Notes}

	var workType WorkType
	if err := d.db.GetContext(ctx, &workType, query, args...); err != nil {
		return WorkType{}, fmt.Errorf("get context from db: %w", err)
	}

	return workType, nil
}

func (d *DB) DeleteWorkLog(ctx context.Context, id int64, employeeID int64) error {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	if err := d.softDeleteWorkLogUnits(ctx, tx, id, employeeID); err != nil {
		return fmt.Errorf("soft delete work log units: %w", err)
	}

	if err := d.softDeleteWorkLog(ctx, tx, id, employeeID); err != nil {
		return fmt.Errorf("soft delete work log: %w", err)
	}

	return nil
}

func (d *DB) softDeleteWorkLogUnits(ctx context.Context, tx *sqlx.Tx, workLogID int64, employeeID int64) error {
	query := `
	UPDATE work_log_units 
	SET deleted_at = CURRENT_TIMESTAMP, deleted_by = ?
	WHERE work_log_id = ? AND deleted_at IS NULL`
	query = tx.Rebind(query)
	args := []any{employeeID, workLogID}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec context to db: %w", err)
	}

	return nil
}

func (d *DB) softDeleteWorkLog(ctx context.Context, tx *sqlx.Tx, id int64, employeeID int64) error {
	query := `
	UPDATE work_logs 
	SET deleted_at = CURRENT_TIMESTAMP, deleted_by = ?
	WHERE id = ? AND deleted_at IS NULL`
	query = tx.Rebind(query)
	args := []any{employeeID, id}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec context to db: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrWorkLogNotFound
	}

	return nil
}
