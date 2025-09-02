package salary

import (
	"github.com/go-json-experiment/json"

	decimal "github.com/shopspring/decimal"
)

type Salary struct {
	Components []Component `json:"components"`
}

func (s Salary) Total() decimal.Decimal {
	total := decimal.Zero
	for _, component := range s.Components {
		total = total.Add(component.Total())
	}

	return total.RoundUp(0)
}

func (s Salary) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Components []Component     `json:"components"`
		Total      decimal.Decimal `json:"total"`
	}{
		Components: s.Components,
		Total:      s.Total(),
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
