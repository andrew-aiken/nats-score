package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func Auth(standalone bool, teams int) error {
	// Adds one to team list so index of 1 team include team 0
	offsetTeams := teams + 1

	for team := range offsetTeams {
		if standalone {
			return createAgentCredentials("*")
		}

		if err := createAgentCredentials(strconv.Itoa(team)); err != nil {
			return err
		}
	}

	return nil
}

func createAgentCredentials(streamIndex string) error {
	accountPubKey := "ACAZUY774GKV27BNGEAPA6RFO2OMMLVLRG5TTZLAAJZDR5ZRORV6QCJ7"
	accountSeed := "SAAJAURE5C35ZJQUUQSSVKYZP5V5L3A6RFPMMNEFRLPDBP2GCMID5TZUXA"

	// Validate the account seed
	_, err := nkeys.FromSeed([]byte(accountSeed))
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
		userClaim.Name = "agent-standalong"
	}

	userClaim.IssuedAt = time.Now().Unix()
	userClaim.Expires = time.Now().Add(24 * time.Hour).Unix()
	userClaim.IssuerAccount = accountPubKey

	// Apply permissions
	userClaim.Permissions.Pub.Allow.Add("results." + streamIndex + ".>")

	userClaim.Permissions.Pub.Allow.Add("$JS.API.STREAM.INFO.KV_settings")
	userClaim.Permissions.Pub.Allow.Add("$JS.API.CONSUMER.CREATE.KV_settings.*") // This is a known risk. Allows any agent to view settings of other users
	userClaim.Permissions.Pub.Allow.Add("$JS.API.CONSUMER.DELETE.KV_settings.*")

	userClaim.Permissions.Sub.Allow.Add("events.score.>")
	userClaim.Permissions.Sub.Allow.Add("_INBOX." + streamIndex + ".>")


	// Sign with account key
	accountKP, err := nkeys.FromSeed([]byte(accountSeed))
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
