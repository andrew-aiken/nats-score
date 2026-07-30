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
	"server/cmd/user"

	"server/cmd/agent"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "nats-score",
		Usage: "score",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Control plane for distributed scoring",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "path to the score server configuration file",
						Value:   "config.json",
					},
				},
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
									return checks.List(cmd.String("config"))
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
									return checks.Import(cmd.String("config"), cmd.String("directory"))
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
									return checks.Export(cmd.String("config"), cmd.String("directory"))
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
									return checks.Purge(cmd.String("config"), cmd.Bool("force"))
								},
							},
							{
								Name:      "remove",
								Aliases:   []string{"rm"},
								Usage:     "Removes a check",
								ArgsUsage: "check",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									return checks.Remove(cmd.String("config"), cmd.Args().First())
								},
							},
							{
								Name:      "describe",
								Usage:     "Prints out a checks definition",
								ArgsUsage: "check",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									return checks.Describe(cmd.String("config"), cmd.Args().First())
								},
							},
							{
								Name:      "validate",
								Usage:     "Validates that a check if formatted correctly",
								ArgsUsage: "check",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									return checks.Validate(cmd.String("config"), cmd.Args().First())
								},
							},
						},
					},
					{
						Name:  "start",
						Usage: "Run the scoring controller",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return server.Server(server.ServerArgs{
								LogLevel:       cmd.String("log-level"),
								ConfigFilePath: cmd.String("config"),
							})
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "log-level",
								Aliases:  []string{"l"},
								Usage:    "Sets the program log level",
								Required: false,
								Value:    "info",
							},
						},
					},
					{
						Name:    "user",
						Aliases: []string{"users"},
						Usage:   "Manage username/password login accounts",
						Commands: []*cli.Command{
							{
								Name:  "add",
								Usage: "Create or overwrite a login account",
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "username",
										Aliases:  []string{"u"},
										Usage:    "Login username",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "team",
										Aliases:  []string{"t"},
										Usage:    `Team assignment: "admin", "observer", or a team number (e.g. "0", "1")`,
										Required: true,
									},
									&cli.StringFlag{
										Name:     "password",
										Aliases:  []string{"p"},
										Usage:    "Password (omit to be prompted securely)",
										Required: true,
									},
									&cli.BoolFlag{
										Name:    "force",
										Aliases: []string{"f"},
										Usage:   "Overwrite an existing user with this username",
									},
								},
								Action: func(ctx context.Context, cmd *cli.Command) error {
									return user.Add(cmd.String("config"), cmd.String("username"), cmd.String("team"), cmd.String("password"), cmd.Bool("force"))
								},
							},
							{
								Name:    "list",
								Aliases: []string{"ls"},
								Usage:   "Lists all registered login accounts",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									return user.List(cmd.String("config"))
								},
							},
							{
								Name:      "remove",
								Aliases:   []string{"rm"},
								Usage:     "Removes a login account",
								ArgsUsage: "username",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									return user.Remove(cmd.String("config"), cmd.Args().First())
								},
							},
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
									return initialize.Initialize(cmd.String("config"))
								},
							},
						},
					},
				},
			},
			{
				Name:  "agent",
				Usage: "a distributed scoring service",
				Flags: []cli.Flag{
					&cli.Uint16Flag{
						Name:    "team",
						Aliases: []string{"t"},
						Usage:   "single team number (mutually exclusive with --teams)",
					},
					&cli.StringFlag{
						Name:    "teams",
						Aliases: []string{"T"},
						Usage:   `comma-separated team numbers or ranges, e.g. "1,3,5-8,10" (mutually exclusive with --team)`,
					},
					&cli.StringFlag{
						Name:     "nats-creds",
						Usage:    "path to the nats credentials file",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "nats-address",
						Usage: "NATS server address",
						Value: "nats://127.0.0.1:4222",
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
						teamNumbers, err = agent.ParseTeams(cmd.String("teams"))
						if err != nil {
							return fmt.Errorf("invalid --teams value: %w", err)
						}
					}

					return agent.Run(agent.RunArgs{
						LogLevel:      cmd.String("log-level"),
						NatsUrl:       cmd.String("nats-address"),
						NatsCredsFile: cmd.String("nats-creds"),
						TeamNumbers:   teamNumbers,
					})
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
