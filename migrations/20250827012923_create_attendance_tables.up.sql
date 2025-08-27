-- Create attendance_payable_type enum
CREATE TYPE attendance_payable_type AS ENUM ('working', 'benefit', 'none');

-- Create attendance_types table
CREATE TABLE IF NOT EXISTS attendance_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    payable_type attendance_payable_type NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create unique index on attendance_type name
CREATE UNIQUE INDEX idx_attendance_types_name ON attendance_types(name);

-- Create attendances table
CREATE TABLE IF NOT EXISTS attendances (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    date DATE NOT NULL,
    type_id BIGINT NOT NULL REFERENCES attendance_types(id),
    overtime_hours NUMERIC(5,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_operator_employee_id BIGINT NOT NULL REFERENCES employees(id)
);

-- Create indexes for attendances table
CREATE INDEX idx_attendances_date ON attendances(date);

-- Create unique constraint to prevent duplicate attendance records for the same employee on the same date
CREATE UNIQUE INDEX idx_attendances_employee_date_unique ON attendances(employee_id, date);
