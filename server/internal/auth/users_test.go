package auth_test

import (
	"errors"
	"testing"

	"server/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHash(t *testing.T) {
	plaintext := "gcp9dvr3xrw"

	hash, err := auth.HashPassword(plaintext)
	if err != nil {
		t.Fatal("Failed to hash password")
	}

	err = auth.VerifyPassword(hash, plaintext)
	if err != nil {
		t.Fatal("Failed to verify password")
	}
}

func TestFailVerify(t *testing.T) {
	plaintext := "goodValue"
	badPlaintext := "incorrect"

	hash, err := auth.HashPassword(plaintext)
	if err != nil {
		t.Fatal("Failed to hash password")
	}

	err = auth.VerifyPassword(hash, badPlaintext)

	if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		t.Fatal("Password should not match")
	}
}
