-- Create salary_snapshots table
CREATE TABLE IF NOT EXISTS salary_snapshots (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    month VARCHAR(7) NOT NULL, -- Format: YYYY-MM
    salary JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

-- Create indexes for better query performance
CREATE INDEX idx_salary_snapshots_employee_month ON salary_snapshots(employee_id, month) WHERE deleted_at IS NULL;
CREATE INDEX idx_salary_snapshots_month ON salary_snapshots(month) WHERE deleted_at IS NULL;
CREATE INDEX idx_salary_snapshots_created_at ON salary_snapshots(created_at);
