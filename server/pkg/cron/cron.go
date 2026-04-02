package cron

import (
	"log"
	"time"

	"server/pkg/nats"

	"github.com/go-co-op/gocron/v2"
)

func AddCheckCron(cron gocron.Scheduler, client *nats.NATSKVClient, checkName string, checkFrequency int16) {
	cron.NewJob(
		gocron.DurationJob(
			time.Duration(checkFrequency)*time.Second,
		),
		gocron.NewTask(
			func(checkName string) {
				log.Printf("Published trigger for %s the check\n", checkName)
				client.GetNATSClient().Publish("events.score."+checkName, []byte{})
			},
			checkName,
		),
		gocron.WithTags(checkName),
	)
}

func RemoveCheckCron(cron gocron.Scheduler, checkName string) {
	cron.RemoveByTags(checkName)
}
