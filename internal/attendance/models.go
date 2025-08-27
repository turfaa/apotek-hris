package attendance

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/turfaa/apotek-hris/pkg/slicex"
	"github.com/turfaa/go-date"
)

type Attendance struct {
	ID int64 `db:"id" json:"id"`

	EmployeeID    int64           `db:"employee_id" json:"employeeID"`
	Date          date.Date       `db:"date" json:"date"`
	Type          Type            `db:"type" json:"type"`
	OvertimeHours decimal.Decimal `db:"overtime_hours" json:"overtimeHours"`

	CreatedAt              time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt              time.Time `db:"updated_at" json:"updatedAt"`
	LastOperatorEmployeeID int64     `db:"last_operator_employee_id" json:"lastOperatorEmployeeID"`
}

type Type struct {
	ID int64 `db:"id" json:"id"`

	Name        string      `db:"name" json:"name"`
	PayableType PayableType `db:"payable_type" json:"payableType"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type UpsertAttendanceRequest struct {
	EmployeeID    int64           `json:"-" validate:"required,gt=0"`
	Date          date.Date       `json:"-" validate:"required"`
	TypeID        int64           `json:"typeID" validate:"required"`
	OvertimeHours decimal.Decimal `json:"overtimeHours" validate:"dgte=0"`

	OperatorEmployeeID int64 `json:"-" validate:"required,gt=0"`
}

type CreateAttendanceTypeRequest struct {
	Name        string      `json:"name" validate:"required"`
	PayableType PayableType `json:"payableType" validate:"required"`
}

type ListAtDate struct {
	Date        date.Date    `json:"date"`
	Attendances []Attendance `json:"attendances"`
}

func CreateListAtDate(from date.Date, to date.Date, attendances []Attendance) []ListAtDate {
	attendancesByDate := slicex.GroupBy(attendances, func(a Attendance) date.Date {
		return a.Date
	})

	dayDelta := 1
	if from.After(to) {
		dayDelta = -1
	}

	var res []ListAtDate
	for day := from; day.Before(to); day = day.AddDate(0, 0, dayDelta) {
		res = append(res, ListAtDate{
			Date:        day,
			Attendances: attendancesByDate[day],
		})
	}

	return res
}

type EmployeeSummary struct {
	EmployeeID    int64           `json:"employeeID"`
	WorkingDays   int             `json:"workingDays"`
	DaysByBenefit map[string]int  `json:"daysByBenefit"`
	OvertimeHours decimal.Decimal `json:"overtimeHours"`
}

func CreateEmployeeSummaries(attendances []Attendance) []EmployeeSummary {
	attendancesByEmployee := slicex.GroupBy(attendances, func(a Attendance) int64 {
		return a.EmployeeID
	})

	res := make([]EmployeeSummary, 0, len(attendancesByEmployee))
	for _, attendances := range attendancesByEmployee {
		res = append(res, CreateEmployeeSummary(attendances))
	}

	return res
}

// CreateEmployeeSummary assumes that the attendances are only for one employee.
func CreateEmployeeSummary(attendances []Attendance) EmployeeSummary {
	if len(attendances) == 0 {
		return EmployeeSummary{}
	}

	summary := EmployeeSummary{
		EmployeeID:    attendances[0].EmployeeID,
		DaysByBenefit: make(map[string]int),
	}

	for _, attendance := range attendances {
		summary.OvertimeHours = summary.OvertimeHours.Add(attendance.OvertimeHours)

		switch attendance.Type.PayableType {
		case PayableTypeWorking:
			summary.WorkingDays++
		case PayableTypeBenefit:
			summary.DaysByBenefit[attendance.Type.Name]++
		}
	}

	return summary
}
