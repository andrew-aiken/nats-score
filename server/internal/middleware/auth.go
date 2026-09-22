package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

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
			err := json.NewEncoder(w).Encode(result)
			if err != nil {
				slog.Error("Failed to encode admin route non-authenticated headers", "error", err.Error())
			}
			return
		}

		if claims.TeamID != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			err := json.NewEncoder(w).Encode(result)
			if err != nil {
				slog.Error("Failed to encode admin route non-authorized headers", "error", err.Error())
			}
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
			err := json.NewEncoder(w).Encode(result)
			if err != nil {
				slog.Error("Failed to encode authenticated route headers", "error", err.Error())
			}
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

	tokenString, err := auth.ParseBearerToken(authHeader)
	if err != nil {
		return AuthResult{
			Authorized: false,
			Error:      err.Error(),
		}, nil
	}

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

// GetClaimsFromContext retrieves claims from request context
func GetClaimsFromContext(ctx context.Context) *auth.UserClaims {
	if claims, ok := ctx.Value(ClaimsContextKey).(*auth.UserClaims); ok {
		return claims
	}
	return nil
}
