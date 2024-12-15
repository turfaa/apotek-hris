-- Write your UP migration SQL here

-- Add work_multiplier column with default value 1
ALTER TABLE work_log_units
    ADD COLUMN work_multiplier DECIMAL NOT NULL DEFAULT 1;

-- Update existing records to use their work type's multiplier
UPDATE work_log_units wlu
SET work_multiplier = wt.multiplier
FROM work_types wt
WHERE wlu.work_type_id = wt.id;
