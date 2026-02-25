-- Add has_quota flag to attendance_types
ALTER TABLE attendance_types ADD COLUMN has_quota BOOLEAN NOT NULL DEFAULT FALSE;

-- Per-employee quota allocations for quota-enabled types
CREATE TABLE employee_attendance_quotas (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    attendance_type_id BIGINT NOT NULL REFERENCES attendance_types(id),
    remaining_quota INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (employee_id, attendance_type_id)
);

CREATE INDEX idx_employee_attendance_quotas_employee_id
    ON employee_attendance_quotas(employee_id);
