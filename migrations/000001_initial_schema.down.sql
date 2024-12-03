DROP INDEX IF EXISTS idx_work_log_units_work_log_id;
DROP TABLE IF EXISTS work_log_units;

DROP INDEX IF EXISTS idx_work_logs_employee_id;
DROP INDEX IF EXISTS idx_work_logs_created_at_employee_id;
DROP TABLE IF EXISTS work_logs;

DROP INDEX IF EXISTS idx_work_types_name;
DROP TABLE IF EXISTS work_types;

DROP INDEX IF EXISTS idx_leave_balance_changes_employee_id;
DROP TABLE IF EXISTS leave_balance_changes;

DROP TABLE IF EXISTS employees;
