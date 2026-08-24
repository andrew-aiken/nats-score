package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/internal/handlers"

	"github.com/go-co-op/gocron/v2"
)

func TestStartScoringCron(t *testing.T) {
	s, err := gocron.NewScheduler()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer func() {
		err := s.Shutdown()
		if err != nil {
			t.Fatalf("Failed to stop cron scheduler: %s", err.Error())
		}
	}()

	cronHandler := handlers.Handler{
		CronScheduler: s,
	}

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
			Name: "NoCronScheduler",
			Request: request{
				Method: "PUT",
			},
			ExpectedCode:     http.StatusServiceUnavailable,
			MessageSubstring: `{"error":"cron scheduler not initialized"}`,
		},
		{
			Name: "StartCron",
			Request: request{
				Method: "PUT",
			},
			Handler:          cronHandler,
			ExpectedCode:     http.StatusOK,
			MessageSubstring: "Started Scoring CronJob",
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
			handler := http.HandlerFunc(tt.Handler.StartScoringCron)

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

func TestStopScoringCron(t *testing.T) {
	s, err := gocron.NewScheduler()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer func() {
		err := s.Shutdown()
		if err != nil {
			t.Fatalf("Failed to stop cron scheduler: %s", err.Error())
		}
	}()

	cronHandler := handlers.Handler{
		CronScheduler: s,
	}

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
			Name: "NoCronScheduler",
			Request: request{
				Method: "PUT",
			},
			ExpectedCode:     http.StatusServiceUnavailable,
			MessageSubstring: `{"error":"cron scheduler not initialized"}`,
		},
		{
			Name: "StopCron",
			Request: request{
				Method: "PUT",
			},
			Handler:          cronHandler,
			ExpectedCode:     http.StatusOK,
			MessageSubstring: "Stopped Scoring CronJob",
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
			handler := http.HandlerFunc(tt.Handler.StopScoringCron)

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
