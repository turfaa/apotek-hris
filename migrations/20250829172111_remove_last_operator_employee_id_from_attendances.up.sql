-- Remove last_operator_employee_id column from attendances table
ALTER TABLE attendances DROP COLUMN IF EXISTS last_operator_employee_id;
