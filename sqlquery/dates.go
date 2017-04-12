package sqlquery

import "time"

const queryDateFormat = "2006-01-02T15:04:05" // YYYY-MM-DDTHH:mm:ss

// DateParam stores date query information.
type DateParam struct {
	Date     string
	DateTime time.Time
	Operator string
	TimeUsed bool
	Type     string
}

func createDateParam(date, operator string) DateParam {
	dateParam := DateParam{Operator: operator, TimeUsed: true}
	date = stripQuotes(date)
	if len(date) > 0 && len(date) <= 10 {
		dateParam.TimeUsed = false
		if operator == "<=" {
			date = date + "T23:59:59"
		} else {
			date = date + "T00:00:00"
		}
	}
	dateParam.Date = date
	dateParam.Type = "start"
	if operator == "<" || operator == "<=" {
		dateParam.Type = "end"
	}

	dateTime, err := time.Parse(queryDateFormat, date)
	if len(date) > 0 && err == nil {
		dateParam.DateTime = dateTime
	}

	return dateParam
}
