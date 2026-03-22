package main

import "time"

func getWeekAndYear(date time.Time) (week, year int) {
	year, week = date.ISOWeek()
	return
}

func getCurrentWeekAndYear() (int, int) {
	return getWeekAndYear(time.Now())
}
