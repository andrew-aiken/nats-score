package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andrew-aiken/score/internal/auth"
	"github.com/andrew-aiken/score/internal/handlers"
	"github.com/andrew-aiken/score/internal/middleware"

	natsserverserver "github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

type request struct {
	Method string
}

type headers map[string]string

type testObj struct {
	Name             string
	Handler          handlers.Handler
	Request          request
	Headers          headers
	Body             string
	MessageSubstring string
	ExpectedCode     int
	UserClaim        *auth.UserClaims
}

func TestVerify(t *testing.T) {
	tests := []testObj{
		{
			Name: "NoAuthorizationHeader",
			Request: request{
				Method: "GET",
			},
			Headers:          headers{},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"valid":false}`,
		},
		{
			Name: "NotBearerFormat",
			Request: request{
				Method: "GET",
			},
			Headers: headers{
				"Authorization": "wrong format",
			},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"valid":false}`,
		},
		{
			Name: "NotValidJWT",
			Request: request{
				Method: "GET",
			},
			Headers: headers{
				"Authorization": "Bearer XYZ",
			},
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"valid":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req, err := http.NewRequest(tt.Request.Method, "/", strings.NewReader(tt.Body))
			if err != nil {
				t.Fatal(err)
			}

			for k, v := range tt.Headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(tt.Handler.Verify)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.ExpectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.ExpectedCode)
			}

			if tt.MessageSubstring != "" && !strings.Contains(rr.Body.String(), tt.MessageSubstring) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.MessageSubstring)
			}
		})
	}
}

func TestGetMutableFields(t *testing.T) {
	natsHandler, nc, s := setupNatsHandler(t)
	defer s.Shutdown()
	defer nc.Close()

	tests := []testObj{
		{
			Name: "FailMutableFields",
			Request: request{
				Method: "GET",
			},
			Handler:          natsHandler,
			ExpectedCode:     200,
			MessageSubstring: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req, err := http.NewRequest(tt.Request.Method, "/", strings.NewReader(tt.Body))
			if err != nil {
				t.Fatal(err)
			}

			for k, v := range tt.Headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(tt.Handler.GetMutableFields)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.ExpectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.ExpectedCode)
			}

			if tt.MessageSubstring != "" && !strings.Contains(rr.Body.String(), tt.MessageSubstring) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.MessageSubstring)
			}
		})
	}
}

func TestChecks(t *testing.T) {
	natsHandler, nc, s := setupNatsHandler(t)
	defer s.Shutdown()
	defer nc.Close()

	tests := []testObj{
		{
			Name: "Test",
			Request: request{
				Method: "GET",
			},
			Handler:          natsHandler,
			ExpectedCode:     http.StatusOK,
			MessageSubstring: `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req, err := http.NewRequest(tt.Request.Method, "/", strings.NewReader(tt.Body))
			if err != nil {
				t.Fatal(err)
			}

			for k, v := range tt.Headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(tt.Handler.Checks)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.ExpectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.ExpectedCode)
			}

			if tt.MessageSubstring != "" && !strings.Contains(rr.Body.String(), tt.MessageSubstring) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.MessageSubstring)
			}
		})
	}
}

func TestTeamSettings(t *testing.T) {
	natsHandler, nc, s := setupNatsHandler(t)
	defer s.Shutdown()
	defer nc.Close()

	userClaim := &auth.UserClaims{
		UserID:    "foo",
		TeamID:    "0",
		PubAllow:  []string{},
		SubAllow:  []string{},
		ExpiresAt: time.Now().Add(10 * time.Minute),
		IssuedAt:  time.Now().Add(-10 * time.Minute),
	}

	bodyPayload, err := json.Marshal(map[string]map[string]string{
		"noop": {
			"key": "value",
		},
	})
	if err != nil {
		t.Fatalf("Failed to marshal json body: %s", err.Error())
	}

	tests := []testObj{
		{
			Name: "NotAuthorized",
			Request: request{
				Method: "GET",
			},
			Handler:          natsHandler,
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"error":"Unauthorized"}`,
		},
		{
			Name: "WrongMethod",
			Request: request{
				Method: "POST",
			},
			UserClaim:        userClaim,
			Handler:          natsHandler,
			ExpectedCode:     http.StatusMethodNotAllowed,
			MessageSubstring: `{"error":"Method not allowed"}`,
		},
		{
			Name: "Get",
			Request: request{
				Method: "GET",
			},
			UserClaim:        userClaim,
			Handler:          natsHandler,
			ExpectedCode:     http.StatusOK,
			MessageSubstring: `{}`,
		},
		{
			Name: "PutNoBody",
			Request: request{
				Method: "PUT",
			},
			UserClaim:        userClaim,
			Handler:          natsHandler,
			ExpectedCode:     http.StatusBadRequest,
			MessageSubstring: `{"error":"Invalid request body"}`,
		},
		{
			Name: "Put",
			Request: request{
				Method: "PUT",
			},
			Body:             string(bodyPayload),
			UserClaim:        userClaim,
			Handler:          natsHandler,
			ExpectedCode:     http.StatusOK,
			MessageSubstring: `{"success":true,"team":"0"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req, err := http.NewRequest(tt.Request.Method, "/", strings.NewReader(tt.Body))
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.WithValue(req.Context(), middleware.ClaimsContextKey, tt.UserClaim)
			req = req.WithContext(ctx)

			for k, v := range tt.Headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(tt.Handler.TeamSettings)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.ExpectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.ExpectedCode)
			}

			if tt.MessageSubstring != "" && !strings.Contains(rr.Body.String(), tt.MessageSubstring) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.MessageSubstring)
			}
		})
	}
}

func setupNatsHandler(t *testing.T) (handlers.Handler, *nats.Conn, *natsserverserver.Server) {
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)

	natsServerAddress := server.Addr().String()

	nc, err := nats.Connect(natsServerAddress)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	settingsBucket, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:       "settings",
		Description:  "Check & configuration storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		t.Fatal(err)
	}

	kv, err := js.KeyValue(settingsBucket.Bucket())
	if err != nil {
		t.Fatalf("Failed to connect to key value: %v", err)
	}

	usersBucket, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:       "users",
		Description:  "Username/password login account storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		t.Fatalf("Failed to connect to user key value: %v", err)
	}

	userKV, err := js.KeyValue(usersBucket.Bucket())
	if err != nil {
		t.Fatalf("Failed to connect to key value: %v", err)
	}

	userPassword, err := auth.HashPassword("dummyPasswd")
	if err != nil {
		t.Fatalf("Failed hash user password: %v", err)
	}

	userPayload, err := json.Marshal(auth.User{
		Username:     "foo",
		Team:         "0",
		PasswordHash: userPassword,
	})
	if err != nil {
		t.Fatalf("Failed marshal user payload: %v", err)
	}

	_, err = userKV.Put("user.foo", userPayload)
	if err != nil {
		t.Fatalf("Failed write test user data: %v", err)
	}

	badUserPayload, err := json.Marshal(`{`)
	if err != nil {
		t.Fatalf("Failed marshal user payload: %v", err)
	}

	_, err = userKV.Put("user.bad", badUserPayload)
	if err != nil {
		t.Fatalf("Failed write test user data: %v", err)
	}

	return handlers.Handler{
		NatsKVClient:      kv,
		NatsUsersKVClient: userKV,
		NatsAuthService:   &auth.NATSAuthService{},
	}, nc, server
}

func TestRequireMethod(t *testing.T) {
	tests := []struct {
		Name           string
		AllowedMethod  string
		RequestMethod  string
		ExpectedCode   int
		ExpectNextCall bool
	}{
		{
			Name:          "WrongMethod",
			AllowedMethod: http.MethodGet,
			RequestMethod: http.MethodPost,
			ExpectedCode:  http.StatusMethodNotAllowed,
		},
		{
			Name:           "MatchingMethod",
			AllowedMethod:  http.MethodGet,
			RequestMethod:  http.MethodGet,
			ExpectedCode:   http.StatusOK,
			ExpectNextCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			nextCalled := false
			next := func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}

			req, err := http.NewRequest(tt.RequestMethod, "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handlers.RequireMethod(tt.AllowedMethod, next).ServeHTTP(rr, req)

			if status := rr.Code; status != tt.ExpectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.ExpectedCode)
			}

			if nextCalled != tt.ExpectNextCall {
				t.Errorf("next called = %v, want %v", nextCalled, tt.ExpectNextCall)
			}

			if !tt.ExpectNextCall {
				want := `{"error":"Method not allowed"}`
				if !strings.Contains(rr.Body.String(), want) {
					t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), want)
				}
			}
		})
	}
}
