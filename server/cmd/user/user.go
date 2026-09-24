package user

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/andrew-aiken/score/internal/auth"
	"github.com/andrew-aiken/score/internal/logging"
	"github.com/andrew-aiken/score/internal/nats"
)

// Add creates a new user login account in the NATS "users" KV bucket.
func Add(natsAddress string, natsCreds string, username, team, password string, force bool) error {
	logging.SetupLogging("warn")

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

	natsClient := nats.NatsConnection{
		NatsUrl:       natsAddress,
		NatsCredsFile: natsCreds,
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

	if exists {
		fmt.Printf("Update user %s\n", username)
	} else {
		fmt.Printf("Created user %s\n", username)
	}
	return nil
}

// List prints all teams and their registered users.
func List(natsAddress string, natsCreds string) error {
	logging.SetupLogging("warn")

	natsClient := nats.NatsConnection{
		NatsUrl:       natsAddress,
		NatsCredsFile: natsCreds,
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

	teamUserMap := map[string][]string{}

	for _, u := range users {
		teamUserMap[u.Team] = append(teamUserMap[u.Team], u.Username)
	}

	for team, users := range teamUserMap {
		fmt.Printf("Team %s\n", team)

		for _, user := range users {
			fmt.Printf("  %s\n", user)
		}
	}

	return nil
}

// Remove deletes a user login account from the NATS "users" KV bucket.
func Remove(natsAddress string, natsCreds string, username string) error {
	logging.SetupLogging("warn")

	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username must not be empty")
	}

	natsClient := nats.NatsConnection{
		NatsUrl:       natsAddress,
		NatsCredsFile: natsCreds,
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

	fmt.Printf("Removed user %s\n", username)
	return nil
}
