package cron

import (
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/nats-io/nats.go"
)

func AddCheckCron(cron gocron.Scheduler, natsConnection *nats.Conn, checkName string, checkFrequency int16) (gocron.Job, error) {
	return cron.NewJob(
		gocron.DurationJob(time.Duration(checkFrequency)*time.Second),
		gocron.NewTask(
			func(checkName string) {
				slog.Debug("Published check trigger", "check", checkName)
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
