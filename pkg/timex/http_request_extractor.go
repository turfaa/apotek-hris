package timex

import (
	"fmt"
	"net/http"
	"time"

	"github.com/turfaa/go-date"
)

func GetOneDayFromQuery(request *http.Request) (from time.Time, to time.Time, err error) {
	return ParseTimeRange(request.URL.Query().Get("date"), "", "")
}

func GetTimeRangeFromQuery(request *http.Request) (from time.Time, to time.Time, err error) {
	query := request.URL.Query()

	return ParseTimeRange(
		query.Get("date"),
		query.Get("from"),
		query.Get("to"),
	)
}

func GetMonthDateRangeFromQuery(request *http.Request) (from date.Date, to date.Date, err error) {
	monthStr := request.URL.Query().Get("month")
	if monthStr == "" {
		return 0, 0, fmt.Errorf("month is required")
	}

	return MonthDateRangeFromMonthString(monthStr)
}

func GetMonthFromQuery(request *http.Request) (year int, month int, err error) {
	monthStr := request.URL.Query().Get("month")
	if monthStr == "" {
		return 0, 0, fmt.Errorf("month is required")
	}

	return ParseMonth(monthStr)
}
