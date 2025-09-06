package salary

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/turfaa/apotek-hris/pkg/timex"
)

type DB struct {
	db *sqlx.DB
}

func NewDB(db *sqlx.DB) *DB {
	return &DB{db: db}
}

func (d *DB) GetEmployeeStaticComponents(ctx context.Context, employeeID int64) ([]StaticComponent, error) {
	query := `
		SELECT id, employee_id, description, amount, multiplier, created_at
		FROM salary_static_components
		WHERE employee_id = ?
		ORDER BY id ASC
	`

	query = d.db.Rebind(query)
	args := []any{employeeID}

	var staticComponents []StaticComponent
	if err := d.db.SelectContext(ctx, &staticComponents, query, args...); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return staticComponents, nil
}

func (d *DB) CreateStaticComponent(
	ctx context.Context,
	employeeID int64,
	component Component,
) (StaticComponent, error) {
	query := `
		INSERT INTO salary_static_components (employee_id, description, amount, multiplier, created_at)
		VALUES (?, ?, ?, ?, NOW())
		RETURNING id, employee_id, description, amount, multiplier, created_at
	`

	query = d.db.Rebind(query)
	args := []any{employeeID, component.Description, component.Amount, component.Multiplier}

	var staticComponent StaticComponent
	if err := d.db.GetContext(ctx, &staticComponent, query, args...); err != nil {
		return StaticComponent{}, fmt.Errorf("d.db.GetContext: %w", err)
	}

	return staticComponent, nil
}

func (d *DB) DeleteStaticComponent(ctx context.Context, employeeID int64, id int64) error {
	query := `
		DELETE FROM salary_static_components WHERE id = ? AND employee_id = ?
	`
	query = d.db.Rebind(query)
	args := []any{id, employeeID}

	if _, err := d.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec context to db: %w", err)
	}

	return nil
}

func (d *DB) GetEmployeeAdditionalComponents(ctx context.Context, employeeID int64, month timex.Month) ([]AdditionalComponent, error) {
	query := `
		SELECT id, employee_id, month, description, amount, multiplier, created_at
		FROM salary_additional_components
		WHERE employee_id = ? AND month = ?
		ORDER BY id ASC
	`

	query = d.db.Rebind(query)
	args := []any{employeeID, month}

	var additionalComponents []AdditionalComponent
	if err := d.db.SelectContext(ctx, &additionalComponents, query, args...); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return additionalComponents, nil
}

func (d *DB) CreateAdditionalComponent(
	ctx context.Context,
	employeeID int64,
	month timex.Month,
	component Component,
) (AdditionalComponent, error) {
	query := `
		INSERT INTO salary_additional_components (employee_id, month, description, amount, multiplier, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())
		RETURNING id, employee_id, month, description, amount, multiplier, created_at
	`

	query = d.db.Rebind(query)
	args := []any{employeeID, month, component.Description, component.Amount, component.Multiplier}

	var additionalComponent AdditionalComponent
	if err := d.db.GetContext(ctx, &additionalComponent, query, args...); err != nil {
		return AdditionalComponent{}, fmt.Errorf("d.db.GetContext: %w", err)
	}

	return additionalComponent, nil
}

// DeleteAdditionalComponent deletes an additional component by id.
// The employeeID and month are used to verify that the component belongs to the employee and month.
func (d *DB) DeleteAdditionalComponent(ctx context.Context, employeeID int64, month timex.Month, id int64) error {
	query := `
		DELETE FROM salary_additional_components WHERE id = ? AND employee_id = ? AND month = ?
	`

	query = d.db.Rebind(query)
	args := []any{id, employeeID, month}

	if _, err := d.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("d.db.ExecContext: %w", err)
	}

	return nil
}

func (d *DB) GetEmployeeExtraInfos(ctx context.Context, employeeID int64, month timex.Month) ([]ExtraInfo, error) {
	query := `
		SELECT id, employee_id, month, title, description, created_at
		FROM salary_extra_infos
		WHERE employee_id = ? AND month = ?
		ORDER BY id ASC
	`

	query = d.db.Rebind(query)
	args := []any{employeeID, month}

	var extraInfos []ExtraInfo
	if err := d.db.SelectContext(ctx, &extraInfos, query, args...); err != nil {
		return nil, fmt.Errorf("d.db.SelectContext: %w", err)
	}

	return extraInfos, nil
}

func (d *DB) CreateExtraInfo(ctx context.Context, employeeID int64, month timex.Month, title string, description string) (ExtraInfo, error) {
	query := `
		INSERT INTO salary_extra_infos (employee_id, month, title, description, created_at)
		VALUES (?, ?, ?, ?, NOW())
		RETURNING id, employee_id, month, title, description, created_at
	`

	query = d.db.Rebind(query)
	args := []any{employeeID, month, title, description}

	var extraInfo ExtraInfo
	if err := d.db.GetContext(ctx, &extraInfo, query, args...); err != nil {
		return ExtraInfo{}, fmt.Errorf("d.db.GetContext: %w", err)
	}

	return extraInfo, nil
}

func (d *DB) DeleteExtraInfo(ctx context.Context, employeeID int64, month timex.Month, id int64) error {
	query := `
		DELETE FROM salary_extra_infos WHERE id = ? AND employee_id = ? AND month = ?
	`

	query = d.db.Rebind(query)
	args := []any{id, employeeID, month}

	if _, err := d.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("d.db.ExecContext: %w", err)
	}

	return nil
}
