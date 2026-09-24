package routes_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/andrew-aiken/score/internal/auth"
	"github.com/andrew-aiken/score/internal/handlers"
	"github.com/andrew-aiken/score/internal/middleware"
	"github.com/andrew-aiken/score/internal/routes"
)

func TestStartServer(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		server := &http.Server{
			Addr:         "0.0.0.0:1337",
			ReadTimeout:  1 * time.Second,
			WriteTimeout: 1 * time.Second,
			IdleTimeout:  3 * time.Second,
		}

		go func() {
			err := routes.StartServer(server)
			if err != nil {
				t.Error(err)
			}
		}()

		time.Sleep(time.Millisecond)

		err := server.Close()
		if err != nil {
			t.Fatalf("Error stopping server: %s", err.Error())
		}
	})

	t.Run("Error", func(t *testing.T) {
		server := &http.Server{
			Addr:         "not-addr",
			ReadTimeout:  1 * time.Second,
			WriteTimeout: 1 * time.Second,
			IdleTimeout:  3 * time.Second,
		}

		go func() {
			err := routes.StartServer(server)
			if err != nil {
				if !strings.Contains(err.Error(), "start http server: listen tcp: address not-addr: missing port in address") {
					t.Errorf("returned unexpected error: got %v", err.Error())
				}
			}
		}()

		time.Sleep(time.Millisecond)

		err := server.Close()
		if err != nil {
			t.Fatalf("Error stopping server: %s", err.Error())
		}
	})
}

func TestSetupRoutes(t *testing.T) {
	natsAuthService := &auth.NATSAuthService{}
	h := &handlers.Handler{}

	authMiddleware := middleware.NewAuthMiddleware(natsAuthService)

	mux := routes.SetupRoutes(h, *authMiddleware)

	server := &http.Server{
		Addr:         "0.0.0.0:1337",
		Handler:      mux,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  3 * time.Second,
	}

	go func() {
		err := routes.StartServer(server)
		if err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(time.Millisecond)

	err := server.Close()
	if err != nil {
		t.Fatalf("Error stopping server: %s", err.Error())
	}
}
