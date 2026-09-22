package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogin(t *testing.T) {
	natsHandler, nc, s := setupNatsHandler(t)
	defer s.Shutdown()
	defer nc.Close()

	tests := []testObj{
		{
			Name: "BadBody",
			Request: request{
				Method: "POST",
			},
			Body:             `}`,
			ExpectedCode:     http.StatusBadRequest,
			MessageSubstring: `{"error":"Invalid request body"}`,
		},
		{
			Name: "MissingUser",
			Request: request{
				Method: "POST",
			},
			Handler:          natsHandler,
			Body:             `{"username":"bad","password":"bar"}`,
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"error":"Invalid username or password"}`,
		},
		{
			Name: "ValidUserInvalidPassword",
			Request: request{
				Method: "POST",
			},
			Handler:          natsHandler,
			Body:             `{"username":"foo","password":"bar"}`,
			ExpectedCode:     http.StatusUnauthorized,
			MessageSubstring: `{"error":"Invalid username or password"}`,
		},
		{
			Name: "FailedAuthNATS",
			Request: request{
				Method: "POST",
			},
			Handler:          natsHandler,
			Body:             `{"username":"foo","password":"dummyPasswd"}`,
			ExpectedCode:     http.StatusInternalServerError,
			MessageSubstring: `{"error":"Failed to generate credentials"}`,
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
			handler := http.HandlerFunc(tt.Handler.Login)

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
