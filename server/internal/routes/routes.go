package routes

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"server/internal/handlers"
	"server/internal/middleware"
)

func StartServer(server *http.Server) {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error(fmt.Sprintf("Failed to start http server: %v", err))
		os.Exit(1) // Kill since we're in a gorouting, probably a cleaner way but this works
	}
}

func SetupRoutes(serverConfig *handlers.Handler, corsMiddleware middleware.CORSMiddleware, authMiddleware middleware.AuthMiddleware) {
	http.HandleFunc("/login", serverConfig.Login)
	http.HandleFunc("/auth/verify", corsMiddleware.Handler(serverConfig.Verify))
	http.HandleFunc("/auth/callback", serverConfig.Callback)
	http.HandleFunc("/auth/token", corsMiddleware.Handler(serverConfig.TokenLogin))

	http.HandleFunc("/api/checks/mutable-fields", corsMiddleware.Handler(authMiddleware.RequireAuth(serverConfig.GetMutableFields)))
	http.HandleFunc("/api/checks", corsMiddleware.Handler(serverConfig.Checks))
	http.HandleFunc("/api/settings", corsMiddleware.Handler(authMiddleware.RequireAuth(serverConfig.TeamSettings)))

	http.HandleFunc("/api/admin/settings", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(serverConfig.GetChecks)))
	http.HandleFunc("/api/admin/cron/start", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(serverConfig.StartScoringCron)))
	http.HandleFunc("/api/admin/cron/stop", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(serverConfig.StopScoringCron)))
}
