package auth

import (
	"errors"
	"strings"
)

// ParseBearerToken extracts the token from an "Authorization: Bearer <token>" header value
func ParseBearerToken(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}
	return parts[1], nil
}
