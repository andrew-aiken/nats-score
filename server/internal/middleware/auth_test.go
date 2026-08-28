package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/internal/auth"
	"server/internal/middleware"

	"github.com/nats-io/nkeys"
)

func newTestAuthService(t *testing.T) *auth.NATSAuthService {
	t.Helper()

	kp, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatalf("failed to create account keypair: %v", err)
	}

	seed, err := kp.Seed()
	if err != nil {
		t.Fatalf("failed to get account seed: %v", err)
	}

	pubKey, err := kp.PublicKey()
	if err != nil {
		t.Fatalf("failed to get account public key: %v", err)
	}

	svc, err := auth.NewNATSAuthService(string(seed), pubKey)
	if err != nil {
		t.Fatalf("failed to create auth service: %v", err)
	}

	return svc
}

func generateToken(t *testing.T, svc *auth.NATSAuthService, team string) string {
	t.Helper()

	creds, err := svc.GenerateCredentials("dummy", team)
	if err != nil {
		t.Fatalf("failed to generate credentials for team %q: %v", team, err)
	}

	return creds.JWT
}

type request struct {
	Method string
}

type headers map[string]string

type testObj struct {
	Name             string
	Request          request
	Headers          headers
	ExpectedCode     int
	MessageSubstring string
	ExpectNextCalled bool
}

// runMiddlewareTest exercises a middleware-wrapping function against a stub handler
func runMiddlewareTest(t *testing.T, tt testObj, mw func(http.HandlerFunc) http.HandlerFunc) {
	t.Helper()

	nextCalled := false
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}

	req, err := http.NewRequest(tt.Request.Method, "/", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range tt.Headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	handler := mw(next)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != tt.ExpectedCode {
		t.Errorf("handler returned wrong status code: got %v want %v", status, tt.ExpectedCode)
	}

	if tt.MessageSubstring != "" && !strings.Contains(rr.Body.String(), tt.MessageSubstring) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.MessageSubstring)
	}

	if nextCalled != tt.ExpectNextCalled {
		t.Errorf("next handler called = %v, want %v", nextCalled, tt.ExpectNextCalled)
	}
}

func TestRequireAuth(t *testing.T) {
	svc := newTestAuthService(t)
	adminToken := generateToken(t, svc, "admin")
	validToken := generateToken(t, svc, "0")
	am := middleware.NewAuthMiddleware(svc)

	tests := []testObj{
		{
			Name:             "MissingHeader",
			Request:          request{Method: "GET"},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `"authorized":false`,
			ExpectNextCalled: false,
		},
		{
			Name:             "BadHeaderFormat",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "NotBearer"},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `invalid authorization header format`,
			ExpectNextCalled: false,
		},
		{
			Name:             "InvalidToken",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "Bearer garbage"},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `invalid or expired token`,
			ExpectNextCalled: false,
		},
		{
			Name:             "Unauthenticated",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJleHAiOjE3ODc5NjY2NjUsImp0aSI6IkhBUENLTk5PNTNZM1YyM0NRVUlVV0VOSUpIU1VSQ0laSTZJWjVTS0YzNVUzWFRWSEJXTUEiLCJpYXQiOjE3ODc4ODAyNjUsImlzcyI6IkFEWjNDSUJCS0pSUEFEWEZNNEJQNUZLVFlLUkZMUzdCWTI2UTJHU0xQM1E2M01BTURQS1lNUkxMIiwibmFtZSI6ImFkbWluIiwic3ViIjoiVUNGUkpaUUlRSEpIUEtMNUdVWDRWVVJaVVZLREpNQUhWN1IyVldMUUhXQUVYRU4zWDJFUUdBNDQiLCJuYXRzIjp7InB1YiI6eyJhbGxvdyI6WyJcdTAwM2UiXX0sInN1YiI6eyJhbGxvdyI6WyJcdTAwM2UiXX0sInN1YnMiOi0xLCJkYXRhIjotMSwicGF5bG9hZCI6LTEsImlzc3Vlcl9hY2NvdW50IjoiQURaM0NJQkJLSlJQQURYRk00QlA1RktUWUtSRkxTN0JZMjZRMkdTTFAzUTYzTUFNRFBLWU1STEwiLCJ0YWdzIjpbInVzZXJfaWQ6ZHVtbXkiLCJ0ZWFtX2lkOmFkbWluIl0sInR5cGUiOiJ1c2VyIiwidmVyc2lvbiI6Mn19.DXJ1u5Vb3kmU9QC2fRGkjaC_ZovKsVggBWV5ghRirEuMvgIBvAgD3qv4AMq7sbdRnUMyM4NJjVRLBw8XY_a5AQ"},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"authorized":false,"error":"invalid or expired token: invalid issuer"}`,
			ExpectNextCalled: false,
		},
		{
			Name:             "ValidToken",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "Bearer " + validToken},
			ExpectedCode:     http.StatusOK,
			ExpectNextCalled: true,
		},
		{
			Name:             "AdminTeam",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "Bearer " + adminToken},
			ExpectedCode:     http.StatusOK,
			ExpectNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			runMiddlewareTest(t, tt, am.RequireAuth)
		})
	}
}

func TestRequireAdminAuth(t *testing.T) {
	svc := newTestAuthService(t)
	adminToken := generateToken(t, svc, "admin")
	teamToken := generateToken(t, svc, "0")
	am := middleware.NewAuthMiddleware(svc)

	tests := []testObj{
		{
			Name:             "MissingHeader",
			Request:          request{Method: "GET"},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `"authorized":false`,
			ExpectNextCalled: false,
		},
		{
			Name:             "Unauthenticated",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJleHAiOjE3ODc5NjY2NjUsImp0aSI6IkhBUENLTk5PNTNZM1YyM0NRVUlVV0VOSUpIU1VSQ0laSTZJWjVTS0YzNVUzWFRWSEJXTUEiLCJpYXQiOjE3ODc4ODAyNjUsImlzcyI6IkFEWjNDSUJCS0pSUEFEWEZNNEJQNUZLVFlLUkZMUzdCWTI2UTJHU0xQM1E2M01BTURQS1lNUkxMIiwibmFtZSI6ImFkbWluIiwic3ViIjoiVUNGUkpaUUlRSEpIUEtMNUdVWDRWVVJaVVZLREpNQUhWN1IyVldMUUhXQUVYRU4zWDJFUUdBNDQiLCJuYXRzIjp7InB1YiI6eyJhbGxvdyI6WyJcdTAwM2UiXX0sInN1YiI6eyJhbGxvdyI6WyJcdTAwM2UiXX0sInN1YnMiOi0xLCJkYXRhIjotMSwicGF5bG9hZCI6LTEsImlzc3Vlcl9hY2NvdW50IjoiQURaM0NJQkJLSlJQQURYRk00QlA1RktUWUtSRkxTN0JZMjZRMkdTTFAzUTYzTUFNRFBLWU1STEwiLCJ0YWdzIjpbInVzZXJfaWQ6ZHVtbXkiLCJ0ZWFtX2lkOmFkbWluIl0sInR5cGUiOiJ1c2VyIiwidmVyc2lvbiI6Mn19.DXJ1u5Vb3kmU9QC2fRGkjaC_ZovKsVggBWV5ghRirEuMvgIBvAgD3qv4AMq7sbdRnUMyM4NJjVRLBw8XY_a5AQ"},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"authorized":false,"error":"invalid or expired token: invalid issuer"}`,
			ExpectNextCalled: false,
		},
		{
			Name:             "NonAdminTeam",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "Bearer " + teamToken},
			ExpectedCode:     http.StatusForbidden,
			MessageSubstring: `"authorized":true`,
			ExpectNextCalled: false,
		},
		{
			Name:             "AdminTeam",
			Request:          request{Method: "GET"},
			Headers:          headers{"Authorization": "Bearer " + adminToken},
			ExpectedCode:     http.StatusOK,
			ExpectNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			runMiddlewareTest(t, tt, am.RequireAdminAuth)
		})
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	t.Run("NoClaims", func(t *testing.T) {
		if claims := middleware.GetClaimsFromContext(context.Background()); claims != nil {
			t.Errorf("expected nil claims, got %v", claims)
		}
	})

	t.Run("WrongType", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), middleware.ClaimsContextKey, "not-a-claims")
		if claims := middleware.GetClaimsFromContext(ctx); claims != nil {
			t.Errorf("expected nil claims, got %v", claims)
		}
	})

	t.Run("WithClaims", func(t *testing.T) {
		want := &auth.UserClaims{UserID: "dummy", TeamID: "0"}
		ctx := context.WithValue(context.Background(), middleware.ClaimsContextKey, want)
		got := middleware.GetClaimsFromContext(ctx)
		if got != want {
			t.Errorf("expected claims %v, got %v", want, got)
		}
	})
}
