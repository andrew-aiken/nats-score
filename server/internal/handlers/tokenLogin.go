package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// TokenLoginRequest represents the request body for token-based login
type TokenLoginRequest struct {
	Token string `json:"token"`
}

// TokenLogin handles authentication via predefined access tokens
func (h *Handler) TokenLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Parse request body
	var req TokenLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate token is not empty
	if req.Token == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token is required"})
		return
	}

	var userRole string

	for name, token := range h.AccessTokens {
		if token == req.Token {
			userRole = name
			continue
		}
	}

	if userRole == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid token"})
		return
	}

	var userRoleID string

	for roleID, roleName := range h.RoleMap {
		if roleName == userRole {
			userRoleID = roleID
			continue
		}
	}

	if userRoleID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "No valid role found"})
		return
	}

	// Generate NATS credentials with the token's role
	roles := []string{userRoleID}
	creds, err := h.NatsAuthService.GenerateCredentials(
		fmt.Sprintf("token:%s", userRole), // Use token: prefix for user ID
		userRole,
		roles,
	)
	if err != nil {
		slog.Error("Failed to generate NATS credentials for token user", "roleID", userRoleID, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate credentials"})
		return
	}

	// Return the credentials
	json.NewEncoder(w).Encode(creds)
}
