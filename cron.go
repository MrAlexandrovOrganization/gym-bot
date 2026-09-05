package main

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

var moscow = time.FixedZone("UTC+3", 3*60*60)

func startCron(schedule string, job func()) {
	c := cron.New(cron.WithLocation(moscow))
	c.AddFunc(schedule, job)
	c.Start()
	log.Printf("[cron] Запущен с расписанием: %s (UTC+3)", schedule)
}
