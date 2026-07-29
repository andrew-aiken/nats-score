package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"server/internal/auth"
	"server/internal/config"
	"server/internal/middleware"
	"server/internal/nats"

	"github.com/go-co-op/gocron/v2"
	natsnats "github.com/nats-io/nats.go"
)

// TokenConfig represents a predefined access token with associated role
type TokenConfig struct {
	Role     string // Role ID for permissions
	Username string // Display name for the token user
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	NatsAuthService *auth.NATSAuthService
	NatsKVClient    natsnats.KeyValue
	RoleMap         config.DiscordRoleMap // role ID -> role name
	AccessTokens    config.StaticAuthMap  // access token -> config
	CronScheduler   gocron.Scheduler
}

// NewHandler creates a new Handler with the given dependencies
func NewHandler(handler *Handler) *Handler {
	return handler
}

// Verify validates a NATS JWT token from the Authorization header
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]bool{"valid": false})
		return
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]bool{"valid": false})
		return
	}

	tokenString := parts[1]
	_, err := h.NatsAuthService.VerifyJWT(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]bool{"valid": false})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"valid": true})
}

// GetMutableFields returns a map of check names to their mutable fields
// from the NATS KV settings bucket
func (h *Handler) GetMutableFields(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	if h.NatsKVClient == nil {
		slog.Warn("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	mutableFields, err := nats.GetMutableFields(h.NatsKVClient)
	if err != nil {
		slog.Warn("Failed to get mutable fields", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve mutable fields"})
		return
	}

	json.NewEncoder(w).Encode(mutableFields)
}

// TeamSettings handles GET and PUT for team-specific settings in NATS KV
// The team number is extracted from the user's JWT roles
func (h *Handler) TeamSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.NatsKVClient == nil {
		slog.Warn("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	// Get claims from context (set by auth middleware)
	claims := middleware.GetClaimsFromContext(r.Context())
	if claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	teamNumber := claims.TeamID

	switch r.Method {
	case http.MethodGet:
		// Fetch team settings
		settings, err := nats.GetTeamSettings(h.NatsKVClient, teamNumber)
		if err != nil {
			slog.Warn("Failed to get team settings", "team", teamNumber, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve settings"})
			return
		}
		json.NewEncoder(w).Encode(settings)

	case http.MethodPut:
		// Parse request body
		var settings map[string]map[string]string
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		// Write to NATS KV
		if err := nats.PutTeamSettings(h.NatsKVClient, teamNumber, settings); err != nil {
			slog.Warn("Failed to update team settings", "team", teamNumber, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save settings"})
			return
		}

		slog.Info("Team settings updated", "team", teamNumber, "user", claims.UserID)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"team":    teamNumber,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

// Checks returns a sorted list of all check names from the NATS KV settings bucket
func (h *Handler) Checks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	if h.NatsKVClient == nil {
		slog.Warn("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	checks, err := nats.GetChecks(h.NatsKVClient)
	if err != nil {
		slog.Warn("Failed to get checks", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve checks"})
		return
	}

	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)

	json.NewEncoder(w).Encode(names)
}
