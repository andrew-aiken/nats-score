package auth

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"server/internal/config"
	"server/internal/logging"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

type CliParameters struct {
	ConfigFile string
	Standalone bool
	Teams      uint16
}

func Auth(args CliParameters) error {
	logging.SetupLogging("warn")

	// Adds one to team list so index of 1 team include team 0
	offsetTeams := args.Teams + 1

	// Load configuration
	cfg, err := config.Load(args.ConfigFile)
	if err != nil {
		slog.Error("Failed to load config")
		return err
	}

	for team := range offsetTeams {
		// If its standalone only one set of credentials is needed so it returns one wildcard credential
		if args.Standalone {
			return createAgentCredentials(cfg, "*")
		}

		if err := createAgentCredentials(cfg, strconv.FormatUint(uint64(team), 10)); err != nil {
			return err
		}
	}

	return nil
}

func createAgentCredentials(conf config.Config, streamIndex string) error {
	// Validate the account seed
	_, err := nkeys.FromSeed([]byte(conf.AccountSigningSeed))
	if err != nil {
		return fmt.Errorf("invalid account seed: %w", err)
	}

	// Create a new user keypair
	userKP, err := nkeys.CreateUser()
	if err != nil {
		return fmt.Errorf("failed to create user keypair: %w", err)
	}

	userPub, err := userKP.PublicKey()
	if err != nil {
		return fmt.Errorf("failed to get user public key: %w", err)
	}

	// Create claim permissions
	userClaim := jwt.NewUserClaims(userPub)

	userClaim.Name = fmt.Sprint("agent-" + streamIndex)
	if streamIndex == "*" {
		userClaim.Name = "agent-standalone"
	}

	userClaim.IssuedAt = time.Now().Unix()
	userClaim.Expires = time.Now().Add(24 * time.Hour).Unix()
	userClaim.IssuerAccount = conf.AccountPublicKey

	// Apply permissions
	userClaim.Pub.Allow.Add("results." + streamIndex + ".>")

	userClaim.Pub.Allow.Add("$JS.API.STREAM.INFO.KV_settings")
	userClaim.Pub.Allow.Add("$JS.API.CONSUMER.CREATE.KV_settings.*") // This is a known risk. Allows any agent to view settings of other users
	userClaim.Pub.Allow.Add("$JS.API.CONSUMER.DELETE.KV_settings.*")

	userClaim.Sub.Allow.Add("events.score.>")
	userClaim.Sub.Allow.Add("_INBOX." + streamIndex + ".>")

	// Sign with account key
	accountKP, err := nkeys.FromSeed([]byte(conf.AccountSigningSeed))
	if err != nil {
		return fmt.Errorf("failed to create account keypair: %w", err)
	}

	signedJWT, err := userClaim.Encode(accountKP)
	if err != nil {
		return fmt.Errorf("failed to encode user JWT: %w", err)
	}

	// Get user seed
	userSeed, err := userKP.Seed()
	if err != nil {
		return fmt.Errorf("failed to get user seed: %w", err)
	}

	fmt.Printf(`
-----BEGIN NATS USER JWT-----
%s
------END NATS USER JWT------

-----BEGIN USER NKEY SEED-----
%s
------END USER NKEY SEED------
`, signedJWT, string(userSeed))

	return nil
}
