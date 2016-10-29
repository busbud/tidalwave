package sqlquery

import (
	"strings"
	"time"
)

// TODO: Only strip at beginning and end
func stripQuotes(value string) string {
	value = strings.Replace(value, "'", "", -1)
	value = strings.Replace(value, "\"", "", -1)
	return value
}

// ProcessInt handles processing an integer in a query
func ProcessInt(q *QueryParam, res int) bool {
	switch q.Operator {
	case "exists":
		return true
	case "=", "==":
		if res == q.ValInt {
			return true
		}
	case "!=":
		if res != q.ValInt {
			return true
		}
	case ">":
		if res > q.ValInt {
			return true
		}
	case ">=":
		if res >= q.ValInt {
			return true
		}
	case "<":
		if res < q.ValInt {
			return true
		}
	case "<=":
		if res <= q.ValInt {
			return true
		}
	}

	return false
}

// ProcessString handles processing a string for a query
func ProcessString(q *QueryParam, res string) bool {
	switch q.Operator { // TODO: Support "like", both for value and entire log string
	case "exists":
		if len(res) > 0 {
			return true
		}
	case "=", "==":
		if res == q.ValString {
			return true
		}
	case "!=":
		if res != q.ValString {
			return true
		}
	}
	return false
}

// ProcessDate handles processing a date in a query
func ProcessDate(d *DateParam, logDate time.Time) bool {
	if len(d.Date) == 0 {
		return true
	}

	dayStart := d.DateTime.Truncate(time.Duration(24 * time.Hour)).Add(time.Duration(-1) * time.Second)
	dayEnd := dayStart.Add(time.Duration(86400) * time.Second)
	switch d.Operator {
	case "exists":
		return true
	case "=", "==":
		if d.DateTime.Equal(logDate) || (logDate.After(dayStart) && logDate.Before(dayEnd)) {
			return true
		}
	case "!=":
		if !d.DateTime.Equal(logDate) || (!logDate.After(dayStart) && !logDate.Before(dayEnd)) {
			return true
		}
	case ">":
		if d.DateTime.Before(logDate) {
			return true
		}
	case ">=":
		if d.DateTime.Before(logDate) || d.DateTime.Equal(logDate) {
			return true
		}
	case "<":
		if d.DateTime.After(logDate) {
			return true
		}
	case "<=":
		if d.DateTime.After(logDate) || d.DateTime.Equal(logDate) {
			return true
		}
	}
	return false
}
