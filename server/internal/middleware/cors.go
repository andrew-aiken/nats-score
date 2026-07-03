package middleware

import "net/http"

// CORSMiddleware handles CORS headers for cross-origin requests
type CORSMiddleware struct {
	allowedOrigins []string
}

// NewCORSMiddleware creates a new CORS middleware with allowed origins
func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
	}
}

// Handler wraps an http.HandlerFunc with CORS headers
func (c *CORSMiddleware) Handler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range c.allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// Wrap wraps an existing http.Handler with CORS headers
func (c *CORSMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range c.allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
