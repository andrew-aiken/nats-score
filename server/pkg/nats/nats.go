package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

// Role IDs for permission mapping
const (
	AdminRoleID = "833740336010100793"
	Team1RoleID = "1065039522443833364"
)

// RoleToTeamNumber maps role IDs to team numbers
var RoleToTeamNumber = map[string]string{
	Team1RoleID: "1",
	// Add more team mappings here as needed
}

// GetTeamNumberFromRoles extracts the team number from a list of role IDs
// Returns the team number and true if found, or empty string and false if not found
// Admin role returns "admin" as a special case
func GetTeamNumberFromRoles(roles []string) (string, bool) {
	for _, role := range roles {
		if role == AdminRoleID {
			return "admin", true
		}
		if teamNum, ok := RoleToTeamNumber[role]; ok {
			return teamNum, true
		}
	}
	return "", false
}

// Credentials holds the NATS JWT and seed for authentication
type Credentials struct {
	JWT  string `json:"jwt"`
	Seed string `json:"seed"`
}

// UserClaims represents the extracted claims from a NATS JWT
type UserClaims struct {
	UserID        string
	Username      string
	Roles         []string
	PubAllow      []string
	SubAllow      []string
	ExpiresAt     time.Time
	IssuedAt      time.Time
	IssuerAccount string
}

// NATSAuthService handles NATS JWT operations
type NATSAuthService struct {
	accountSeed   []byte
	accountPubKey string
}

// NewNATSAuthService creates a new NATS auth service
func NewNATSAuthService(accountSeed, accountPubKey string) (*NATSAuthService, error) {
	// Validate the account seed
	_, err := nkeys.FromSeed([]byte(accountSeed))
	if err != nil {
		return nil, fmt.Errorf("invalid account seed: %w", err)
	}

	return &NATSAuthService{
		accountSeed:   []byte(accountSeed),
		accountPubKey: accountPubKey,
	}, nil
}

// GenerateCredentials creates NATS credentials for a user with role-based permissions
func (s *NATSAuthService) GenerateCredentials(userID, username string, roles []string) (*Credentials, error) {
	// Create a new user keypair
	userKP, err := nkeys.CreateUser()
	if err != nil {
		return nil, fmt.Errorf("failed to create user keypair: %w", err)
	}

	userPub, err := userKP.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get user public key: %w", err)
	}

	// Create user claims
	userClaim := jwt.NewUserClaims(userPub)
	userClaim.Name = username
	userClaim.IssuedAt = time.Now().Unix()
	userClaim.Expires = time.Now().Add(24 * time.Hour).Unix()
	userClaim.IssuerAccount = s.accountPubKey

	// Store user metadata in tags
	userClaim.Tags.Add(fmt.Sprintf("user_id:%s", userID))
	for _, role := range roles {
		userClaim.Tags.Add(fmt.Sprintf("role:%s", role))
	}

	// Apply role-based permissions
	s.applyPermissions(userClaim, roles)

	// Sign with account key
	accountKP, err := nkeys.FromSeed(s.accountSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to create account keypair: %w", err)
	}

	signedJWT, err := userClaim.Encode(accountKP)
	if err != nil {
		return nil, fmt.Errorf("failed to encode user JWT: %w", err)
	}

	// Get user seed
	userSeed, err := userKP.Seed()
	if err != nil {
		return nil, fmt.Errorf("failed to get user seed: %w", err)
	}

	return &Credentials{
		JWT:  signedJWT,
		Seed: string(userSeed),
	}, nil
}

// applyPermissions sets NATS pub/sub permissions based on Discord roles
func (s *NATSAuthService) applyPermissions(userClaim *jwt.UserClaims, roles []string) {
	// Check for admin role - full access
	for _, role := range roles {
		if role == AdminRoleID {
			userClaim.Permissions.Pub.Allow.Add(">")
			userClaim.Permissions.Sub.Allow.Add(">")
			return
		}
	}

	// Check for team roles
	for _, role := range roles {
		if role == Team1RoleID {
			userClaim.Permissions.Pub.Allow.Add("_INBOX.>")
			userClaim.Permissions.Pub.Allow.Add("$JS.API.STREAM.NAMES")
			userClaim.Permissions.Pub.Allow.Add("$JS.API.STREAM.INFO.results")
			userClaim.Permissions.Pub.Allow.Add("$JS.API.CONSUMER.CREATE.results.*.results.1.>")
			userClaim.Permissions.Pub.Allow.Add("$JS.API.CONSUMER.MSG.NEXT.results.*")
			userClaim.Permissions.Pub.Allow.Add("$JS.API.CONSUMER.DELETE.results.*")
			userClaim.Permissions.Pub.Allow.Add("$JS.ACK.results.>")

			// userClaim.Permissions.Pub.Allow.Add("$JS.API.CONSUMER.MSG.NEXT.results.1.*")
			// userClaim.Permissions.Pub.Allow.Add("$JS.API.INFO")
			// userClaim.Permissions.Pub.Allow.Add("$JS.API.STREAM.INFO.KV_settings")
			// userClaim.Permissions.Pub.Allow.Add("$JS.API.STREAM.MSG.GET.KV_settings")
			// userClaim.Permissions.Pub.Allow.Add("$JS.API.DIRECT.GET.KV_settings.>")
			// userClaim.Permissions.Pub.Allow.Add("$JS.API.CONSUMER.>")

			// userClaim.Permissions.Pub.Deny.Add(">")

			userClaim.Permissions.Sub.Allow.Add("results.1.>")
			userClaim.Permissions.Sub.Allow.Add("_INBOX.>")
			// userClaim.Permissions.Sub.Allow.Add("$KV.settings.1.settings")
		}
	}

	// Default permissions for all authenticated users
	// userClaim.Permissions.Pub.Allow.Add("foo")
	// userClaim.Permissions.Sub.Allow.Add("foo")
}

// VerifyJWT validates a NATS JWT and returns the user claims
func (s *NATSAuthService) VerifyJWT(jwtString string) (*UserClaims, error) {
	// Decode the JWT
	claim, err := jwt.DecodeUserClaims(jwtString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT: %w", err)
	}

	// Verify the issuer account matches
	if claim.IssuerAccount != s.accountPubKey {
		return nil, fmt.Errorf("invalid issuer account")
	}

	// Validate the claims (checks expiry, etc.)
	vr := jwt.CreateValidationResults()
	claim.Validate(vr)
	if vr.IsBlocking(true) {
		return nil, fmt.Errorf("JWT validation failed: %v", vr.Errors())
	}

	// Check if expired
	if claim.Expires > 0 && time.Now().Unix() > claim.Expires {
		return nil, fmt.Errorf("JWT has expired")
	}

	// Extract user ID and roles from tags
	var userID string
	var roles []string
	for _, tag := range claim.Tags {
		if len(tag) > 8 && tag[:8] == "user_id:" {
			userID = tag[8:]
		}
		if len(tag) > 5 && tag[:5] == "role:" {
			roles = append(roles, tag[5:])
		}
	}

	// Extract permissions
	var pubAllow, subAllow []string
	for _, perm := range claim.Permissions.Pub.Allow {
		pubAllow = append(pubAllow, perm)
	}
	for _, perm := range claim.Permissions.Sub.Allow {
		subAllow = append(subAllow, perm)
	}

	return &UserClaims{
		UserID:        userID,
		Username:      claim.Name,
		Roles:         roles,
		PubAllow:      pubAllow,
		SubAllow:      subAllow,
		ExpiresAt:     time.Unix(claim.Expires, 0),
		IssuedAt:      time.Unix(claim.IssuedAt, 0),
		IssuerAccount: claim.IssuerAccount,
	}, nil
}

// FormatCredentialsFile creates a .creds file content from JWT and seed
func FormatCredentialsFile(creds *Credentials) string {
	return fmt.Sprintf(`-----BEGIN NATS USER JWT-----
%s
------END NATS USER JWT------
-----BEGIN USER NKEY SEED-----
%s
------END USER NKEY SEED------
`, creds.JWT, creds.Seed)
}
