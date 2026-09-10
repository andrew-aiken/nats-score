package nats

import (
	"encoding/json"
	"fmt"
	"strings"

	"server/internal/auth"

	"github.com/nats-io/nats.go"
)

// GetUser retrieves a user record by username from the users KV bucket.
// Returns (nil, nil) if the user does not exist so callers can distinguish
// "not found" from a real error.
func GetUser(natsKV nats.KeyValue, username string) (auth.User, error) {
	var user auth.User
	key := "user." + username

	entry, err := natsKV.Get(key)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return user, nil
		}
		return user, fmt.Errorf("failed to get '%s' key: %w", key, err)
	}

	if err := json.Unmarshal(entry.Value(), &user); err != nil {
		return user, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return user, nil
}

// PutUser writes (or overwrites) a user record in the users KV bucket.
func PutUser(natsKV nats.KeyValue, user auth.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	key := "user." + user.Username
	if _, err := natsKV.Put(key, data); err != nil {
		return fmt.Errorf("failed to put '%s' key: %w", key, err)
	}

	return nil
}

// ListUsers retrieves all user records from the users KV bucket.
func ListUsers(natsKV nats.KeyValue) ([]auth.User, error) {
	var users []auth.User

	keys, err := natsKV.ListKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	for key := range keys.Keys() {
		if !strings.HasPrefix(key, "user.") {
			continue
		}

		entry, err := natsKV.Get(key)
		if err != nil {
			continue
		}

		var user auth.User
		if err := json.Unmarshal(entry.Value(), &user); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user '%s': %w", key, err)
		}

		users = append(users, user)
	}

	return users, nil
}

// DeleteUser removes a user record from the users KV bucket.
func DeleteUser(natsKV nats.KeyValue, username string) error {
	key := "user." + username
	if err := natsKV.Delete(key); err != nil {
		return fmt.Errorf("failed to delete '%s' key: %w", key, err)
	}
	return nil
}

// UserExists reports whether a username is already registered.
func UserExists(natsKV nats.KeyValue, username string) (bool, error) {
	user, err := GetUser(natsKV, username)
	if err != nil {
		return false, err
	}
	return user != auth.User{}, nil
}
