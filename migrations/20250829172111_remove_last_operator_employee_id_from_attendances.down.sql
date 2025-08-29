-- Add back last_operator_employee_id column to attendances table
ALTER TABLE attendances ADD COLUMN last_operator_employee_id BIGINT REFERENCES employees(id);
