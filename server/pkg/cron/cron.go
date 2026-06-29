package cron

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/nats-io/nats.go"
)

func AddCheckCron(cron gocron.Scheduler, natsConnection *nats.Conn, checkName string, checkFrequency int16) (gocron.Job, error) {
	return cron.NewJob(
		gocron.DurationJob(time.Duration(checkFrequency)*time.Second),
		gocron.NewTask(
			func(checkName string) {
				log.Printf("Published trigger for %s the check\n", checkName)
				natsConnection.Publish("events.score."+checkName, []byte{})
			},
			checkName,
		),
		gocron.WithTags(checkName),
	)
}

func RemoveCheckCron(cron gocron.Scheduler, checkName string) {
	cron.RemoveByTags(checkName)
}
