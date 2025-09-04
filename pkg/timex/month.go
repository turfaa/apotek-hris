package timex

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/turfaa/go-date"
)

type Month struct {
	Year  int
	Month int
}

func NewMonth(year int, month int) Month {
	return Month{Year: year, Month: month}
}

func NewMonthFromString(monthStr string) (Month, error) {
	year, month, err := ParseMonth(monthStr)
	if err != nil {
		return Month{}, fmt.Errorf("parse month: %w", err)
	}

	return NewMonth(year, month), nil
}

func (m Month) String() string {
	return fmt.Sprintf("%04d-%02d", m.Year, m.Month)
}

func (m Month) DateRange() (from date.Date, to date.Date, err error) {
	return MonthDateRange(m.Year, m.Month)
}

func (m Month) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *Month) UnmarshalJSON(data []byte) error {
	s := string(data)
	s = strings.TrimSpace(s)

	parsed, err := NewMonthFromString(s)
	if err != nil {
		return fmt.Errorf("parse month: %w", err)
	}

	*m = parsed
	return nil
}

// Value implements the driver.Valuer interface.
// It returns the Month as a string in "YYYY-MM" format.
func (m Month) Value() (driver.Value, error) {
	return m.String(), nil
}

// Scan implements the sql.Scanner interface.
// It expects a string in "YYYY-MM" format.
func (m *Month) Scan(value any) error {
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("timex.Month: cannot scan type %T", value)
	}

	return m.UnmarshalJSON(bytes)
}
