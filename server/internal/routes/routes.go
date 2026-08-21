package routes

import (
	"fmt"
	"log/slog"
	"net/http"

	"server/internal/handlers"
	"server/internal/middleware"
	"server/internal/static"
)

func StartServer(server *http.Server) error {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error(fmt.Sprintf("Failed to start http server: %v", err))
		return fmt.Errorf("start http server: %w", err)
	}

	return nil
}

func SetupRoutes(serverConfig *handlers.Handler, authMiddleware middleware.AuthMiddleware) *http.ServeMux {
	mux := http.NewServeMux()

	staticHandler, err := static.Handler()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to set up static file handler: %v", err))
	} else {
		mux.Handle("/", staticHandler)
	}

	mux.HandleFunc("/auth/verify", serverConfig.Verify)
	mux.HandleFunc("/auth/login", serverConfig.Login)

	mux.HandleFunc("/api/checks/mutable-fields", authMiddleware.RequireAuth(serverConfig.GetMutableFields))
	mux.HandleFunc("/api/checks", serverConfig.Checks)
	mux.HandleFunc("/api/settings", authMiddleware.RequireAuth(serverConfig.TeamSettings))

	mux.HandleFunc("/api/admin/settings", authMiddleware.RequireAdminAuth(serverConfig.GetChecks))
	mux.HandleFunc("/api/admin/cron/start", authMiddleware.RequireAdminAuth(serverConfig.StartScoringCron))
	mux.HandleFunc("/api/admin/cron/stop", authMiddleware.RequireAdminAuth(serverConfig.StopScoringCron))

	return mux
}
