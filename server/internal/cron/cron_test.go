package cron_test

import (
	"slices"
	"testing"
	"time"

	"server/internal/cron"

	"github.com/go-co-op/gocron/v2"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
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

		var checkFrequency uint16 = 30000 // This is set high so it never triggers naturally

		opts := natsserver.DefaultTestOptions
		opts.Port = -1
		opts.StoreDir = t.TempDir()

		server := natsserver.RunServer(&opts)
		defer server.Shutdown()

		nc, err := nats.Connect(server.Addr().String())
		if err != nil {
			t.Fatalf("Failed to connect to NATS: %v", err)
		}
		defer nc.Close()
		if err = nc.Flush(); err != nil {
			t.Errorf("Error when flushing nats connection: %v", err)
		}

		// Setup a channel to follow score check
		ch := make(chan *nats.Msg, 64)
		_, err = nc.ChanSubscribe("events.score."+checkName, ch)

		job, err := cron.AddCheckCron(s, nc, checkName, checkFrequency)
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
			t.Fatalf("Error when triggering job: %v", err)
		}

		select {
		case msg := <-ch:
			if len(msg.Data) != 0 {
				t.Error("Expected 0 length nats response")
			}
		case <-time.After(3 * time.Second):
			t.Error("Timed out waiting for nats event")
		}

		if s.RemoveJob(job.ID()) != nil {
			t.Error("Failed to remove job from schedule")
		}
	})
}
