package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"server/pkg/config"
	"server/pkg/discord"
	"server/pkg/middleware"
	"server/pkg/nats"

	"golang.org/x/oauth2"
)

// TokenConfig represents a predefined access token with associated role
type TokenConfig struct {
	Role     string // Role ID for permissions
	Username string // Display name for the token user
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	oauthConfig     *oauth2.Config
	natsAuthService *nats.NATSAuthService
	natsKVClient    *nats.NATSKVClient
	targetGuildID   string
	roleMap         config.DiscordRoleMap // role ID -> role name
	accessTokens    config.StaticAuthMap  // access token -> config
	state           string
	frontendURL     string
}

// NewHandler creates a new Handler with the given dependencies
func NewHandler(oauthConfig *oauth2.Config, natsAuthService *nats.NATSAuthService, natsKVClient *nats.NATSKVClient, targetGuildID string, roleMap config.DiscordRoleMap, accessTokens config.StaticAuthMap, state string, frontendURL string) *Handler {
	return &Handler{
		oauthConfig:     oauthConfig,
		natsAuthService: natsAuthService,
		natsKVClient:    natsKVClient,
		targetGuildID:   targetGuildID,
		roleMap:         roleMap,
		accessTokens:    accessTokens,
		state:           state,
		frontendURL:     frontendURL,
	}
}

// Login redirects to the OAuth 2.0 Authorization page
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.oauthConfig.AuthCodeURL(h.state), http.StatusTemporaryRedirect)
}

// TokenLoginRequest represents the request body for token-based login
type TokenLoginRequest struct {
	Token string `json:"token"`
}

// TokenLoginResponse represents the response for successful token login
type TokenLoginResponse struct {
	JWT  string `json:"jwt"`
	Seed string `json:"seed"`
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

	for name, token := range h.accessTokens {
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

	for roleID, roleName := range h.roleMap {
		fmt.Println("RoleID:", roleID, "RoleName:", roleName)
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
	creds, err := h.natsAuthService.GenerateCredentials(
		fmt.Sprintf("token:%s", userRole), // Use token: prefix for user ID
		userRole,
		roles,
	)
	if err != nil {
		log.Printf("Failed to generate NATS credentials for token user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate credentials"})
		return
	}

	// Return the credentials
	json.NewEncoder(w).Encode(TokenLoginResponse{
		JWT:  creds.JWT,
		Seed: creds.Seed,
	})
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
	_, err := h.natsAuthService.VerifyJWT(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]bool{"valid": false})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"valid": true})
}

// Callback handles the OAuth 2.0 callback
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("state") != h.state {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("State does not match."))
		return
	}

	// Exchange the code for an access token
	token, err := h.oauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// Create Discord client
	client := discord.NewClient(context.Background(), h.oauthConfig, token)

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
	targetGuild := discord.FindGuild(guilds, h.targetGuildID)
	if targetGuild == nil {
		log.Printf("✗ User is NOT in guild %s\n", h.targetGuildID)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("User is not in the required guild"))
		return
	}

	log.Printf("✓ User IS in guild %s (Name: %s)\n", h.targetGuildID, targetGuild.Name)

	// Fetch member details to get roles
	member, err := client.GetGuildMember(h.targetGuildID)
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
	if member.Nick != "" {
		log.Printf("  Nickname: %s\n", member.Nick)
	}

	// Generate NATS credentials for authorized user
	creds, err := h.natsAuthService.GenerateCredentials(user.ID, user.Username, member.Roles)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error generating NATS credentials: " + err.Error()))
		return
	}

	// Redirect to frontend with credentials as URL params
	redirectURL := fmt.Sprintf("%s/auth/callback?jwt=%s&seed=%s",
		h.frontendURL,
		url.QueryEscape(creds.JWT),
		url.QueryEscape(creds.Seed))
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// TestAuth is a test endpoint that returns the authentication result
// This endpoint uses the middleware to check auth and returns the result
func (h *Handler) TestAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get claims and matched roles from context (set by middleware)
	claims := middleware.GetClaimsFromContext(r.Context())
	matchedRoles := middleware.GetMatchedRolesFromContext(r.Context())

	response := map[string]interface{}{
		"authorized":    true,
		"user_id":       claims.UserID,
		"username":      claims.Username,
		"roles":         claims.Roles,
		"matched_roles": matchedRoles,
		"pub_allow":     claims.PubAllow,
		"sub_allow":     claims.SubAllow,
		"expires_at":    claims.ExpiresAt,
	}

	json.NewEncoder(w).Encode(response)
}

// GetMutableFields returns a map of check names to their mutable fields
// from the NATS KV settings bucket
func (h *Handler) GetMutableFields(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.natsKVClient == nil {
		log.Printf("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	mutableFields, err := h.natsKVClient.GetMutableFields()
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

	if h.natsKVClient == nil {
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

	// Extract team number from roles
	teamNumber, ok := nats.GetTeamNumberFromRoles(claims.Roles)
	if !ok {
		log.Printf("User %s has no team role", claims.Username)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "No team role found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Fetch team settings
		settings, err := h.natsKVClient.GetTeamSettings(teamNumber)
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
		if err := h.natsKVClient.PutTeamSettings(teamNumber, settings); err != nil {
			log.Printf("Failed to update team settings for team %s: %v", teamNumber, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save settings"})
			return
		}

		log.Printf("Team %s settings updated by %s", teamNumber, claims.Username)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"team":    teamNumber,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}
