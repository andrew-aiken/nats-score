package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"server/pkg/config"
	"server/pkg/discord"
	"server/pkg/middleware"
	"server/pkg/nats"

	"github.com/go-co-op/gocron/v2"
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
	NatsAuthService *nats.NATSAuthService
	NatsKVClient    *nats.NATSKVClient
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
	http.Redirect(w, r, h.OauthConfig.AuthCodeURL(h.State), http.StatusTemporaryRedirect)
}

// Verify validates a NATS JWT token from the Authorization header
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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
	if r.FormValue("state") != h.State {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("State does not match."))
		return
	}

	// Exchange the code for an access token
	token, err := h.OauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// Create Discord client
	client := discord.NewClient(context.Background(), h.OauthConfig, token)

	// Get user info
	user, err := client.GetCurrentUser()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching user: " + err.Error()))
		return
	}

	log.Printf("User Info: ID=%s, Username=%s\n", user.ID, user.Username)

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
		log.Printf("✗ User is NOT in guild %s\n", h.TargetGuildID)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("User is not in the required guild"))
		return
	}

	log.Printf("✓ User IS in guild %s (Name: %s)\n", h.TargetGuildID, targetGuild.Name)

	// Fetch member details to get roles
	member, err := client.GetGuildMember(h.TargetGuildID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching member info: " + err.Error()))
		return
	}

	log.Printf("User's Roles in guild %s:\n", targetGuild.Name)
	if len(member.Roles) == 0 {
		log.Println("  No roles (only @everyone)")
	} else {
		for i, roleID := range member.Roles {
			log.Printf("  %d. Role ID: %s\n", i+1, roleID)
		}
	}
	// if member.Nick != "" {
	// 	log.Printf("  Nickname: %s\n", member.Nick)
	// }

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

	if h.NatsKVClient == nil {
		log.Printf("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	mutableFields, err := h.NatsKVClient.GetMutableFields()
	if err != nil {
		log.Printf("Failed to get mutable fields: %v", err)
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
		log.Printf("NATS KV client not initialized")
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
	// Extract team number from roles
	// teamNumber, ok := nats.GetTeamNumberFromRoles(claims.Roles)
	// if !ok {
	// 	log.Printf("User %s has no team role", claims.Username)
	// 	w.WriteHeader(http.StatusForbidden)
	// 	json.NewEncoder(w).Encode(map[string]string{"error": "No team role found"})
	// 	return
	// }

	switch r.Method {
	case http.MethodGet:
		// Fetch team settings
		settings, err := h.NatsKVClient.GetTeamSettings(teamNumber)
		if err != nil {
			log.Printf("Failed to get team settings for team %s: %v", teamNumber, err)
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
		if err := h.NatsKVClient.PutTeamSettings(teamNumber, settings); err != nil {
			log.Printf("Failed to update team settings for team %s: %v", teamNumber, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save settings"})
			return
		}

		log.Printf("Team %s settings updated by %s", teamNumber, claims.UserID)
		json.NewEncoder(w).Encode(map[string]interface{}{
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

	if h.NatsKVClient == nil {
		log.Printf("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	checks, err := h.NatsKVClient.GetChecks()
	if err != nil {
		log.Printf("Failed to get checks: %v", err)
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
			fmt.Println("Role found: " + teamID)
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
