package data

import "time"

func getTTL(date time.Time, days int) int64 {
	return date.AddDate(0, 0, days).Unix()
}

// budgetPeriod gets the epoch for the start of a period
func getBudgetPeriodTime(date time.Time, budgetPeriod string) time.Time {
	var new time.Time
	if budgetPeriod == "MONTHLY" {
		new = time.Date(date.Year(), date.Month(), 0, 0, 0, 0, 0, time.UTC)
	} else {
		new = firstDayOfISOWeek(date.ISOWeek())
	}

	return new
}

func firstDayOfISOWeek(year int, week int) time.Time {
	date := time.Date(year, 0, 0, 0, 0, 0, 0, time.UTC)
	isoYear, isoWeek := date.ISOWeek()

	// iterate back to Monday
	for date.Weekday() != time.Monday {
		date = date.AddDate(0, 0, -1)
		isoYear, isoWeek = date.ISOWeek()
	}

	// iterate forward to the first day of the first week
	for isoYear < year {
		date = date.AddDate(0, 0, 7)
		isoYear, isoWeek = date.ISOWeek()
	}

	// iterate forward to the first day of the given week
	for isoWeek < week {
		date = date.AddDate(0, 0, 7)
		isoYear, isoWeek = date.ISOWeek()
	}

	return date
}
