package attendance

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/turfaa/apotek-hris/pkg/slicex"
	"github.com/turfaa/go-date"
)

// ErrQuotaExhausted is returned when an employee has no remaining quota for an attendance type.
var ErrQuotaExhausted = errors.New("attendance quota exhausted")

// ErrAlreadyHasQuota is returned when trying to enable quota on an attendance type that already has quota enabled.
var ErrAlreadyHasQuota = errors.New("attendance type already has quota enabled")

type Attendance struct {
	ID int64 `db:"id" json:"id"`

	EmployeeID    int64           `db:"employee_id" json:"employeeID"`
	Date          date.Date       `db:"date" json:"date"`
	Type          Type            `db:"type" json:"type"`
	OvertimeHours decimal.Decimal `db:"overtime_hours" json:"overtimeHours"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type Type struct {
	ID int64 `db:"id" json:"id"`

	Name        string      `db:"name" json:"name"`
	PayableType PayableType `db:"payable_type" json:"payableType"`
	HasQuota    bool        `db:"has_quota" json:"hasQuota"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type UpsertAttendanceRequest struct {
	EmployeeID    int64           `json:"-" validate:"required,gt=0"`
	Date          date.Date       `json:"-" validate:"required"`
	TypeID        int64           `json:"typeID" validate:"required"`
	OvertimeHours decimal.Decimal `json:"overtimeHours" validate:"dgte=0"`
}

type CreateAttendanceTypeRequest struct {
	Name        string      `json:"name" validate:"required"`
	PayableType PayableType `json:"payableType" validate:"required"`
	HasQuota    bool        `json:"hasQuota"`
}

type EmployeeAttendanceQuota struct {
	ID             int64     `db:"id" json:"id"`
	EmployeeID     int64     `db:"employee_id" json:"employeeID"`
	AttendanceType Type      `db:"attendance_type" json:"attendanceType"`
	RemainingQuota int       `db:"remaining_quota" json:"remainingQuota"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

// QuotaAuditReason describes why a quota changed.
type QuotaAuditReason string

const (
	// QuotaAuditReasonManualSet is used when an admin explicitly sets the quota value.
	QuotaAuditReasonManualSet QuotaAuditReason = "manual_set"
	// QuotaAuditReasonAttendanceDeduction is used when quota is decremented by marking attendance.
	QuotaAuditReasonAttendanceDeduction QuotaAuditReason = "attendance_deduction"
	// QuotaAuditReasonAttendanceRestoration is used when quota is restored by changing attendance type.
	QuotaAuditReasonAttendanceRestoration QuotaAuditReason = "attendance_restoration"
)

// QuotaAuditLog records a change to an employee's attendance quota.
type QuotaAuditLog struct {
	ID             int64            `db:"id" json:"id"`
	EmployeeID     int64            `db:"employee_id" json:"employeeID"`
	AttendanceType Type             `db:"attendance_type" json:"attendanceType"`
	PreviousQuota  int              `db:"previous_quota" json:"previousQuota"`
	NewQuota       int              `db:"new_quota" json:"newQuota"`
	Reason         QuotaAuditReason `db:"reason" json:"reason"`
	CreatedAt      time.Time        `db:"created_at" json:"createdAt"`
}

type SetEmployeeAttendanceQuotaRequest struct {
	EmployeeID       int64 `json:"-" validate:"required,gt=0"`
	AttendanceTypeID int64 `json:"-" validate:"required,gt=0"`
	RemainingQuota   int   `json:"remainingQuota" validate:"gte=0"`
}

// AttendanceTypeQuotaPage groups employee quotas by their attendance type.
type AttendanceTypeQuotaPage struct {
	AttendanceType Type                      `json:"attendanceType"`
	Quotas         []EmployeeAttendanceQuota `json:"quotas"`
}

// GroupQuotasByAttendanceType groups a flat list of employee quotas into pages keyed by attendance type.
// For each quota-enabled type, every employee in allEmployeeIDs is included. Employees without a quota
// entry for a given type get a zero-value entry with remaining_quota = 0.
// The order of pages preserves the first-seen order from the input slice, followed by any added quota-enabled types.
func GroupQuotasByAttendanceType(quotas []EmployeeAttendanceQuota, quotaEnabledTypes []Type, allEmployeeIDs []int64) []AttendanceTypeQuotaPage {
	pageMap := make(map[int64]*AttendanceTypeQuotaPage)
	// Track which employees already have a quota entry per type.
	existingEmployees := make(map[int64]map[int64]bool) // typeID -> employeeID -> true
	var pageOrder []int64

	for _, q := range quotas {
		typeID := q.AttendanceType.ID
		if _, exists := pageMap[typeID]; !exists {
			pageMap[typeID] = &AttendanceTypeQuotaPage{
				AttendanceType: q.AttendanceType,
			}
			existingEmployees[typeID] = make(map[int64]bool)
			pageOrder = append(pageOrder, typeID)
		}
		pageMap[typeID].Quotas = append(pageMap[typeID].Quotas, q)
		existingEmployees[typeID][q.EmployeeID] = true
	}

	// Add quota-enabled types that have no entries yet.
	for _, t := range quotaEnabledTypes {
		if _, exists := pageMap[t.ID]; !exists {
			pageMap[t.ID] = &AttendanceTypeQuotaPage{
				AttendanceType: t,
			}
			existingEmployees[t.ID] = make(map[int64]bool)
			pageOrder = append(pageOrder, t.ID)
		}
	}

	// Fill in missing employees with quota 0 for each type.
	for _, typeID := range pageOrder {
		page := pageMap[typeID]
		for _, empID := range allEmployeeIDs {
			if !existingEmployees[typeID][empID] {
				page.Quotas = append(page.Quotas, EmployeeAttendanceQuota{
					ID:             -(typeID*1_000_000_000 + empID), // fake id to avoid conflicts with real ids
					EmployeeID:     empID,
					AttendanceType: page.AttendanceType,
					RemainingQuota: 0,
				})
			}
		}
	}

	pages := make([]AttendanceTypeQuotaPage, 0, len(pageOrder))
	for _, typeID := range pageOrder {
		pages = append(pages, *pageMap[typeID])
	}

	return pages
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
	for day := from; !day.After(to); day = day.AddDate(0, 0, dayDelta) {
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
