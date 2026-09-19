package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"server/cmd/agent"
	"server/cmd/auth"
	"server/cmd/checks"
	"server/cmd/initialize"
	"server/cmd/query"
	"server/cmd/server"
	"server/cmd/user"
	"server/internal/sink"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "score",
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
								UsageText: "score server check rm <name>",
								ArgsUsage: "check",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									checkName := cmd.Args().First()

									if checkName == "" {
										fmt.Println("Check name required")
										fmt.Println(cmd.UsageText)
										return nil
									}

									return checks.Remove(cmd.String("config"), checkName)
								},
							},
							{
								Name:      "describe",
								Usage:     "Prints out a checks definition",
								UsageText: "score server check describe <name>",
								ArgsUsage: "check",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									checkName := cmd.Args().First()

									if checkName == "" {
										fmt.Println("Check name required")
										fmt.Println(cmd.UsageText)
										return nil
									}

									return checks.Describe(cmd.String("config"), checkName)
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
								ConfigFilePath: cmd.String("config"),
								LogLevel:       cmd.String("log-level"),
								DB:             cmd.Bool("db"),
								DBPath:         cmd.String("db-path"),
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
							&cli.BoolFlag{
								Name:  "db",
								Usage: "Store NATS results messages into a sqlite database for simpler querying",
							},
							&cli.StringFlag{
								Name:     "db-path",
								Usage:    "Path to sqlite database file",
								Required: false,
								Value:    "results.db",
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

									return auth.Auth(auth.CliParameters{
										ConfigFile: cmd.String("config"),
										Standalone: cmd.Bool("standalone"),
										Teams:      cmd.Uint16("count"),
									})
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
				Name:  "query",
				Usage: "Query prints summed team points over a time range",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "db-path",
						Usage:    "Path to sqlite database file",
						Required: false,
						Value:    "results.db",
					},
					&cli.TimestampFlag{
						Name:   "start",
						Usage:  "Start of the time range (RFC3339); defaults to 24 hours ago",
						Config: cli.TimestampConfig{Layouts: []string{time.RFC3339}},
					},
					&cli.TimestampFlag{
						Name:   "end",
						Usage:  "End of the time range (RFC3339); defaults to now",
						Config: cli.TimestampConfig{Layouts: []string{time.RFC3339}},
					},
					&cli.Uint16Flag{
						Name:  "team",
						Usage: "Filter to a single team ID",
					},
					&cli.StringFlag{
						Name:  "check",
						Usage: "Filter to a single check name",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return query.List(os.Stdout, cmd.String("db-path"), buildScoreQuery(cmd))
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
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}

// buildScoreQuery builds a sink.ScoreQuery from the "query" command's
// shared flags, applying the --start/--end defaults and the --team
// IsSet check shared by both "query total" and "query graph".
func buildScoreQuery(cmd *cli.Command) sink.ScoreQuery {
	start := cmd.Timestamp("start")
	if start.IsZero() {
		start = time.Now().Add(-24 * time.Hour)
	}

	end := cmd.Timestamp("end")
	if end.IsZero() {
		end = time.Now()
	}

	query := sink.ScoreQuery{
		Start:     start,
		End:       end,
		CheckName: cmd.String("check"),
	}
	if cmd.IsSet("team") {
		team := cmd.Uint16("team")
		query.TeamID = &team
	}

	return query
}
