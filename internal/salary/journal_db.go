package salary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/turfaa/apotek-hris/pkg/timex"
)

const journalEntryColumns = `
	id, snapshot_id, employee_id, month, kind, component_index, description,
	credit_account_code, debit_account_code, amount, journal_date, status,
	attempts, idempotency_key, payload, vmedis_voucher_no, vmedis_id, error,
	posted_by, posted_by_name, posted_at, created_at, updated_at
`

// errUniqueViolation is wrapped by inserts that violate a unique index.
var errUniqueViolation = errors.New("unique violation")

// WithLockedSnapshot runs fn in a transaction holding the snapshot's row
// lock, so the snapshot's journal entries are changed by one request at a
// time and the snapshot isn't deleted meanwhile. It returns sql.ErrNoRows
// when the snapshot doesn't exist.
func (d *DB) WithLockedSnapshot(ctx context.Context, snapshotID int64, fn func(tx *sqlx.Tx) error) error {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("d.db.BeginTxx: %w", err)
	}
	defer tx.Rollback()

	var id int64
	query := tx.Rebind(`SELECT id FROM salary_snapshots WHERE id = ? AND deleted_at IS NULL FOR UPDATE`)
	if err := tx.GetContext(ctx, &id, query, snapshotID); err != nil {
		return fmt.Errorf("lock snapshot %d: %w", snapshotID, err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

// GetJournalEntries returns the snapshot's recorded journal entries, the
// salary entry first.
func (d *DB) GetJournalEntries(ctx context.Context, q sqlx.QueryerContext, snapshotID int64) ([]JournalEntryRecord, error) {
	query := d.db.Rebind(`
		SELECT ` + journalEntryColumns + `
		FROM salary_snapshot_journal_entries
		WHERE snapshot_id = ?
		ORDER BY kind = 'debt', component_index NULLS FIRST, id
	`)

	var records []JournalEntryRecord
	if err := sqlx.SelectContext(ctx, q, &records, query, snapshotID); err != nil {
		return nil, fmt.Errorf("select journal entries of snapshot %d: %w", snapshotID, err)
	}
	return records, nil
}

// GetOtherPostedSnapshotID returns another snapshot of the employee's month
// that has journal entries posted (or being posted), or nil.
func (d *DB) GetOtherPostedSnapshotID(ctx context.Context, q sqlx.QueryerContext, employeeID int64, month timex.Month, snapshotID int64) (*int64, error) {
	query := d.db.Rebind(`
		SELECT snapshot_id
		FROM salary_snapshot_journal_entries
		WHERE employee_id = ? AND month = ? AND snapshot_id <> ? AND status IN ('posted', 'pending')
		ORDER BY snapshot_id
		LIMIT 1
	`)

	var id int64
	if err := sqlx.GetContext(ctx, q, &id, query, employeeID, month, snapshotID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select other posted snapshot: %w", err)
	}
	return &id, nil
}

// DeleteFailedJournalEntries deletes the snapshot's failed journal entries,
// which exist nowhere else.
func (d *DB) DeleteFailedJournalEntries(ctx context.Context, tx *sqlx.Tx, snapshotID int64) error {
	query := tx.Rebind(`DELETE FROM salary_snapshot_journal_entries WHERE snapshot_id = ? AND status = 'failed'`)
	if _, err := tx.ExecContext(ctx, query, snapshotID); err != nil {
		return fmt.Errorf("delete failed journal entries of snapshot %d: %w", snapshotID, err)
	}
	return nil
}

// InsertJournalEntry inserts the record and sets its ID and timestamps. It
// wraps errUniqueViolation when a unique index rejects it.
func (d *DB) InsertJournalEntry(ctx context.Context, tx *sqlx.Tx, record *JournalEntryRecord) error {
	query := tx.Rebind(`
		INSERT INTO salary_snapshot_journal_entries (
			snapshot_id, employee_id, month, kind, component_index, description,
			credit_account_code, debit_account_code, amount, journal_date, status,
			attempts, idempotency_key, posted_by, posted_by_name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at
	`)

	row := tx.QueryRowxContext(ctx, query,
		record.SnapshotID, record.EmployeeID, record.Month, record.Kind, record.ComponentIndex, record.Description,
		record.CreditAccountCode, record.DebitAccountCode, record.Amount, record.journalDateString(), record.Status,
		record.Attempts, record.IdempotencyKey, record.PostedBy, record.PostedByName,
	)
	if err := row.Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("insert journal entry %s: %w: %s", record.IdempotencyKey, errUniqueViolation, pgErr.ConstraintName)
		}
		return fmt.Errorf("insert journal entry %s: %w", record.IdempotencyKey, err)
	}
	return nil
}

// ClaimJournalEntry marks the record pending and counts an attempt, if no one
// claimed it since it was read, and reports whether it did. It wraps
// errUniqueViolation when that would post a second salary for the month.
func (d *DB) ClaimJournalEntry(ctx context.Context, tx *sqlx.Tx, record *JournalEntryRecord, now time.Time) (bool, error) {
	query := tx.Rebind(`
		UPDATE salary_snapshot_journal_entries
		SET status = 'pending', attempts = attempts + 1, error = NULL, updated_at = ?
		WHERE id = ? AND attempts = ?
	`)

	res, err := tx.ExecContext(ctx, query, now, record.ID, record.Attempts)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, fmt.Errorf("claim journal entry %d: %w: %s", record.ID, errUniqueViolation, pgErr.ConstraintName)
		}
		return false, fmt.Errorf("claim journal entry %d: %w", record.ID, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim journal entry %d: %w", record.ID, err)
	}
	if n == 1 {
		record.Status = JournalEntryPending
		record.Attempts++
		record.Error = nil
		record.UpdatedAt = now
	}
	return n == 1, nil
}

// RecordJournalEntryPayload stores the request about to be sent for the
// record's current attempt.
func (d *DB) RecordJournalEntryPayload(ctx context.Context, record JournalEntryRecord, payload []byte, now time.Time) error {
	query := d.db.Rebind(`
		UPDATE salary_snapshot_journal_entries
		SET payload = ?, updated_at = ?
		WHERE id = ? AND attempts = ?
	`)

	if _, err := d.db.ExecContext(ctx, query, payload, now, record.ID, record.Attempts); err != nil {
		return fmt.Errorf("record payload of journal entry %d: %w", record.ID, err)
	}
	return nil
}

// FinishJournalEntry records the outcome of the record's current attempt
// (status, voucher number, Vmedis ID, error and posted time), unless someone
// claimed it since, and reports whether it did.
func (d *DB) FinishJournalEntry(ctx context.Context, record JournalEntryRecord, now time.Time) (bool, error) {
	query := d.db.Rebind(`
		UPDATE salary_snapshot_journal_entries
		SET status = ?, vmedis_voucher_no = ?, vmedis_id = ?, error = ?, posted_at = ?, updated_at = ?
		WHERE id = ? AND attempts = ?
	`)

	res, err := d.db.ExecContext(ctx, query,
		record.Status, record.VoucherNo, record.VmedisID, record.Error, record.PostedAt, now,
		record.ID, record.Attempts,
	)
	if err != nil {
		return false, fmt.Errorf("finish journal entry %d: %w", record.ID, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("finish journal entry %d: %w", record.ID, err)
	}
	return n == 1, nil
}

// HasJournalEntriesInVmedis reports whether the snapshot has journal entries
// posted or being posted.
func (d *DB) HasJournalEntriesInVmedis(ctx context.Context, q sqlx.QueryerContext, snapshotID int64) (bool, error) {
	query := d.db.Rebind(`
		SELECT EXISTS (
			SELECT 1 FROM salary_snapshot_journal_entries
			WHERE snapshot_id = ? AND status IN ('posted', 'pending')
		)
	`)

	var exists bool
	if err := sqlx.GetContext(ctx, q, &exists, query, snapshotID); err != nil {
		return false, fmt.Errorf("check journal entries of snapshot %d: %w", snapshotID, err)
	}
	return exists, nil
}
