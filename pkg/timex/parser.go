package timex

import (
	"fmt"
	"strconv"
	"time"
)

func ParseTimeRange(dateQuery, fromQuery string, toQuery string) (from time.Time, to time.Time, err error) {
	if fromQuery != "" {
		from, _, err = Day(fromQuery)
		if err != nil {
			err = fmt.Errorf("parse time range from `from` query [%s]: %w", fromQuery, err)
			return
		}

		if toQuery != "" {
			_, to, err = Day(toQuery)
			if err != nil {
				err = fmt.Errorf("parse time range from `to` query [%s]: %w", toQuery, err)
				return
			}
		} else {
			to = EndOfToday()
		}
	} else if dateQuery == "" {
		from, to = Today()
	} else {
		from, to, err = Day(dateQuery)
		if err != nil {
			err = fmt.Errorf("parse time range from `date` query [%s]: %w", dateQuery, err)
			return
		}
	}

	return
}

// ParseMonth parses month string in YYYY-MM format.
func ParseMonth(monthStr string) (year int, month int, err error) {
	if len(monthStr) != 7 {
		return 0, 0, fmt.Errorf("invalid month format (%s), expected YYYY-MM", monthStr)
	}

	year, err = strconv.Atoi(monthStr[:4])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year (%s): %w", monthStr[:4], err)
	}

	month, err = strconv.Atoi(monthStr[5:])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month (%s): %w", monthStr[5:], err)
	}

	return year, month, nil
}
