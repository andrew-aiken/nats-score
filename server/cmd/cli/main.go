package main

import (
	"context"
	"log"
	"os"

	"server/cmd/checks"
	"server/cmd/initialize"
	"server/cmd/server"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "server",
		Usage: "Control plane for distributed scoring agents",
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
						Usage:     "Describe a check",
						ArgsUsage: "check",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Describe(cmd.Args().First())
						},
					},
					{
						Name:      "validate",
						Usage:     "Validate a check",
						ArgsUsage: "check",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return checks.Validate(cmd.Args().First())
						},
					},
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
			{
				Name:  "server",
				Usage: "Run the scoring controller",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return server.Server()
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
