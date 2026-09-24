package static_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andrew-aiken/score/internal/static"
)

func TestHandler(t *testing.T) {
	staticHandler, err := static.Handler()
	if err != nil {
		t.Fatalf("Failed to generate handler: %s", err.Error())
	}

	t.Run("Root", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		staticHandler.ServeHTTP(rr, req)

		if !strings.Contains(rr.Body.String(), "Frontend build not found") {
			t.Errorf("returned unexpected body: got %v", rr.Body.String())
		}
	})

	t.Run("SubDir", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/dne", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		staticHandler.ServeHTTP(rr, req)

		if !strings.Contains(rr.Body.String(), "Frontend build not found") {
			t.Errorf("returned unexpected body: got %v", rr.Body.String())
		}
	})
}
