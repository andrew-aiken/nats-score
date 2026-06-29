package cron

import (
	"log"
	"time"

	"server/pkg/nats"

	"github.com/go-co-op/gocron/v2"
)

func AddCheckCron(cron gocron.Scheduler, natsClient nats.NatsConnection, checkName string, checkFrequency int16) (gocron.Job, error) {
	return cron.NewJob(
		gocron.DurationJob(time.Duration(checkFrequency)*time.Second),
		gocron.NewTask(
			func(checkName string) {
				log.Printf("Published trigger for %s the check\n", checkName)
				natsClient.NatsConn.Publish("events.score."+checkName, []byte{})
			},
			checkName,
		),
		gocron.WithTags(checkName),
	)
}

func RemoveCheckCron(cron gocron.Scheduler, checkName string) {
	cron.RemoveByTags(checkName)
}
