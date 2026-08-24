package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/internal/auth"
	"server/internal/handlers"

	natsserver "github.com/nats-io/nats-server/v2/test"
	natsserverserver "github.com/nats-io/nats-server/v2/server"
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
}

func TestVerify(t *testing.T) {
	tests := []testObj{
		{
			Name: "WrongMethod",
			Request: request{
				Method: "POST",
			},
			ExpectedCode:     http.StatusMethodNotAllowed,
			MessageSubstring: `{"error":"Method not allowed"}`,
		},
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
			req, err := http.NewRequest(tt.Request.Method, "/", nil)
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
			Name: "WrongMethod",
			Request: request{
				Method: "POST",
			},
			ExpectedCode:     http.StatusMethodNotAllowed,
			MessageSubstring: `{"error":"Method not allowed"}`,
		},
		{
			Name: "NoNATS",
			Request: request{
				Method: "GET",
			},
			ExpectedCode:     http.StatusServiceUnavailable,
			MessageSubstring: `{"error":"NATS KV not available"}`,
		},
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
			req, err := http.NewRequest(tt.Request.Method, "/", nil)
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
			Name: "WrongMethod",
			Request: request{
				Method: "POST",
			},
			ExpectedCode:     http.StatusMethodNotAllowed,
			MessageSubstring: `{"error":"Method not allowed"}`,
		},
		{
			Name: "NoNATS",
			Request: request{
				Method: "GET",
			},
			ExpectedCode:     http.StatusServiceUnavailable,
			MessageSubstring: `{"error":"NATS KV not available"}`,
		},
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
			req, err := http.NewRequest(tt.Request.Method, "/", nil)
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
		NatsAuthService: &auth.NATSAuthService{},
	}, nc, server
}
