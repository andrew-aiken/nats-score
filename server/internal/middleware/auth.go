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
)

// AuthMiddleware holds dependencies for authentication middleware
type AuthMiddleware struct {
	natsAuthService *auth.NATSAuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(natsAuthService *auth.NATSAuthService) *AuthMiddleware {
	return &AuthMiddleware{
		natsAuthService: natsAuthService,
	}
}

// AuthResult represents the authentication result
type AuthResult struct {
	Authorized bool   `json:"authorized"`
	UserID     string `json:"user_id,omitempty"`
	TeamID     string `json:"team_id,omitempty"`
	Error      string `json:"error,omitempty"`
}

// RequireAdminAuth is a middleware that requires a valid JWT belonging to the admin team
func (m *AuthMiddleware) RequireAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, claims := m.authenticate(r)

		if !result.Authorized {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(result)
			return
		}

		if claims.TeamID != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(result)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// RequireAuth is middleware that requires any valid, successfully-verified NATS JWT
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, claims := m.authenticate(r)

		if !result.Authorized {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(result)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
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

	return AuthResult{
		Authorized: true,
		UserID:     claims.UserID,
		TeamID:     claims.TeamID,
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
