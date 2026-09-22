package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"server/internal/auth"
	"server/internal/nats"
)

// LoginRequest represents the request body for username/password login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// dummyBcryptHash is a fixed valid bcrypt hash used to keep timing/behavior
// identical between "unknown user" and "wrong password" cases, preventing
// username enumeration.
var dummyBcryptHash, _ = auth.HashPassword("dummy-password-for-timing-safety")

const invalidCredentialsError = "Invalid username or password"

// Login handles authentication via username/password against the NATS
// "users" KV bucket.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeJson(w, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		encodeJson(w, map[string]string{"error": invalidCredentialsError})
		return
	}

	user, err := nats.GetUser(h.NatsUsersKVClient, req.Username)
	if err != nil {
		slog.Warn("Failed to look up user", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		encodeJson(w, map[string]string{"error": invalidCredentialsError})
		return
	}

	// Always run the bcrypt comparison, even when the user doesn't exist,
	// against a fixed dummy hash, so unknown-username and wrong-password
	// requests take a comparable code path/timing and return identical
	// responses (no username enumeration).
	hashToCompare := dummyBcryptHash
	if user != (auth.User{}) {
		hashToCompare = user.PasswordHash
	}
	pwErr := auth.VerifyPassword(hashToCompare, req.Password)

	if user == (auth.User{}) || pwErr != nil {
		w.WriteHeader(http.StatusUnauthorized)
		encodeJson(w, map[string]string{"error": invalidCredentialsError})
		return
	}

	creds, err := h.NatsAuthService.GenerateCredentials(req.Username, user.Team)
	if err != nil {
		slog.Error("Failed to generate NATS credentials", "username", req.Username, "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		encodeJson(w, map[string]string{"error": "Failed to generate credentials"})
		return
	}

	encodeJson(w, creds)
}
