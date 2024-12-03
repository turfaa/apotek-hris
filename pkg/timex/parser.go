package timex

import (
	"fmt"
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
