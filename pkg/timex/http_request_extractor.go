package timex

import (
	"net/http"
	"time"
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
