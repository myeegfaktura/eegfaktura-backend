package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// jwksFromRSA marshals a single RSA public key into a minimal JWKS
// document suitable for a Keycloak-shaped JWKS endpoint.
func jwksFromRSA(kid string, pub *rsa.PublicKey) []byte {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	doc := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kid,
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	}
	b, _ := json.Marshal(doc)
	return b
}

type e2eFixture struct {
	priv     *rsa.PrivateKey
	srv      *httptest.Server
	issuer   string
	audience string
	kid      string
	verifier *TokenVerifier
}

func newE2EFixture(t *testing.T) *e2eFixture {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	kid := "test-kid-1"
	jwks := jwksFromRSA(kid, &priv.PublicKey)

	mux := http.NewServeMux()
	mux.HandleFunc("/realms/test/protocol/openid-connect/certs",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/jwk-set+json")
			_, _ = w.Write(jwks)
		})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	issuer := srv.URL + "/realms/test"
	audience := "test-audience"
	verifier, err := NewTokenVerifier(context.Background(), OIDCConfig{
		IssuerURL: issuer,
		Audience:  audience,
	})
	if err != nil {
		t.Fatalf("NewTokenVerifier: %v", err)
	}

	return &e2eFixture{
		priv:     priv,
		srv:      srv,
		issuer:   issuer,
		audience: audience,
		kid:      kid,
		verifier: verifier,
	}
}

// signToken signs a JWT with RS256 and the fixture's private key,
// applying overrides on top of a sane default claim set.
func (f *e2eFixture) signToken(t *testing.T, override map[string]any) string {
	t.Helper()
	claims := jwtv5.MapClaims{
		"iss":                f.issuer,
		"aud":                f.audience,
		"exp":                time.Now().Add(time.Hour).Unix(),
		"iat":                time.Now().Add(-1 * time.Minute).Unix(),
		"sub":                "user-uuid-1",
		"preferred_username": "alice",
		"tenant":             []string{"TE100200", "RC100130"},
		"access_groups":      []string{"/EEG_USER"},
	}
	for k, v := range override {
		if v == nil {
			delete(claims, k)
			continue
		}
		claims[k] = v
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
	tok.Header["kid"] = f.kid
	signed, err := tok.SignedString(f.priv)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return signed
}

func TestVerifyValidToken(t *testing.T) {
	t.Parallel()
	f := newE2EFixture(t)
	tok := f.signToken(t, nil)

	claims, err := f.verifier.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Username != "alice" {
		t.Errorf("Username: got %q want %q", claims.Username, "alice")
	}
	if len(claims.Tenants) != 2 || claims.Tenants[0] != "TE100200" {
		t.Errorf("Tenants: %v", claims.Tenants)
	}
	if len(claims.AccessGroups) != 1 || claims.AccessGroups[0] != "/EEG_USER" {
		t.Errorf("AccessGroups: %v", claims.AccessGroups)
	}
}

func TestVerifyRejectsWrongAudience(t *testing.T) {
	t.Parallel()
	f := newE2EFixture(t)
	tok := f.signToken(t, map[string]any{"aud": "other-audience"})

	_, err := f.verifier.Verify(tok)
	if err == nil {
		t.Fatal("expected error for wrong audience, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "audience") {
		t.Errorf("error should mention audience: %v", err)
	}
}

func TestVerifyRejectsWrongIssuer(t *testing.T) {
	t.Parallel()
	f := newE2EFixture(t)
	tok := f.signToken(t, map[string]any{"iss": "https://attacker.example/realms/test"})

	_, err := f.verifier.Verify(tok)
	if err == nil {
		t.Fatal("expected error for wrong issuer, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "iss") {
		t.Errorf("error should mention issuer: %v", err)
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	t.Parallel()
	f := newE2EFixture(t)
	tok := f.signToken(t, map[string]any{
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})

	_, err := f.verifier.Verify(tok)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "expired") {
		t.Errorf("error should mention expiry: %v", err)
	}
}

func TestVerifyRejectsMissingExp(t *testing.T) {
	t.Parallel()
	f := newE2EFixture(t)
	tok := f.signToken(t, map[string]any{"exp": nil})

	_, err := f.verifier.Verify(tok)
	if err == nil {
		t.Fatal("expected error for missing exp, got nil")
	}
}

func TestVerifyRejectsHS256(t *testing.T) {
	t.Parallel()
	f := newE2EFixture(t)
	// Build an HS256 token with the same claim shape; it must be
	// rejected by WithValidMethods([RS256]).
	claims := jwtv5.MapClaims{
		"iss": f.issuer,
		"aud": f.audience,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	tok.Header["kid"] = f.kid
	signed, err := tok.SignedString([]byte("symmetric-secret"))
	if err != nil {
		t.Fatalf("HS256 SignedString: %v", err)
	}

	_, err = f.verifier.Verify(signed)
	if err == nil {
		t.Fatal("expected error for HS256 token, got nil")
	}
}

func TestVerifyRejectsUnknownKid(t *testing.T) {
	t.Parallel()
	f := newE2EFixture(t)
	claims := jwtv5.MapClaims{
		"iss": f.issuer,
		"aud": f.audience,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
	tok.Header["kid"] = "unknown-kid-not-in-jwks"
	signed, err := tok.SignedString(f.priv)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = f.verifier.Verify(signed)
	if err == nil {
		t.Fatal("expected error for unknown kid, got nil")
	}
}
