package user

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"server/internal/auth"
	"server/internal/config"
	"server/internal/logging"
	"server/internal/nats"

	"golang.org/x/term"
)

// Add creates a new username/password login account in the NATS "users" KV
// bucket. If password is empty, it is read interactively via a masked
// terminal prompt (with confirmation re-entry). Refuses to overwrite an
// existing username unless force is true.
func Add(username, team, password string, force bool) error {
	logging.SetupLogging("info")

	username = strings.TrimSpace(username)
	team = strings.TrimSpace(team)

	if username == "" {
		return fmt.Errorf("username must not be empty")
	}
	if team == "" {
		return fmt.Errorf("team must not be empty")
	}

	cfg, err := config.Load("config.json")
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

	if password == "" {
		password, err = promptPassword()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
	}
	if password == "" {
		return fmt.Errorf("password must not be empty")
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
func List() error {
	logging.SetupLogging("info")

	cfg, err := config.Load("config.json")
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
func Remove(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username must not be empty")
	}

	cfg, err := config.Load("config.json")
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

// promptPassword reads a password from the terminal with echo disabled,
// with a confirmation re-entry to catch typos.
func promptPassword() (string, error) {
	fmt.Print("Password: ")
	pw1, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}

	fmt.Print("Confirm password: ")
	pw2, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}

	if string(pw1) != string(pw2) {
		return "", fmt.Errorf("passwords do not match")
	}

	return string(pw1), nil
}
