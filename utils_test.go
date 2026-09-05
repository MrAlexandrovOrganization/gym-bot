package main

import (
	"testing"
	"time"
)

func TestGetWeekAndYearUsesISOWeek(t *testing.T) {
	week, year := getWeekAndYear(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if week != 1 || year != 2024 {
		t.Fatalf("getWeekAndYear() = (%d, %d), want (1, 2024)", week, year)
	}
}
