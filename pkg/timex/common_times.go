package timex

import (
	"fmt"
	"time"
)

func Today() (from time.Time, until time.Time) {
	return BeginningOfToday(), EndOfToday()
}

func Day(date string) (from time.Time, until time.Time, err error) {
	from, err = BeginningOfDate(date)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("beginning of date: %w", err)
	}

	until, err = EndOfDate(date)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("end of date: %w", err)
	}

	return from, until, nil
}

func BeginningOfToday() time.Time {
	year, month, day := time.Now().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func EndOfToday() time.Time {
	year, month, day := time.Now().Date()
	return time.Date(year, month, day, 23, 59, 59, 999999999, time.Local)
}

func BeginningOfDate(date string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %w", err)
	}

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
}

func EndOfDate(date string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %w", err)
	}

	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.Local), nil
}

func BeginningOfLastMonth() time.Time {
	year, month, _ := time.Now().AddDate(0, -1, 0).Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
}

func EndOfLastMonth() time.Time {
	year, month, _ := time.Now().AddDate(0, -1, 0).Date()
	return time.Date(year, month+1, 0, 23, 59, 59, 999999999, time.Local)
}
