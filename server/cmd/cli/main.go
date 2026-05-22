package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"server/cmd/auth"
	"server/cmd/checks"
	"server/cmd/initialize"
	"server/cmd/server"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "server",
		Usage: "Control plane for distributed scoring",
		Commands: []*cli.Command{
			{
				Name:  "checks",
				Usage: "Manage checks",
				Commands: []*cli.Command{
					{
						Name:    "list",
						Aliases: []string{"ls"},
						Usage:   "Displays loaded checks",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.List()
						},
					},
					{
						Name:    "import",
						Aliases: []string{"add"},
						Usage:   "Loads checks from configuration files",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "directory",
								Aliases:  []string{"d"},
								Usage:    "Directory to load checks from",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Import(cmd.String("directory"))
						},
					},
					{
						Name:  "export",
						Usage: "Write checks to a directory",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "directory",
								Aliases:  []string{"d"},
								Usage:    "Directory to load checks from",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Export(cmd.String("directory"))
						},
					},
					{
						Name:  "purge",
						Usage: "Removes all loaded checks",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:     "force",
								Aliases:  []string{"f"},
								Usage:    "Skip destruction confirmation",
								Required: false,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Purge(cmd.Bool("force"))
						},
					},
					{
						Name:      "remove",
						Aliases:   []string{"rm"},
						Usage:     "Removes a check",
						ArgsUsage: "check",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Remove(cmd.Args().First())
						},
					},
					{
						Name:      "describe",
						Usage:     "Prints out a checks definition",
						ArgsUsage: "check",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Describe(cmd.Args().First())
						},
					},
					{
						Name:      "validate",
						Usage:     "Validates that a check if formatted correctly",
						ArgsUsage: "check",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Validate(cmd.Args().First())
						},
					},
				},
			},
			{
				Name:  "start",
				Usage: "Run the scoring controller",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return server.Server()
				},
			},
			{
				Name:  "nats",
				Usage: "Collection of commands to populate NATS structures",
				Commands: []*cli.Command{
					{
						Name:  "auth",
						Usage: "Generate agent NATS credentials",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:     "standalone",
								Aliases:  []string{"s"},
								Usage:    "Generate agent credentials that support any amount of teams",
								Required: false,
							},
							&cli.IntFlag{
								Name:     "count",
								Aliases:  []string{"c"},
								Usage:    "Number of indivitual agent certs to generate",
								Required: false,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							hasStandalone := cmd.IsSet("standalone")
							hasCount := cmd.IsSet("count")

							if hasStandalone && hasCount {
								return fmt.Errorf("--standalone and --count are mutually exclusive")
							}
							if !hasStandalone && !hasCount {
								return fmt.Errorf("one of --standalone or --count is required")
							}

							if hasCount && 0 > cmd.Int("count") {
								return fmt.Errorf("Count must be a positive number")
							}

							return auth.Auth(cmd.Bool("standalone"), cmd.Int("count"))
						},
					},
					{
						Name:    "initialize",
						Aliases: []string{"init"},
						Usage:   "Initialize NATS KV and streams",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return initialize.Initialize()
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
