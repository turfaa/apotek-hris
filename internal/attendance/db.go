package attendance

import (
	"context"
	"database/sql"
	"errors"
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

func newDB(db *sqlx.DB) *DB {
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
			at.has_quota AS "type.has_quota",
			at.created_at AS "type.created_at",
			at.updated_at AS "type.updated_at",
			a.overtime_hours,
			a.created_at,
			a.updated_at
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

func (d *DB) GetEmployeeAttendancesBetweenDates(ctx context.Context, employeeID int64, from date.Date, to date.Date) ([]Attendance, error) {
	query := `
		SELECT
			a.id,
			a.employee_id,
			a.date,
			at.id AS "type.id",
			at.name AS "type.name",
			at.payable_type AS "type.payable_type",
			at.has_quota AS "type.has_quota",
			at.created_at AS "type.created_at",
			at.updated_at AS "type.updated_at",
			a.overtime_hours,
			a.created_at,
			a.updated_at
		FROM attendances a
		JOIN attendance_types at ON a.type_id = at.id
		WHERE
			a.employee_id = ? AND
			a.date BETWEEN ? AND ?
	`

	query = d.db.Rebind(query)
	args := []any{employeeID, from, to}

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
			at.has_quota AS "type.has_quota",
			at.created_at AS "type.created_at",
			at.updated_at AS "type.updated_at",
			a.overtime_hours,
			a.created_at,
			a.updated_at
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

// getAttendanceTypeWithSelector fetches a single attendance type by ID within a transaction or db.
func (d *DB) getAttendanceTypeWithSelector(ctx context.Context, selector SelectorContext, typeID int64) (Type, error) {
	query := selector.Rebind(`
		SELECT id, name, payable_type, has_quota, created_at, updated_at
		FROM attendance_types
		WHERE id = ?
	`)

	var t Type
	if err := selector.GetContext(ctx, &t, query, typeID); err != nil {
		return Type{}, fmt.Errorf("selector.GetContext: %w", err)
	}

	return t, nil
}

// getCurrentQuota returns the current remaining_quota for an employee+type pair within a transaction.
// Returns 0 and sql.ErrNoRows if no quota record exists.
func (d *DB) getCurrentQuota(ctx context.Context, tx *sqlx.Tx, employeeID int64, typeID int64) (int, error) {
	query := tx.Rebind(`
		SELECT remaining_quota FROM employee_attendance_quotas
		WHERE employee_id = ? AND attendance_type_id = ?
	`)

	var quota int
	if err := tx.GetContext(ctx, &quota, query, employeeID, typeID); err != nil {
		return 0, err
	}

	return quota, nil
}

// insertQuotaAuditLog inserts an audit log entry for a quota change within a transaction.
func (d *DB) insertQuotaAuditLog(ctx context.Context, tx *sqlx.Tx, employeeID int64, typeID int64, previousQuota int, newQuota int, reason QuotaAuditReason) error {
	query := tx.Rebind(`
		INSERT INTO attendance_quota_audit_logs (employee_id, attendance_type_id, previous_quota, new_quota, reason)
		VALUES (?, ?, ?, ?, ?)
	`)

	if _, err := tx.ExecContext(ctx, query, employeeID, typeID, previousQuota, newQuota, reason); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}

	return nil
}

// decrementQuota atomically decrements remaining_quota by 1 only if it is > 0.
// Returns ErrQuotaExhausted if no row was updated (quota is zero or no record exists).
// Also inserts an audit log entry.
func (d *DB) decrementQuota(ctx context.Context, tx *sqlx.Tx, employeeID int64, typeID int64) error {
	previousQuota, err := d.getCurrentQuota(ctx, tx, employeeID, typeID)
	if err != nil {
		return ErrQuotaExhausted
	}

	if previousQuota <= 0 {
		return ErrQuotaExhausted
	}

	query := tx.Rebind(`
		UPDATE employee_attendance_quotas
		SET remaining_quota = remaining_quota - 1, updated_at = NOW()
		WHERE employee_id = ? AND attendance_type_id = ? AND remaining_quota > 0
	`)

	result, err := tx.ExecContext(ctx, query, employeeID, typeID)
	if err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("result.RowsAffected: %w", err)
	}

	if rows == 0 {
		return ErrQuotaExhausted
	}

	if err := d.insertQuotaAuditLog(ctx, tx, employeeID, typeID, previousQuota, previousQuota-1, QuotaAuditReasonAttendanceDeduction); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

// incrementQuota restores one unit of quota. It is a no-op if no record exists.
// Also inserts an audit log entry when a quota record exists.
func (d *DB) incrementQuota(ctx context.Context, tx *sqlx.Tx, employeeID int64, typeID int64) error {
	previousQuota, err := d.getCurrentQuota(ctx, tx, employeeID, typeID)
	if err != nil {
		// No record exists, nothing to increment.
		return nil
	}

	query := tx.Rebind(`
		UPDATE employee_attendance_quotas
		SET remaining_quota = remaining_quota + 1, updated_at = NOW()
		WHERE employee_id = ? AND attendance_type_id = ?
	`)

	if _, err := tx.ExecContext(ctx, query, employeeID, typeID); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}

	if err := d.insertQuotaAuditLog(ctx, tx, employeeID, typeID, previousQuota, previousQuota+1, QuotaAuditReasonAttendanceRestoration); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

func (d *DB) UpsertAttendance(
	ctx context.Context,
	employeeID int64,
	date date.Date,
	typeID int64,
	overtimeHours decimal.Decimal,
) (Attendance, error) {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return Attendance{}, fmt.Errorf("d.db.BeginTxx: %w", err)
	}

	defer tx.Rollback()

	// Determine the existing attendance type (if any).
	existingTypeID := int64(0)
	existingTypeHasQuota := false

	existing, err := d.GetEmployeeAttendanceAtDateWithSelector(ctx, tx, employeeID, date)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Attendance{}, fmt.Errorf("get existing attendance: %w", err)
	}
	if err == nil {
		existingTypeID = existing.Type.ID
		existingTypeHasQuota = existing.Type.HasQuota
	}

	// Handle quota changes only when the attendance type is changing (or this is a new record).
	if existingTypeID != typeID {
		newType, err := d.getAttendanceTypeWithSelector(ctx, tx, typeID)
		if err != nil {
			return Attendance{}, fmt.Errorf("get new attendance type: %w", err)
		}

		// Restore quota for the old type before deducting from the new type.
		if existingTypeID != 0 && existingTypeHasQuota {
			if err := d.incrementQuota(ctx, tx, employeeID, existingTypeID); err != nil {
				return Attendance{}, fmt.Errorf("restore quota for old type: %w", err)
			}
		}

		// Deduct quota for the new type.
		if newType.HasQuota {
			if err := d.decrementQuota(ctx, tx, employeeID, typeID); err != nil {
				// Transaction will be rolled back by defer, restoring any quota increment above.
				return Attendance{}, fmt.Errorf("deduct quota for new type: %w", err)
			}
		}
	}

	// Upsert the attendance record.
	upsertQuery := tx.Rebind(`
		INSERT INTO attendances (employee_id, date, type_id, overtime_hours, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (employee_id, date) DO UPDATE SET type_id = ?, overtime_hours = ?, updated_at = NOW()
	`)

	if _, err := tx.ExecContext(ctx, upsertQuery, employeeID, date, typeID, overtimeHours, typeID, overtimeHours); err != nil {
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

// EnableAttendanceTypeQuota sets has_quota = true for an attendance type.
// Returns ErrAlreadyHasQuota if the type already has quota enabled.
func (d *DB) EnableAttendanceTypeQuota(ctx context.Context, typeID int64) (Type, error) {
	query := d.db.Rebind(`
		UPDATE attendance_types
		SET has_quota = TRUE, updated_at = NOW()
		WHERE id = ? AND has_quota = FALSE
		RETURNING id, name, payable_type, has_quota, created_at, updated_at
	`)

	var t Type
	err := d.db.GetContext(ctx, &t, query, typeID)
	if errors.Is(err, sql.ErrNoRows) {
		// Either the type doesn't exist or it already has quota enabled.
		// Check which case it is.
		existing, getErr := d.getAttendanceTypeWithSelector(ctx, d.db, typeID)
		if errors.Is(getErr, sql.ErrNoRows) {
			return Type{}, fmt.Errorf("d.db.GetContext: %w", sql.ErrNoRows)
		}
		if getErr != nil {
			return Type{}, fmt.Errorf("get attendance type: %w", getErr)
		}
		if existing.HasQuota {
			return Type{}, ErrAlreadyHasQuota
		}
		return Type{}, fmt.Errorf("d.db.GetContext: %w", err)
	}
	if err != nil {
		return Type{}, fmt.Errorf("d.db.GetContext: %w", err)
	}

	return t, nil
}

func (d *DB) GetAttendanceTypes(ctx context.Context) ([]Type, error) {
	query := `
		SELECT id, name, payable_type, has_quota, created_at, updated_at
		FROM attendance_types
	`

	var types []Type
	if err := d.db.SelectContext(ctx, &types, query); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return types, nil
}

func (d *DB) CreateAttendanceType(ctx context.Context, name string, payableType PayableType, hasQuota bool) (Type, error) {
	query := d.db.Rebind(`
		INSERT INTO attendance_types (name, payable_type, has_quota)
		VALUES (?, ?, ?)
		RETURNING id, name, payable_type, has_quota, created_at, updated_at
	`)

	var t Type
	if err := d.db.GetContext(ctx, &t, query, name, payableType, hasQuota); err != nil {
		return Type{}, fmt.Errorf("d.db.GetContext: %w", err)
	}

	return t, nil
}

// UpsertEmployeeQuota sets the remaining_quota for an employee+attendance type pair.
// This is an administrative operation to allocate quota to an employee.
// Also inserts an audit log entry recording the change.
func (d *DB) UpsertEmployeeQuota(ctx context.Context, employeeID int64, typeID int64, remainingQuota int) (EmployeeAttendanceQuota, error) {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return EmployeeAttendanceQuota{}, fmt.Errorf("d.db.BeginTxx: %w", err)
	}

	defer tx.Rollback()

	// Get the previous quota value (0 if not yet allocated).
	previousQuota, err := d.getCurrentQuota(ctx, tx, employeeID, typeID)
	if err != nil {
		previousQuota = 0
	}

	query := tx.Rebind(`
		INSERT INTO employee_attendance_quotas (employee_id, attendance_type_id, remaining_quota)
		VALUES (?, ?, ?)
		ON CONFLICT (employee_id, attendance_type_id)
		DO UPDATE SET remaining_quota = ?, updated_at = NOW()
		RETURNING id
	`)

	var id int64
	if err := tx.GetContext(ctx, &id, query, employeeID, typeID, remainingQuota, remainingQuota); err != nil {
		return EmployeeAttendanceQuota{}, fmt.Errorf("tx.GetContext: %w", err)
	}

	if err := d.insertQuotaAuditLog(ctx, tx, employeeID, typeID, previousQuota, remainingQuota, QuotaAuditReasonManualSet); err != nil {
		return EmployeeAttendanceQuota{}, fmt.Errorf("insert audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return EmployeeAttendanceQuota{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return d.getEmployeeQuotaByID(ctx, id)
}

func (d *DB) getEmployeeQuotaByID(ctx context.Context, id int64) (EmployeeAttendanceQuota, error) {
	query := d.db.Rebind(`
		SELECT
			q.id,
			q.employee_id,
			at.id AS "attendance_type.id",
			at.name AS "attendance_type.name",
			at.payable_type AS "attendance_type.payable_type",
			at.has_quota AS "attendance_type.has_quota",
			at.created_at AS "attendance_type.created_at",
			at.updated_at AS "attendance_type.updated_at",
			q.remaining_quota,
			q.created_at,
			q.updated_at
		FROM employee_attendance_quotas q
		JOIN attendance_types at ON q.attendance_type_id = at.id
		WHERE q.id = ?
	`)

	var quota EmployeeAttendanceQuota
	if err := d.db.GetContext(ctx, &quota, query, id); err != nil {
		return EmployeeAttendanceQuota{}, fmt.Errorf("d.db.GetContext: %w", err)
	}

	return quota, nil
}

// GetAllQuotas returns all quota allocations across all employees for quota-enabled attendance types.
func (d *DB) GetAllQuotas(ctx context.Context) ([]EmployeeAttendanceQuota, error) {
	query := `
		SELECT
			q.id,
			q.employee_id,
			at.id AS "attendance_type.id",
			at.name AS "attendance_type.name",
			at.payable_type AS "attendance_type.payable_type",
			at.has_quota AS "attendance_type.has_quota",
			at.created_at AS "attendance_type.created_at",
			at.updated_at AS "attendance_type.updated_at",
			q.remaining_quota,
			q.created_at,
			q.updated_at
		FROM employee_attendance_quotas q
		JOIN attendance_types at ON q.attendance_type_id = at.id
		ORDER BY at.name, q.employee_id
	`

	var quotas []EmployeeAttendanceQuota
	if err := d.db.SelectContext(ctx, &quotas, query); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return quotas, nil
}

// GetEmployeeQuotas returns all quota allocations for a given employee.
func (d *DB) GetEmployeeQuotas(ctx context.Context, employeeID int64) ([]EmployeeAttendanceQuota, error) {
	query := d.db.Rebind(`
		SELECT
			q.id,
			q.employee_id,
			at.id AS "attendance_type.id",
			at.name AS "attendance_type.name",
			at.payable_type AS "attendance_type.payable_type",
			at.has_quota AS "attendance_type.has_quota",
			at.created_at AS "attendance_type.created_at",
			at.updated_at AS "attendance_type.updated_at",
			q.remaining_quota,
			q.created_at,
			q.updated_at
		FROM employee_attendance_quotas q
		JOIN attendance_types at ON q.attendance_type_id = at.id
		WHERE q.employee_id = ?
		ORDER BY at.name
	`)

	var quotas []EmployeeAttendanceQuota
	if err := d.db.SelectContext(ctx, &quotas, query, employeeID); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return quotas, nil
}

// GetQuotaAuditLogs returns all audit logs for quota changes, ordered by most recent first.
func (d *DB) GetQuotaAuditLogs(ctx context.Context) ([]QuotaAuditLog, error) {
	query := `
		SELECT
			l.id,
			l.employee_id,
			at.id AS "attendance_type.id",
			at.name AS "attendance_type.name",
			at.payable_type AS "attendance_type.payable_type",
			at.has_quota AS "attendance_type.has_quota",
			at.created_at AS "attendance_type.created_at",
			at.updated_at AS "attendance_type.updated_at",
			l.previous_quota,
			l.new_quota,
			l.reason,
			l.created_at
		FROM attendance_quota_audit_logs l
		JOIN attendance_types at ON l.attendance_type_id = at.id
		ORDER BY l.created_at DESC
	`

	var logs []QuotaAuditLog
	if err := d.db.SelectContext(ctx, &logs, query); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return logs, nil
}

// GetEmployeeQuotaAuditLogs returns audit logs for a specific employee, ordered by most recent first.
func (d *DB) GetEmployeeQuotaAuditLogs(ctx context.Context, employeeID int64) ([]QuotaAuditLog, error) {
	query := d.db.Rebind(`
		SELECT
			l.id,
			l.employee_id,
			at.id AS "attendance_type.id",
			at.name AS "attendance_type.name",
			at.payable_type AS "attendance_type.payable_type",
			at.has_quota AS "attendance_type.has_quota",
			at.created_at AS "attendance_type.created_at",
			at.updated_at AS "attendance_type.updated_at",
			l.previous_quota,
			l.new_quota,
			l.reason,
			l.created_at
		FROM attendance_quota_audit_logs l
		JOIN attendance_types at ON l.attendance_type_id = at.id
		WHERE l.employee_id = ?
		ORDER BY l.created_at DESC
	`)

	var logs []QuotaAuditLog
	if err := d.db.SelectContext(ctx, &logs, query, employeeID); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return logs, nil
}
