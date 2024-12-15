-- Write your DOWN migration SQL here

-- Remove work_multiplier column
ALTER TABLE work_log_units
    DROP COLUMN work_multiplier;
