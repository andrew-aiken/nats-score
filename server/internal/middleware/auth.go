package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"server/internal/auth"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// ClaimsContextKey is the key for storing claims in context
	ClaimsContextKey contextKey = "claims"
	// MatchedRolesContextKey is the key for storing matched roles in context
	MatchedRolesContextKey contextKey = "matchedRoles"
)

// AuthMiddleware holds dependencies for authentication middleware
type AuthMiddleware struct {
	natsAuthService *auth.NATSAuthService
	requiredRoles   map[string]string
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(natsAuthService *auth.NATSAuthService, requiredRoles map[string]string) *AuthMiddleware {
	return &AuthMiddleware{
		natsAuthService: natsAuthService,
		requiredRoles:   requiredRoles,
	}
}

// AuthResult represents the authentication result
type AuthResult struct {
	Authorized   bool              `json:"authorized"`
	UserID       string            `json:"user_id,omitempty"`
	TeamID       string            `json:"team_id,omitempty"`
	MatchedRoles map[string]string `json:"matched_roles,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// RequireAdminAuth is a middleware that checks if the request has an admin role
func (m *AuthMiddleware) RequireAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, claim := m.authenticate(r)

		if !result.Authorized {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(result)
			return
		}

		for _, role := range claim.Roles {
			if result.MatchedRoles[role] == "admin" {
				next(w, r)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		// TODO: Obfuscate results
		json.NewEncoder(w).Encode(result)
	}
}

// RequireAuth is middleware that requires a valid NATS JWT with at least one required role
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, claims := m.authenticate(r)

		if !result.Authorized {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(result)
			return
		}

		// Add claims and matched roles to context
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		ctx = context.WithValue(ctx, MatchedRolesContextKey, result.MatchedRoles)

		next(w, r.WithContext(ctx))
	}
}

// authenticate performs the authentication check
func (m *AuthMiddleware) authenticate(r *http.Request) (AuthResult, *auth.UserClaims) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return AuthResult{
			Authorized: false,
			Error:      "missing authorization header",
		}, nil
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return AuthResult{
			Authorized: false,
			Error:      "invalid authorization header format",
		}, nil
	}

	tokenString := parts[1]
	claims, err := m.natsAuthService.VerifyJWT(tokenString)

	if err != nil {
		return AuthResult{
			Authorized: false,
			Error:      "invalid or expired token: " + err.Error(),
		}, nil
	}

	// Check if user has any required roles
	matchedRoles := make(map[string]string)
	for _, roleID := range claims.Roles {
		if roleName, ok := m.requiredRoles[roleID]; ok {
			matchedRoles[roleID] = roleName
		}
	}

	if len(matchedRoles) == 0 {
		return AuthResult{
			Authorized: false,
			UserID:     claims.UserID,
			TeamID:     claims.TeamID,
			Error:      "user does not have any required roles",
		}, claims
	}

	return AuthResult{
		Authorized:   true,
		UserID:       claims.UserID,
		TeamID:       claims.TeamID,
		MatchedRoles: matchedRoles,
	}, claims
}

// GetAuthResult returns the auth result for the current request (for test endpoints)
func (m *AuthMiddleware) GetAuthResult(r *http.Request) AuthResult {
	result, _ := m.authenticate(r)
	return result
}

// GetClaimsFromContext retrieves claims from request context
func GetClaimsFromContext(ctx context.Context) *auth.UserClaims {
	if claims, ok := ctx.Value(ClaimsContextKey).(*auth.UserClaims); ok {
		return claims
	}
	return nil
}

// GetMatchedRolesFromContext retrieves matched roles from request context
func GetMatchedRolesFromContext(ctx context.Context) map[string]string {
	if roles, ok := ctx.Value(MatchedRolesContextKey).(map[string]string); ok {
		return roles
	}
	return nil
}
