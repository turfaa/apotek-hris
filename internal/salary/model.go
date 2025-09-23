package salary

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/turfaa/apotek-hris/pkg/timex"

	decimal "github.com/shopspring/decimal"
)

type Salary struct {
	Components []Component `json:"components"`
	ExtraInfos []ExtraInfo `json:"extraInfos"`
}

func (s Salary) Total() decimal.Decimal {
	total := decimal.Zero
	for _, component := range s.Components {
		total = total.Add(component.Total())
	}

	return total.RoundUp(0)
}

func (s Salary) TotalWithoutDebt() decimal.Decimal {
	total := decimal.Zero
	for _, component := range s.Components {
		if !strings.Contains(strings.ToLower(component.Description), "utang") {
			total = total.Add(component.Total())
		}
	}

	return total.RoundUp(0)
}

func (s Salary) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Components       []Component     `json:"components"`
		Total            decimal.Decimal `json:"total"`
		TotalWithoutDebt decimal.Decimal `json:"totalWithoutDebt"`
		ExtraInfos       []ExtraInfo     `json:"extraInfos"`
	}{
		Components:       s.Components,
		Total:            s.Total(),
		TotalWithoutDebt: s.TotalWithoutDebt(),
		ExtraInfos:       s.ExtraInfos,
	})
}

type Component struct {
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	Multiplier  decimal.Decimal `json:"multiplier"`
}

func (c Component) Total() decimal.Decimal {
	return c.Amount.Mul(c.Multiplier).RoundUp(0)
}

func (c Component) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Description string          `json:"description"`
		Amount      decimal.Decimal `json:"amount"`
		Multiplier  decimal.Decimal `json:"multiplier"`
		Total       decimal.Decimal `json:"total"`
	}{
		Description: c.Description,
		Amount:      c.Amount,
		Multiplier:  c.Multiplier,
		Total:       c.Total(),
	})
}

type StaticComponent struct {
	ID          int64           `json:"id" db:"id"`
	EmployeeID  int64           `json:"employeeID" db:"employee_id"`
	Description string          `json:"description" db:"description"`
	Amount      decimal.Decimal `json:"amount" db:"amount"`
	Multiplier  decimal.Decimal `json:"multiplier" db:"multiplier"`
	CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
}

func (c StaticComponent) Total() decimal.Decimal {
	return c.Amount.Mul(c.Multiplier).RoundUp(0)
}

func (c StaticComponent) ToComponent() Component {
	return Component{
		Description: c.Description,
		Amount:      c.Amount,
		Multiplier:  c.Multiplier,
	}
}

func (c StaticComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          int64           `json:"id" db:"id"`
		EmployeeID  int64           `json:"employeeID" db:"employee_id"`
		Description string          `json:"description" db:"description"`
		Amount      decimal.Decimal `json:"amount" db:"amount"`
		Multiplier  decimal.Decimal `json:"multiplier" db:"multiplier"`
		CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
		Total       decimal.Decimal `json:"total"`
	}{
		ID:          c.ID,
		EmployeeID:  c.EmployeeID,
		Description: c.Description,
		Amount:      c.Amount,
		Multiplier:  c.Multiplier,
		CreatedAt:   c.CreatedAt,
		Total:       c.Total(),
	})
}

type AdditionalComponent struct {
	ID          int64           `json:"id" db:"id"`
	EmployeeID  int64           `json:"employeeID" db:"employee_id"`
	Month       timex.Month     `json:"month" db:"month"`
	Description string          `json:"description" db:"description"`
	Amount      decimal.Decimal `json:"amount" db:"amount"`
	Multiplier  decimal.Decimal `json:"multiplier" db:"multiplier"`
	CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
}

func (c AdditionalComponent) Total() decimal.Decimal {
	return c.Amount.Mul(c.Multiplier).RoundUp(0)
}

func (c AdditionalComponent) ToComponent() Component {
	return Component{
		Description: c.Description,
		Amount:      c.Amount,
		Multiplier:  c.Multiplier,
	}
}

func (c AdditionalComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          int64           `json:"id" db:"id"`
		EmployeeID  int64           `json:"employeeID" db:"employee_id"`
		Month       timex.Month     `json:"month" db:"month"`
		Description string          `json:"description" db:"description"`
		Amount      decimal.Decimal `json:"amount" db:"amount"`
		Multiplier  decimal.Decimal `json:"multiplier" db:"multiplier"`
		CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
		Total       decimal.Decimal `json:"total"`
	}{
		ID:          c.ID,
		EmployeeID:  c.EmployeeID,
		Month:       c.Month,
		Description: c.Description,
		Amount:      c.Amount,
		Multiplier:  c.Multiplier,
		CreatedAt:   c.CreatedAt,
		Total:       c.Total(),
	})
}

type ExtraInfo struct {
	ID          int64       `json:"id" db:"id"`
	EmployeeID  int64       `json:"employeeID" db:"employee_id"`
	Month       timex.Month `json:"month" db:"month"`
	Title       string      `json:"title" db:"title"`
	Description string      `json:"description" db:"description"`
	CreatedAt   time.Time   `json:"createdAt" db:"created_at"`
}

type CreateExtraInfoRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
}

type Snapshot struct {
	ID         int64       `json:"id"`
	EmployeeID int64       `json:"employeeID"`
	Month      timex.Month `json:"month"`
	Salary     Salary      `json:"salary"`
	CreatedAt  time.Time   `json:"createdAt"`
}

type SnapshotDB struct {
	ID         int64       `db:"id"`
	EmployeeID int64       `db:"employee_id"`
	Month      timex.Month `db:"month"`
	Salary     []byte      `db:"salary"`
	CreatedAt  time.Time   `db:"created_at"`
	DeletedAt  *time.Time  `db:"deleted_at"`
}

func (s SnapshotDB) ToSnapshot() (Snapshot, error) {
	var salary Salary
	if err := json.Unmarshal(s.Salary, &salary); err != nil {
		return Snapshot{}, fmt.Errorf("unmarshal salary snapshot: %w", err)
	}

	return Snapshot{
		ID:         s.ID,
		EmployeeID: s.EmployeeID,
		Month:      s.Month,
		Salary:     salary,
		CreatedAt:  s.CreatedAt,
	}, nil
}

type GetSnapshotsRequest struct {
	EmployeeID *int64       `json:"employeeID"`
	Month      *timex.Month `json:"month"`
}

type CreateSnapshotRequest struct {
	EmployeeID int64       `json:"employeeID" validate:"required"`
	Month      timex.Month `json:"month" validate:"required"`
}
