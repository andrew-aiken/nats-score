package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChecksJSON(t *testing.T) {
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
			MessageSubstring: `{}`,
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
			handler := http.HandlerFunc(tt.Handler.ChecksJSON)

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
