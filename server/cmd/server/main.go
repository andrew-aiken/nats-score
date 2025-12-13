package main

import (
	"log"
	"net/http"

	"server/pkg/config"
	"server/pkg/handlers"
	"server/pkg/middleware"
	"server/pkg/nats"

	"golang.org/x/oauth2"
)

// This is the state key used for security, sent in login, validated in callback.
// For this example we keep it simple and hardcode a string
// but in real apps you must provide a proper function that generates a state.
const state = "random"

// Target guild ID to check
const targetGuildID = "833740335997124608"

// Required roles map (role ID -> role name)
var requiredRoles = map[string]string{
	"833740336010100793":  "Admin",
	"1065039522443833364": "Team1",
}

// Predefined access tokens map (token -> config)
// These tokens bypass Discord OAuth and grant specific roles
var accessTokens = map[string]*handlers.TokenConfig{
	"admin": {
		Role:     "833740336010100793", // Admin role
		Username: "Admin Token User",
	},
	"team1": {
		Role:     "1065039522443833364", // Team1 role
		Username: "Team1 Token User",
	},
}

func main() {
	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize NATS auth service
	natsAuthService, err := nats.NewNATSAuthService(cfg.AccountSigningSeed, cfg.AccountPublicKey)
	if err != nil {
		log.Fatalf("Failed to initialize NATS auth service: %v", err)
	}

	// Initialize NATS KV client (optional - only if NATS URL is configured)
	var natsKVClient *nats.NATSKVClient
	if cfg.NATSUrl != "" {
		natsKVClient, err = nats.NewNATSKVClient(cfg.NATSUrl, cfg.NATSCredsFile)
		if err != nil {
			log.Printf("Warning: Failed to initialize NATS KV client: %v", err)
			log.Println("Mutable fields endpoint will be unavailable")
		} else {
			log.Println("Connected to NATS KV bucket 'settings'")
			defer natsKVClient.Close()
		}
	}

	// Create OAuth2 config
	oauthConfig := &oauth2.Config{
		RedirectURL:  cfg.RedirectURL,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes: []string{
			"guilds",
			"guilds.members.read",
			"identify",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://discord.com/api/oauth2/authorize",
			TokenURL:  "https://discord.com/api/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	// Create handler
	h := handlers.NewHandler(oauthConfig, natsAuthService, natsKVClient, targetGuildID, requiredRoles, accessTokens, state, cfg.FrontendURL)

	// Create CORS middleware (allow frontend origin)
	corsMiddleware := middleware.NewCORSMiddleware([]string{cfg.FrontendURL, "http://localhost:5173"})

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(natsAuthService, requiredRoles)

	// Register routes with CORS
	http.HandleFunc("/login", h.Login)
	http.HandleFunc("/auth/verify", corsMiddleware.Handler(h.Verify))
	http.HandleFunc("/auth/callback", h.Callback)
	http.HandleFunc("/auth/token", corsMiddleware.Handler(h.TokenLogin))
	http.HandleFunc("/auth/test", corsMiddleware.Handler(authMiddleware.RequireAuth(h.TestAuth)))
	http.HandleFunc("/api/checks/mutable-fields", corsMiddleware.Handler(authMiddleware.RequireAuth(h.GetMutableFields)))
	http.HandleFunc("/api/settings", corsMiddleware.Handler(authMiddleware.RequireAuth(h.TeamSettings)))

	log.Println("Listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
