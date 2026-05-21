package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/andrew-aiken/nats-score/agent/pkg/run"
)

func main() {
	cmd := &cli.Command{
		Name:  "agent",
		Usage: "a distributed scoring service",
		Flags: []cli.Flag{
			&cli.Uint16Flag{
				Name:    "team",
				Aliases: []string{"t"},
				Usage:   "single team number (mutually exclusive with --teams)",
				// Sources: cli.EnvVars("SCORE_TEAM_NUMBER"),
			},
			&cli.StringFlag{
				Name:    "teams",
				Aliases: []string{"T"},
				Usage:   `comma-separated team numbers or ranges, e.g. "1,3,5-8,10" (mutually exclusive with --team)`,
				// Sources: cli.EnvVars("SCORE_TEAM_NUMBERS"),
			},
			&cli.StringFlag{
				Name:     "nats-creds",
				Usage:    "path to the nats credentials file",
				Required: true,
				// Sources:  cli.EnvVars("SCORE_NATS_CREDS_FILE"),
			},
			&cli.StringFlag{
				Name:    "nats-address",
				Usage:   "NATS server address",
				Value:   "nats://127.0.0.1:4222",
				// Sources: cli.EnvVars("SCORE_NATS_URL"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Sets the program log level",
				Value:   "info",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			hasTeam := cmd.IsSet("team")
			hasTeams := cmd.IsSet("teams")

			if hasTeam && hasTeams {
				return fmt.Errorf("--team and --teams are mutually exclusive")
			}
			if !hasTeam && !hasTeams {
				return fmt.Errorf("one of --team or --teams is required")
			}

			var teamNumbers []uint16
			if hasTeam {
				teamNumbers = []uint16{cmd.Uint16("team")}
			} else {
				var err error
				teamNumbers, err = run.ParseTeams(cmd.String("teams"))
				if err != nil {
					return fmt.Errorf("invalid --teams value: %w", err)
				}
			}

			return run.Run(run.RunArgs{
				LogLevel:      cmd.String("log-level"),
				NatsUrl:       cmd.String("nats-address"),
				NatsCredsFile: cmd.String("nats-creds"),
				TeamNumbers:   teamNumbers,
			})
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
