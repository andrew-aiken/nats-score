package cron_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"server/pkg/cron"
	"server/pkg/nats"

	"github.com/go-co-op/gocron/v2"
)

func TestRemoveCheckCron(t *testing.T) {
	s, _ := gocron.NewScheduler()
	defer func() { _ = s.Shutdown() }()

	cronjobTag := "dummy"

	s.NewJob(
		gocron.DurationJob(3*time.Second),
		gocron.NewTask(func() {}),
		gocron.WithTags(cronjobTag),
	)

	if len(s.Jobs()) != 1 {
		t.Error("Should be only one job")
	}

	cron.RemoveCheckCron(s, cronjobTag)

	if len(s.Jobs()) != 0 {
		t.Error("Unexpected amount of remaining cron jobs")
	}
}

func TestAddCheckCron(t *testing.T) {
	const checkName string = "example"

	t.Run("Expected functionality", func(t *testing.T) {
		s, _ := gocron.NewScheduler()
		defer func() { _ = s.Shutdown() }()

		var checkFrequency int16 = 30000 // This is set high so it never triggers naturally

		job, err := cron.AddCheckCron(s, nats.NatsConnection{}, checkName, checkFrequency)
		if err != nil {
			t.Error(err)
		}

		if !slices.Contains(job.Tags(), checkName) {
			t.Error("Check tag not in list")
		}

		if len(s.Jobs()) != 1 {
			t.Error("Check not added into cron scheduler")
		}

		s.Start()
		err = job.RunNow()
		if err != nil {
			t.Errorf("Error when triggering job: %v", err)
			t.FailNow()
		}

		if s.RemoveJob(job.ID()) != nil {
			t.Error("Failed to remove job from schedule")
		}
	})

	t.Run("Negative frequency", func(t *testing.T) {
		s, _ := gocron.NewScheduler()
		defer func() { _ = s.Shutdown() }()

		var checkFrequency int16 = -1

		_, err := cron.AddCheckCron(s, nats.NatsConnection{}, checkName, checkFrequency)
		if !errors.Is(err, gocron.ErrDurationJobIntervalNegative) {
			t.Error("Error not properly returned")
		}
	})
}
