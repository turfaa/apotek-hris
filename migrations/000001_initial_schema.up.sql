CREATE TABLE IF NOT EXISTS employees (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    shift_fee NUMERIC NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
); 

CREATE TABLE IF NOT EXISTS leave_balance_changes (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    change_amount INT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
); 

CREATE INDEX idx_leave_balance_changes_employee_id ON leave_balance_changes(employee_id);

CREATE TABLE work_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    outcome_unit VARCHAR(20) NOT NULL,
    multiplier NUMERIC NOT NULL
); 

CREATE UNIQUE INDEX idx_work_types_name ON work_types(name);

CREATE TABLE work_logs (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    patient_name VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
); 

CREATE INDEX idx_work_logs_created_at_employee_id ON work_logs(created_at, employee_id);
CREATE INDEX idx_work_logs_employee_id ON work_logs(employee_id);

CREATE TABLE work_log_units (
    id BIGSERIAL PRIMARY KEY,
    work_log_id BIGINT NOT NULL REFERENCES work_logs(id),
    work_type_id BIGINT NOT NULL REFERENCES work_types(id),
    work_outcome TEXT NOT NULL
); 

CREATE INDEX idx_work_log_units_work_log_id ON work_log_units(work_log_id);