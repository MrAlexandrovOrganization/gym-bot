package main

import "time"

func getWeekAndYear(date time.Time) (int, int) {
	return date.ISOWeek()
}

func getCurrentWeekAndYear() (int, int) {
	return getWeekAndYear(time.Now())
}
