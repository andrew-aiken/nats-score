package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/aaiken/nats-score/pkg/run"
)

func main() {
	cmd := &cli.Command{
		Name:  "agent",
		Usage: "a distributed scoring service",
		Flags: []cli.Flag{
			&cli.Int16Flag{
				Name:     "team",
				Aliases:  []string{"t"},
				Usage:    "team number",
				Required: true,
				Sources:  cli.EnvVars("SCORE_TEAM_NUMBER"),
			},
			&cli.StringFlag{
				Name:     "nats-creds",
				Usage:    "path to the nats credentials file",
				Required: true,
				Sources:  cli.EnvVars("SCORE_NATS_CREDS_FILE"),
			},
			&cli.StringFlag{
				Name:    "nats-address",
				Usage:   "NATS server address",
				Value:   "nats://127.0.0.1:4222",
				Sources: cli.EnvVars("SCORE_NATS_URL"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Sets the program log level",
				Value:   "info",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run.Run(run.RunArgs{
				LogLevel:      cmd.String("log-level"),
				NatsUrl:       cmd.String("nats-address"),
				NatsCredsFile: cmd.String("nats-creds"),
				TeamNumber:    cmd.Int16("team"),
			})
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
