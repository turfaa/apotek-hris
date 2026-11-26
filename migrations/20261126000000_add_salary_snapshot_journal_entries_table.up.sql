-- Journal entries (jurnal keuangan) posted to Vmedis from salary snapshots:
-- one 'salary' entry per snapshot, and one 'debt' entry per utang component
-- repaid through it.
CREATE TABLE IF NOT EXISTS salary_snapshot_journal_entries (
    id BIGSERIAL PRIMARY KEY,
    snapshot_id BIGINT NOT NULL REFERENCES salary_snapshots(id),
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    month VARCHAR(7) NOT NULL, -- Format: YYYY-MM
    kind TEXT NOT NULL CHECK (kind IN ('salary', 'debt')),
    -- The index of the utang component in the snapshot's components, for
    -- 'debt' entries.
    component_index INT NULL CHECK ((kind = 'debt') = (component_index IS NOT NULL)),
    -- The keterangan sent to Vmedis. Fixed when the entry is first recorded,
    -- so a retry sends the same text.
    description TEXT NOT NULL,
    credit_account_code TEXT NOT NULL,
    debit_account_code TEXT NOT NULL,
    amount NUMERIC NOT NULL CHECK (amount > 0),
    journal_date DATE NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'posted', 'failed')),
    -- Counts the times the entry was claimed to be sent; a send only records
    -- its outcome if no one claimed the entry since.
    attempts INT NOT NULL DEFAULT 0,
    -- Sent as the Idempotency-Key to vmedis-proxy.
    idempotency_key TEXT NOT NULL UNIQUE,
    -- The request last sent to vmedis-proxy, for audit.
    payload JSONB NULL,
    vmedis_voucher_no TEXT NULL,
    vmedis_id BIGINT NULL,
    error TEXT NULL,
    -- The email (X-Email) and name of the admin who posted it.
    posted_by TEXT NOT NULL,
    posted_by_name TEXT NOT NULL DEFAULT '',
    posted_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One entry per snapshot and component (the salary entry has no component).
CREATE UNIQUE INDEX idx_salary_snapshot_journal_entries_line
    ON salary_snapshot_journal_entries (snapshot_id, kind, COALESCE(component_index, -1));

-- A month's salary of an employee is posted from one snapshot only.
CREATE UNIQUE INDEX idx_salary_snapshot_journal_entries_salary_once
    ON salary_snapshot_journal_entries (employee_id, month)
    WHERE kind = 'salary' AND status IN ('pending', 'posted');

CREATE INDEX idx_salary_snapshot_journal_entries_employee_month
    ON salary_snapshot_journal_entries (employee_id, month);
