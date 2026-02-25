DROP TABLE IF EXISTS employee_attendance_quotas;

ALTER TABLE attendance_types DROP COLUMN IF EXISTS has_quota;
