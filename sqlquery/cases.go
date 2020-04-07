// Package sqlquery handles parsing SQL and converting to a dialect for Tidalwave.
package sqlquery

import (
	"github.com/dustinblackman/moment"
)

func stripQuotes(s string) string {
	if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
		s = s[1:]
	}
	if len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
		s = s[:len(s)-1]
	}

	return s
}

// ProcessInt handles processing an integer in a query
func ProcessInt(q *QueryParam, res int) bool {
	switch q.Operator {
	case "exists":
		return true
	case "in":
		for _, val := range q.ValIntArray {
			if val == res {
				return true
			}
		}
		return false
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
	case "in":
		for _, val := range q.ValStringArray {
			if val == res {
				return true
			}
		}
		return false
	case "like", "ilike", "~~", "~~*":
		return q.Regex.MatchString(res)
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

func withinDay(logDate, dayStart, dayEnd moment.Moment) bool {
	return logDate.IsSame(dayStart, "YYYY-MM-DD") && logDate.IsSame(dayEnd, "YYYY-MM-DD")
}

// ProcessDate handles processing a date in a query
func ProcessDate(d *DateParam, logDate moment.Moment, dateOnly bool) bool {
	if d.Date == "" {
		return true
	}

	dayStart := *d.DateTime.Clone().StartOfDay().SubSeconds(1)
	dayEnd := *d.DateTime.Clone().EndOfDay()

	switch d.Operator {
	case "exists":
		return true
	case "=", "==":
		if d.DateTime.IsSame(logDate, queryDateFormat) || (!d.TimeUsed && withinDay(logDate, dayStart, dayEnd)) {
			return true
		}
	case "!=":
		if !d.DateTime.IsSame(logDate, queryDateFormat) || (!d.TimeUsed && !withinDay(logDate, dayStart, dayEnd)) {
			return true
		}
	case ">":
		if dateOnly {
			startOfDay := d.DateTime.Clone().StartOfDay()
			if !d.TimeUsed && startOfDay.IsBefore(logDate) {
				return true
			}
			if d.TimeUsed && (startOfDay.IsBefore(logDate) || startOfDay.IsSame(logDate, "YYYY-MM-DD")) {
				return true
			}
		}
		if d.DateTime.IsBefore(logDate) {
			return true
		}
	case ">=":
		if dateOnly {
			startOfDay := d.DateTime.Clone().StartOfDay()
			if startOfDay.IsBefore(logDate) || startOfDay.IsSame(logDate, "YYYY-MM-DD") {
				return true
			}
		}
		if d.DateTime.IsBefore(logDate) || d.DateTime.IsSame(logDate, queryDateFormat) {
			return true
		}
	case "<":
		if dateOnly {
			startOfDay := d.DateTime.Clone().StartOfDay()
			if !d.TimeUsed && startOfDay.IsAfter(logDate) {
				return true
			}
			if d.TimeUsed && (startOfDay.IsAfter(logDate) || startOfDay.IsSame(logDate, "YYYY-MM-DD")) {
				return true
			}
		}
		if d.DateTime.IsAfter(logDate) {
			return true
		}
	case "<=":
		if dateOnly {
			startOfDay := d.DateTime.Clone().StartOfDay()
			if startOfDay.IsAfter(logDate) || startOfDay.IsSame(logDate, "YYYY-MM-DD") {
				return true
			}
		}
		if d.DateTime.IsAfter(logDate) || d.DateTime.IsSame(logDate, queryDateFormat) {
			return true
		}
	}
	return false
}
