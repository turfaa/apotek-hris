-- Create salary_static_components table
CREATE TABLE IF NOT EXISTS salary_static_components (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    description VARCHAR(255) NOT NULL,
    amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    multiplier NUMERIC(10,4) NOT NULL DEFAULT 1.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_salary_static_components_employee_id ON salary_static_components(employee_id);
