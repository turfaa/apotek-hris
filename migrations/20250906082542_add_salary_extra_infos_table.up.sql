-- Create salary_extra_infos table
CREATE TABLE IF NOT EXISTS salary_extra_infos (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    month VARCHAR(7) NOT NULL, -- Format: YYYY-MM
    title VARCHAR(255) NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_salary_extra_infos_employee_month ON salary_extra_infos(employee_id, month);
CREATE INDEX idx_salary_extra_infos_month ON salary_extra_infos(month);
