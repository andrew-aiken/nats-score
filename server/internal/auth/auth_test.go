package auth

import (
	"slices"
	"testing"

	"github.com/nats-io/nkeys"
)

func newTestAuthService(t *testing.T) *NATSAuthService {
	t.Helper()

	kp, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatalf("failed to create account keypair: %v", err)
	}

	seed, err := kp.Seed()
	if err != nil {
		t.Fatalf("failed to get account seed: %v", err)
	}

	pubKey, err := kp.PublicKey()
	if err != nil {
		t.Fatalf("failed to get account public key: %v", err)
	}

	svc, err := NewNATSAuthService(string(seed), pubKey)
	if err != nil {
		t.Fatalf("failed to create auth service: %v", err)
	}

	return svc
}

func generateClaims(t *testing.T, svc *NATSAuthService, team string) *UserClaims {
	t.Helper()

	creds, err := svc.GenerateCredentials("test-user", team)
	if err != nil {
		t.Fatalf("failed to generate credentials for team %q: %v", team, err)
	}

	claims, err := svc.VerifyJWT(creds.JWT)
	if err != nil {
		t.Fatalf("failed to verify generated JWT for team %q: %v", team, err)
	}

	return claims
}

func TestGenerateCredentials_Admin(t *testing.T) {
	svc := newTestAuthService(t)
	claims := generateClaims(t, svc, "admin")

	if !slices.Contains(claims.PubAllow, ">") {
		t.Errorf("expected admin PubAllow to contain \">\", got %v", claims.PubAllow)
	}
	if !slices.Contains(claims.SubAllow, ">") {
		t.Errorf("expected admin SubAllow to contain \">\", got %v", claims.SubAllow)
	}
}

func TestGenerateCredentials_Observer(t *testing.T) {
	svc := newTestAuthService(t)
	claims := generateClaims(t, svc, "observer")

	// Must be the literal "results.>" (not e.g. "results.*.>") since the
	// frontend's JetStream consumer-create call embeds this exact filter
	// subject into its authorization check
	// ($JS.API.CONSUMER.CREATE.results.*.<filter>) — a narrower pattern
	// would be permission-denied even though it covers every real result
	// subject semantically.
	if !slices.Contains(claims.SubAllow, "results.>") {
		t.Errorf("expected observer SubAllow to contain \"results.>\" (read access to all teams), got %v", claims.SubAllow)
	}
	if slices.Contains(claims.PubAllow, ">") {
		t.Errorf("observer PubAllow must not contain \">\" (would grant admin-level publish), got %v", claims.PubAllow)
	}

	// Observer's publish permissions should be the same fixed
	// request-reply/JetStream-API entries a regular team gets, scoped to
	// the observer's own subject filter — never a broader publish grant.
	for _, want := range []string{
		"_INBOX.>",
		"$JS.API.INFO",
		"$JS.API.STREAM.NAMES",
		"$JS.API.STREAM.INFO.results",
		"$JS.API.CONSUMER.CREATE.results.*.results.>",
		"$JS.API.CONSUMER.MSG.NEXT.results.*",
		"$JS.API.CONSUMER.DELETE.results.*",
		"$JS.ACK.results.>",
	} {
		if !slices.Contains(claims.PubAllow, want) {
			t.Errorf("expected observer PubAllow to contain %q, got %v", want, claims.PubAllow)
		}
	}
	if len(claims.PubAllow) != 8 {
		t.Errorf("expected observer PubAllow to have exactly 8 entries (no extra publish rights), got %v", claims.PubAllow)
	}
}

func TestGenerateCredentials_Team(t *testing.T) {
	svc := newTestAuthService(t)
	claims := generateClaims(t, svc, "3")

	if !slices.Contains(claims.SubAllow, "results.3.>") {
		t.Errorf("expected team SubAllow to contain \"results.3.>\", got %v", claims.SubAllow)
	}
	if slices.Contains(claims.SubAllow, "results.>") {
		t.Errorf("expected team SubAllow to NOT contain the observer wildcard \"results.>\", got %v", claims.SubAllow)
	}
	if slices.Contains(claims.PubAllow, ">") {
		t.Errorf("expected team PubAllow to NOT contain \">\", got %v", claims.PubAllow)
	}
}
