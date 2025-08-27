package attendance

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/turfaa/go-date"
)

type SelectorContext interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	Rebind(query string) string
}

type DB struct {
	db *sqlx.DB
}

func NewDB(db *sqlx.DB) *DB {
	return &DB{db: db}
}

func (d *DB) GetAttendancesBetweenDates(ctx context.Context, from date.Date, to date.Date) ([]Attendance, error) {
	query := `
		SELECT 
			a.id, 
			a.employee_id, 
			a.date, 
			at.id AS "type.id", 
			at.name AS "type.name", 
			at.payable_type AS "type.payable_type", 
			at.created_at AS "type.created_at",
			at.updated_at AS "type.updated_at",
			a.overtime_hours, 
			a.created_at, 
			a.updated_at,
			a.last_operator_employee_id
		FROM attendances a
		JOIN attendance_types at ON a.type_id = at.id
		WHERE a.date BETWEEN ? AND ?
	`

	query = d.db.Rebind(query)
	args := []any{from, to}

	var attendances []Attendance
	if err := d.db.SelectContext(ctx, &attendances, query, args...); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return attendances, nil
}

func (d *DB) GetEmployeeAttendanceAtDate(ctx context.Context, employeeID int64, date date.Date) (Attendance, error) {
	return d.GetEmployeeAttendanceAtDateWithSelector(ctx, d.db, employeeID, date)
}

func (d *DB) GetEmployeeAttendanceAtDateWithSelector(ctx context.Context, selector SelectorContext, employeeID int64, date date.Date) (Attendance, error) {
	query := `
		SELECT 
			a.id, 
			a.employee_id, 
			a.date, 
			at.id AS "type.id", 
			at.name AS "type.name", 
			at.payable_type AS "type.payable_type", 
			at.created_at AS "type.created_at",
			at.updated_at AS "type.updated_at",
			a.overtime_hours, 
			a.created_at, 
			a.updated_at,
			a.last_operator_employee_id
		FROM attendances a
		JOIN attendance_types at ON a.type_id = at.id
		WHERE a.employee_id = ? AND a.date = ?
	`

	query = selector.Rebind(query)
	args := []any{employeeID, date}

	var a Attendance
	if err := selector.GetContext(ctx, &a, query, args...); err != nil {
		return Attendance{}, fmt.Errorf("selector.GetContext: %w", err)
	}

	return a, nil
}

func (d *DB) UpsertAttendance(
	ctx context.Context,
	employeeID int64,
	date date.Date,
	typeID int64,
	overtimeHours decimal.Decimal,
	operatorEmployeeID int64,
) (Attendance, error) {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return Attendance{}, fmt.Errorf("d.db.BeginTxx: %w", err)
	}

	defer tx.Rollback()

	query := `
		INSERT INTO attendances (employee_id, date, type_id, overtime_hours, created_at, updated_at, last_operator_employee_id)
		VALUES (?, ?, ?, ?, NOW(), NOW(), ?)
		ON CONFLICT (employee_id, date) DO UPDATE SET type_id = ?, overtime_hours = ?, updated_at = NOW(), last_operator_employee_id = ?
	`

	query = tx.Rebind(query)
	args := []any{employeeID, date, typeID, overtimeHours, operatorEmployeeID, typeID, overtimeHours, operatorEmployeeID}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return Attendance{}, fmt.Errorf("tx.ExecContext: %w", err)
	}

	attendance, err := d.GetEmployeeAttendanceAtDateWithSelector(ctx, tx, employeeID, date)
	if err != nil {
		return Attendance{}, fmt.Errorf("d.GetEmployeeAttendanceAtDateWithSelector: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Attendance{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return attendance, nil
}

func (d *DB) GetAttendanceTypes(ctx context.Context) ([]Type, error) {
	query := `
		SELECT id, name, payable_type, created_at, updated_at
		FROM attendance_types
	`

	var types []Type
	if err := d.db.SelectContext(ctx, &types, query); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return types, nil
}

func (d *DB) CreateAttendanceType(ctx context.Context, name string, payableType PayableType) (Type, error) {
	query := `
		INSERT INTO attendance_types (name, payable_type)
		VALUES (?, ?)
		RETURNING id, name, payable_type, created_at, updated_at
	`

	query = d.db.Rebind(query)
	args := []any{name, payableType}

	var t Type
	if err := d.db.GetContext(ctx, &t, query, args...); err != nil {
		return Type{}, fmt.Errorf("d.db.GetContext: %w", err)
	}

	return t, nil
}
