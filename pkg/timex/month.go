package timex

import (
	"fmt"

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
	return fmt.Sprintf("%d-%d", m.Year, m.Month)
}

func (m Month) DateRange() (from date.Date, to date.Date, err error) {
	return MonthDateRange(m.Year, m.Month)
}
