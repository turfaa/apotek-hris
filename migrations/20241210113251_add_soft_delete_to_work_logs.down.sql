-- Write your DOWN migration SQL here

-- Remove indexes
DROP INDEX IF EXISTS idx_work_logs_deleted_at;
DROP INDEX IF EXISTS idx_work_log_units_deleted_at;
DROP INDEX IF EXISTS idx_work_logs_employee_created_at;

-- Restore original index
CREATE INDEX idx_work_logs_created_at ON work_logs(created_at);

-- Remove soft delete columns
ALTER TABLE work_log_units
    DROP COLUMN deleted_at,
    DROP COLUMN deleted_by;

ALTER TABLE work_logs
    DROP COLUMN deleted_at,
    DROP COLUMN deleted_by;
