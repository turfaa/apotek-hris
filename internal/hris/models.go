package hris

import (
	"time"

	"github.com/shopspring/decimal"
)

type Employee struct {
	ID        int64           `db:"id" json:"id"`
	Name      string          `db:"name" json:"name"`
	ShiftFee  decimal.Decimal `db:"shift_fee" json:"shiftFee"`
	CreatedAt time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time       `db:"updated_at" json:"updatedAt"`
}

type CreateEmployeeRequest struct {
	Name     string          `json:"name" validate:"required"`
	ShiftFee decimal.Decimal `json:"shiftFee" validate:"dgt=0"`
}

type WorkType struct {
	ID          int64           `db:"id" json:"id"`
	Name        string          `db:"name" json:"name"`
	OutcomeUnit string          `db:"outcome_unit" json:"outcomeUnit"`
	Multiplier  decimal.Decimal `db:"multiplier" json:"multiplier"`
	Notes       string          `db:"notes" json:"notes"`
}

type CreateWorkTypeRequest struct {
	Name        string          `json:"name" validate:"required"`
	OutcomeUnit string          `json:"outcomeUnit"`
	Multiplier  decimal.Decimal `json:"multiplier" validate:"dgte=0"`
	Notes       string          `json:"notes"`
}

type WorkLog struct {
	ID          int64         `db:"id" json:"id"`
	Employee    Employee      `db:"employee" json:"employee"`
	PatientName string        `db:"patient_name" json:"patientName"`
	Units       []WorkLogUnit `db:"-" json:"units"`
	CreatedAt   time.Time     `db:"created_at" json:"createdAt"`
	DeletedAt   *time.Time    `db:"deleted_at" json:"deletedAt,omitempty"`
	DeletedBy   *int64        `db:"deleted_by" json:"deletedBy,omitempty"`
}

type WorkLogUnit struct {
	ID          int64      `json:"id" db:"id"`
	WorkType    WorkType   `json:"workType" db:"work_type"`
	WorkOutcome string     `json:"workOutcome" db:"work_outcome"`
	DeletedAt   *time.Time `db:"deleted_at" json:"deletedAt,omitempty"`
	DeletedBy   *int64     `db:"deleted_by" json:"deletedBy,omitempty"`
}

type DBWorkLog struct {
	ID          int64      `db:"id"`
	EmployeeID  int64      `db:"employee_id"`
	PatientName string     `db:"patient_name"`
	CreatedAt   time.Time  `db:"created_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
	DeletedBy   *int64     `db:"deleted_by"`
}

type DBWorkLogUnit struct {
	ID          int64      `db:"id"`
	WorkLogID   int64      `db:"work_log_id"`
	WorkTypeID  int64      `db:"work_type_id"`
	WorkOutcome string     `db:"work_outcome"`
	DeletedAt   *time.Time `db:"deleted_at"`
	DeletedBy   *int64     `db:"deleted_by"`
}

type CreateWorkLogRequest struct {
	EmployeeID  int64                      `json:"employeeID" validate:"required"`
	PatientName string                     `json:"patientName" validate:"required"`
	Units       []CreateWorkLogUnitRequest `json:"units" validate:"min=1,dive"`
}

type CreateWorkLogUnitRequest struct {
	WorkTypeID  int64  `json:"workTypeID" validate:"required"`
	WorkOutcome string `json:"workOutcome" validate:"required"`
}
