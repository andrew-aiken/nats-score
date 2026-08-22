package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// User is the persisted record for a user login account
type User struct {
	// Username
	Username string `json:"username"`
	// "observer", or a team number string ("0", "1", ...).
	Team string `json:"team"`
	// bcrypt hash of the user's password
	PasswordHash string `json:"password_hash"`
}

// HashPassword bcrypt-hashes a plaintext password for storage.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a plaintext password against a stored bcrypt hash.
// Returns nil on match, an error otherwise.
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
