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

	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("d.db.ExecContext: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected")
	}

	return nil
}
