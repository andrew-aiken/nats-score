package routes

import (
	"fmt"
	"log/slog"
	"net/http"

	"server/internal/handlers"
	"server/internal/middleware"
)

func StartServer(server *http.Server) error {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error(fmt.Sprintf("Failed to start http server: %v", err))
		return fmt.Errorf("start http server: %w", err)
	}

	return nil
}

func SetupRoutes(serverConfig *handlers.Handler, corsMiddleware middleware.CORSMiddleware, authMiddleware middleware.AuthMiddleware) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", serverConfig.Login)
	mux.HandleFunc("/auth/verify", corsMiddleware.Handler(serverConfig.Verify))
	mux.HandleFunc("/auth/callback", serverConfig.Callback)
	mux.HandleFunc("/auth/token", corsMiddleware.Handler(serverConfig.TokenLogin))

	mux.HandleFunc("/api/checks/mutable-fields", corsMiddleware.Handler(authMiddleware.RequireAuth(serverConfig.GetMutableFields)))
	mux.HandleFunc("/api/checks", corsMiddleware.Handler(serverConfig.Checks))
	mux.HandleFunc("/api/settings", corsMiddleware.Handler(authMiddleware.RequireAuth(serverConfig.TeamSettings)))

	mux.HandleFunc("/api/admin/settings", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(serverConfig.GetChecks)))
	mux.HandleFunc("/api/admin/cron/start", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(serverConfig.StartScoringCron)))
	mux.HandleFunc("/api/admin/cron/stop", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(serverConfig.StopScoringCron)))

	return mux
}
