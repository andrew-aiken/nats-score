package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"server/internal/auth"
	"server/internal/config"
	"server/internal/discord"
	"server/internal/middleware"
	"server/internal/nats"

	"github.com/go-co-op/gocron/v2"
	natsnats "github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

// TokenConfig represents a predefined access token with associated role
type TokenConfig struct {
	Role     string // Role ID for permissions
	Username string // Display name for the token user
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	OauthConfig     *oauth2.Config
	NatsAuthService *auth.NATSAuthService
	NatsKVClient    natsnats.KeyValue
	TargetGuildID   string
	RoleMap         config.DiscordRoleMap // role ID -> role name
	AccessTokens    config.StaticAuthMap  // access token -> config
	State           string
	FrontendURL     string
	CronScheduler   gocron.Scheduler
}

// NewHandler creates a new Handler with the given dependencies
func NewHandler(handler *Handler) *Handler {
	return handler
}

// Login redirects to the OAuth 2.0 Authorization page
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	state, err := generateOAuthState()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create oauth state"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/auth/callback",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	http.Redirect(w, r, h.OauthConfig.AuthCodeURL(state), http.StatusTemporaryRedirect)
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

// Callback handles the OAuth 2.0 callback
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing oauth state"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/auth/callback",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	if r.FormValue("state") != stateCookie.Value {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("State does not match."))
		return
	}

	// Exchange the code for an access token
	token, err := h.OauthConfig.Exchange(r.Context(), r.FormValue("code"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// Create Discord client
	client := discord.NewClient(r.Context(), h.OauthConfig, token)

	// Get user info
	user, err := client.GetCurrentUser()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching user: " + err.Error()))
		return
	}

	slog.Info("User Info", "ID", user.ID, "Username", user.Username)

	// Get user's guilds
	guilds, err := client.GetUserGuilds()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching guilds: " + err.Error()))
		return
	}

	// Check if user is in the target guild
	targetGuild := discord.FindGuild(guilds, h.TargetGuildID)
	if targetGuild == nil {
		slog.Warn("User is not in Discord server", "guild", h.TargetGuildID)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("User is not in the required guild"))
		return
	}

	slog.Debug("User is in Discord server", "guild", h.TargetGuildID)

	// Fetch member details to get roles
	member, err := client.GetGuildMember(h.TargetGuildID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching member info: " + err.Error()))
		return
	}

	slog.Debug(fmt.Sprintf("User's Roles in Discord server %s", targetGuild.Name))
	if len(member.Roles) == 0 {
		slog.Warn("User does not have any associated roles", "username", user.Username)
	} else {
		for _, roleID := range member.Roles {
			slog.Debug("User roles", "username", user.Username, "roleID", roleID)
		}
	}

	team, err := validateRoles(h.RoleMap, member.Roles)

	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(err.Error()))
		return
	}

	// Generate NATS credentials for authorized user
	creds, err := h.NatsAuthService.GenerateCredentials(user.ID, team, member.Roles)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error generating NATS credentials: " + err.Error()))
		return
	}

	// Redirect to frontend with credentials as URL params
	redirectURL := fmt.Sprintf("%s/auth/callback?jwt=%s&seed=%s",
		h.FrontendURL,
		url.QueryEscape(creds.JWT),
		url.QueryEscape(creds.Seed))
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
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

func validateRoles(roleMap config.DiscordRoleMap, userRoles []string) (teamID string, error error) {
	var userValidRoles []string

	for k := range userRoles {
		if teamID, ok := roleMap[userRoles[k]]; ok {
			slog.Debug("Found role on user claim", "role", teamID)
			userValidRoles = append(userValidRoles, teamID)
		}
	}

	switch len(userValidRoles) {
	case 0:
		return "", fmt.Errorf("user has no valid roles")
	case 1:
		return userValidRoles[0], nil
	default:
		return "", fmt.Errorf("user is assigned to many roles")
	}
}

func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
