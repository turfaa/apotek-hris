CREATE TABLE attendance_quota_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    attendance_type_id BIGINT NOT NULL REFERENCES attendance_types(id),
    previous_quota INT NOT NULL,
    new_quota INT NOT NULL,
    reason VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attendance_quota_audit_logs_employee_id ON attendance_quota_audit_logs(employee_id);
CREATE INDEX idx_attendance_quota_audit_logs_attendance_type_id ON attendance_quota_audit_logs(attendance_type_id);
