-- Write your UP migration SQL here

-- Add soft delete columns
ALTER TABLE work_logs
    ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN deleted_by INTEGER REFERENCES employees(id);

ALTER TABLE work_log_units
    ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN deleted_by INTEGER REFERENCES employees(id);

-- Add indexes for better query performance
CREATE INDEX idx_work_logs_deleted_at ON work_logs(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_log_units_deleted_at ON work_log_units(deleted_at) WHERE deleted_at IS NULL;

-- Update existing indexes to include deleted_at condition
DROP INDEX IF EXISTS idx_work_logs_created_at;
CREATE INDEX idx_work_logs_created_at ON work_logs(created_at) WHERE deleted_at IS NULL;

-- Add composite index for common queries
CREATE INDEX idx_work_logs_employee_created_at ON work_logs(employee_id, created_at) WHERE deleted_at IS NULL;
