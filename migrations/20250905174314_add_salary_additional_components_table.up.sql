-- Create salary_additional_components table
CREATE TABLE IF NOT EXISTS salary_additional_components (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    month VARCHAR(7) NOT NULL, -- Format: YYYY-MM
    description VARCHAR(255) NOT NULL,
    amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    multiplier NUMERIC(10,4) NOT NULL DEFAULT 1.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_salary_additional_components_employee_month ON salary_additional_components(employee_id, month);
CREATE INDEX idx_salary_additional_components_month ON salary_additional_components(month);
