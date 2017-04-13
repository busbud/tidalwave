package sqlquery

import "github.com/dustinblackman/moment"

const queryDateFormat = "YYYY-MM-DDTHH:mm:ss"

// DateParam stores date query information.
type DateParam struct {
	Date     string
	DateTime *moment.Moment
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

	dateParam.DateTime = moment.New().Moment(queryDateFormat, date)
	return dateParam
}
