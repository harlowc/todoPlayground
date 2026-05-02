package main

import (
	"fmt"
	"time"
)

func tomorrow(today time.Time) string {
	return today.AddDate(0, 0, 1).Format("2006-01-02")
}

func nextWeekday(today time.Time) string {
	next := today.AddDate(0, 0, 1)
	for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}
	return next.Format("2006-01-02")
}

func tomorrowFallsOnWeekend(today time.Time) bool {
	next := today.AddDate(0, 0, 1)
	return next.Weekday() == time.Saturday || next.Weekday() == time.Sunday
}

func nextWeekdayPrompt(today time.Time) string {
	tomorrowDate := today.AddDate(0, 0, 1)
	next, err := time.Parse("2006-01-02", nextWeekday(today))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Tomorrow is %s. Recreate for %s instead?", tomorrowDate.Weekday(), next.Weekday())
}
