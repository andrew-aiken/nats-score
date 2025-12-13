package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// readKeyFile reads a key/seed file and returns the trimmed contents
func readKeyFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return []byte(strings.TrimSpace(string(data))), nil
}

// formatCredentials creates a .creds file content from JWT and seed
func formatCredentials(userJWT string, userSeed string) string {
	return fmt.Sprintf(`-----BEGIN NATS USER JWT-----
%s
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
%s
------END USER NKEY SEED------
`, userJWT, userSeed)
}

// generateUserJWT creates a user JWT signed by the account
func generateUserJWT(userPub string, accountSeed []byte, accountPub string) (string, error) {
	userClaim := jwt.NewUserClaims(userPub)
	userClaim.Name = "dynamicuser"
	userClaim.IssuedAt = time.Now().Unix()

	userClaim.Permissions.Pub.Allow.Add("results.1.>")

	userClaim.Permissions.Pub.Allow.Add("foo")
	userClaim.Permissions.Sub.Allow.Add("foo")

	userClaim.Expires = time.Now().Add(24 * time.Hour).Unix()
	userClaim.IssuerAccount = accountPub

	accountKP, err := nkeys.FromSeed(accountSeed)
	if err != nil {
		return "", fmt.Errorf("failed to create account keypair from seed: %w", err)
	}

	signedJWT, err := userClaim.Encode(accountKP)
	if err != nil {
		return "", fmt.Errorf("failed to encode user JWT: %w", err)
	}

	return signedJWT, nil
}

func main() {
	// Command line flags
	// operatorSeedFile := flag.String("operator-seed", "", "Path to operator seed file (SO... key)")
	accountSeedFile := flag.String("account-seed", "", "Path to account seed file (SA... key)")
	userCredsFile := flag.String("user-creds", "", "Path to user credentials file (.creds)")
	outputCredsFile := flag.String("output-creds", "", "Path to write generated credentials file (.creds)")
	// userSeedFile := flag.String("user-seed", "", "Path to user seed file (SU... key)")
	// userJWTFile := flag.String("user-jwt", "", "Path to user JWT file")
	natsURL := flag.String("nats", "nats://localhost:4222", "NATS server URL")
	generateOnly := flag.Bool("generate-only", false, "Only generate JWT, don't connect")

	flag.Parse()

	var nc *nats.Conn
	var err error

	switch {
	// Option 1: Use existing .creds file (simplest)
	case *userCredsFile != "":
		fmt.Printf("Using credentials file: %s\n", *userCredsFile)
		nc, err = nats.Connect(*natsURL, nats.UserCredentials(*userCredsFile))
		if err != nil {
			log.Fatalf("Failed to connect with creds file: %v", err)
		}

	// Option 3: Generate user JWT from account seed (dynamic user creation)
	case *accountSeedFile != "":
		fmt.Printf("Generating user JWT using account seed: %s\n", *accountSeedFile)
		accountSeed, err := readKeyFile(*accountSeedFile)
		if err != nil {
			log.Fatal(err)
		}

		accountPub := "ACWH6VIQI2B5UU7NCH6ZQKZVARKRFSTDKKYGP6IQBCIANL3YMWO765JK"

		// Create a new user keypair
		userKP, err := nkeys.CreateUser()
		if err != nil {
			log.Fatal(err)
		}
		userPub, err := userKP.PublicKey()
		if err != nil {
			log.Fatal(err)
		}

		// Generate the user JWT
		userJWT, err := generateUserJWT(userPub, accountSeed, accountPub)
		if err != nil {
			log.Fatal(err)
		}

		// Get user seed for credentials
		userSeed, err := userKP.Seed()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Account Public Key: %s\n", accountPub)
		fmt.Printf("User Public Key: %s\n", userPub)

		// Write credentials file if requested
		if *outputCredsFile != "" {
			credsContent := formatCredentials(userJWT, string(userSeed))
			err := os.WriteFile(*outputCredsFile, []byte(credsContent), 0600)
			if err != nil {
				log.Fatalf("Failed to write credentials file: %v", err)
			}
			fmt.Printf("Credentials written to: %s\n", *outputCredsFile)
		} else {
			// Print to stdout if not writing to file
			fmt.Printf("User JWT: %s\n", userJWT)
			fmt.Printf("User Seed: %s\n", string(userSeed))
		}

		if *generateOnly {
			return
		}

		nc, err = nats.Connect(*natsURL,
			nats.UserJWT(
				func() (string, error) { return userJWT, nil },
				func(nonce []byte) ([]byte, error) { return userKP.Sign(nonce) },
			),
		)
		if err != nil {
			log.Fatalf("Failed to connect with generated user JWT: %v", err)
		}

	default:
		fmt.Println("NATS JWT Sign Tool")
		fmt.Println("==================")
		fmt.Println("\nUsage modes:")
		fmt.Println("\n1. Connect with existing .creds file:")
		fmt.Println("   go run . -user-creds /path/to/user.creds")
		fmt.Println("\n2. Connect with existing JWT + seed:")
		fmt.Println("   go run . -user-jwt /path/to/user.jwt -user-seed /path/to/user.seed")
		fmt.Println("\n3. Generate user JWT from account seed and connect:")
		fmt.Println("   go run . -account-seed /path/to/account.seed")
		fmt.Println("\n4. Generate user credentials file (don't connect):")
		fmt.Println("   go run . -account-seed /path/to/account.seed -output-creds /path/to/user.creds -generate-only")
		fmt.Println("\n5. Generate user JWT only (print to stdout):")
		fmt.Println("   go run . -account-seed /path/to/account.seed -generate-only")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		return
	}

	defer nc.Close()
	fmt.Println("Connected to NATS!")

	// Simple publish/subscribe test
	sub, err := nc.SubscribeSync("foo")
	if err != nil {
		log.Fatal(err)
	}

	err = nc.Publish("foo", []byte("hello from JWT authenticated user"))
	if err != nil {
		log.Fatal(err)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Received message: %s\n", string(msg.Data))
}
