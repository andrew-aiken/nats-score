package user

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"server/internal/auth"
	"server/internal/config"
	"server/internal/logging"
	"server/internal/nats"
)

// Add creates a new username/password login account in the NATS "users" KV
// bucket. If password is empty, it is read interactively via a masked
// terminal prompt (with confirmation re-entry). Refuses to overwrite an
// existing username unless force is true.
func Add(configFile string, username, team, password string, force bool) error {
	logging.SetupLogging("info")

	username = strings.TrimSpace(username)
	team = strings.TrimSpace(team)

	if username == "" {
		return fmt.Errorf("username must not be empty")
	}
	if team == "" {
		return fmt.Errorf("team must not be empty")
	}
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load config")
		return err
	}

	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	if err := natsClient.SetupConnection(); err != nil {
		slog.Warn("Failed to initialize NATS KV client")
		return err
	}
	defer natsClient.Close()

	if err := natsClient.SetupUsersKV(); err != nil {
		slog.Warn("Failed to initialize NATS users KV client")
		return err
	}

	exists, err := nats.UserExists(natsClient.NatsUsersKV, username)
	if err != nil {
		return fmt.Errorf("failed to check for existing user: %w", err)
	}
	if exists && !force {
		return fmt.Errorf("user %q already exists (use --force to overwrite)", username)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	user := auth.User{Username: username, Team: team, PasswordHash: hash}
	if err := nats.PutUser(natsClient.NatsUsersKV, user); err != nil {
		return fmt.Errorf("failed to store user: %w", err)
	}

	slog.Info("User added", "username", username, "team", team)
	return nil
}

// List prints all registered usernames and their team assignment. Password
// hashes are never printed.
func List(configFile string,) error {
	logging.SetupLogging("info")

	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load config")
		return err
	}

	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	if err := natsClient.SetupConnection(); err != nil {
		slog.Warn("Failed to initialize NATS KV client")
		return err
	}
	defer natsClient.Close()

	if err := natsClient.SetupUsersKV(); err != nil {
		slog.Warn("Failed to initialize NATS users KV client")
		return err
	}

	users, err := nats.ListUsers(natsClient.NatsUsersKV)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})

	for _, u := range users {
		slog.Info(u.Username, "team", u.Team)
	}

	return nil
}

// Remove deletes a username/password login account from the NATS "users" KV
// bucket.
func Remove(configFile string, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username must not be empty")
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load config")
		return err
	}

	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	if err := natsClient.SetupConnection(); err != nil {
		slog.Warn("Failed to initialize NATS KV client")
		return err
	}
	defer natsClient.Close()

	if err := natsClient.SetupUsersKV(); err != nil {
		slog.Warn("Failed to initialize NATS users KV client")
		return err
	}

	exists, err := nats.UserExists(natsClient.NatsUsersKV, username)
	if err != nil {
		return fmt.Errorf("failed to check for existing user: %w", err)
	}
	if !exists {
		return fmt.Errorf("user %q does not exist", username)
	}

	if err := nats.DeleteUser(natsClient.NatsUsersKV, username); err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	slog.Info("User removed", "username", username)
	return nil
}
