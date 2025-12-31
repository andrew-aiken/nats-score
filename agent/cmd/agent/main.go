package main

import (
	"context"
	"log"
	"os"

	"github.com/aaiken/nats-score/pkg/run"
	"github.com/urfave/cli/v3"
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			run.Run(run.RunArgs{
				NatsUrl:       cmd.String("nats-address"),
				NatsCredsFile: cmd.String("nats-creds"),
				TeamNumber:    cmd.Int16("team"),
			})
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
