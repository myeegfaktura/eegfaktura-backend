package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	jwt "github.com/golang-jwt/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// oidcClaims is the internal Claim shape the v5 parser unmarshals into.
// It bridges jwtv5.Claims (validated by the parser) and our public
// PlatformClaims type defined in middelware.go.
type oidcClaims struct {
	jwtv5.RegisteredClaims
	Tenants      []string `json:"tenant"`
	Username     string   `json:"preferred_username"`
	AccessGroups []string `json:"access_groups"`
}

// toPlatform copies the claims that the rest of the codebase reads
// into the legacy PlatformClaims shape (which still embeds the
// jwt-v3 StandardClaims for callers).
func (c oidcClaims) toPlatform() *PlatformClaims {
	out := &PlatformClaims{
		Tenants:      c.Tenants,
		Username:     c.Username,
		AccessGroups: c.AccessGroups,
	}
	// Mirror the standard claims the existing code paths may inspect.
	out.StandardClaims = jwt.StandardClaims{
		Issuer:  c.Issuer,
		Subject: c.Subject,
		Id:      c.ID,
	}
	if c.ExpiresAt != nil {
		out.StandardClaims.ExpiresAt = c.ExpiresAt.Unix()
	}
	if c.IssuedAt != nil {
		out.StandardClaims.IssuedAt = c.IssuedAt.Unix()
	}
	if c.NotBefore != nil {
		out.StandardClaims.NotBefore = c.NotBefore.Unix()
	}
	if len(c.Audience) > 0 {
		out.StandardClaims.Audience = c.Audience[0]
	}
	return out
}

// TokenVerifier validates JWT access tokens against a configured OIDC
// issuer. It owns the JWKS cache and is safe for concurrent use.
type TokenVerifier struct {
	keyfunc  keyfunc.Keyfunc
	issuer   string
	audience string
}

// NewTokenVerifier wires up a TokenVerifier from an OIDCConfig.
// The JWKS cache starts populating in the background as soon as the
// returned verifier is constructed.
func NewTokenVerifier(ctx context.Context, cfg OIDCConfig) (*TokenVerifier, error) {
	cfg = cfg.Defaults()
	if cfg.IssuerURL == "" {
		return nil, errors.New("token verifier: OIDCConfig.IssuerURL is required")
	}
	if cfg.Audience == "" {
		return nil, errors.New("token verifier: OIDCConfig.Audience is required")
	}
	kf, err := cfg.NewKeyfunc(ctx)
	if err != nil {
		return nil, err
	}
	return &TokenVerifier{
		keyfunc:  kf,
		issuer:   cfg.IssuerURL,
		audience: cfg.Audience,
	}, nil
}

// Verify parses, validates, and returns the platform claims of the
// provided access token. Returns an error if any of: signature
// invalid, expired, wrong issuer, wrong audience, malformed.
//
// Note: the `aud` claim is treated as a single string here. If the
// token carries `aud` as an array, parsing fails — this is intentional
// (see auth-layer-spec.md and ADR-0003 in the platform repo for the
// rationale).
func (v *TokenVerifier) Verify(tokenStr string) (*PlatformClaims, error) {
	if v == nil {
		return nil, errors.New("token verifier: nil receiver")
	}

	claims := &oidcClaims{}
	parser := jwtv5.NewParser(
		jwtv5.WithIssuer(v.issuer),
		jwtv5.WithAudience(v.audience),
		jwtv5.WithExpirationRequired(),
		jwtv5.WithValidMethods([]string{jwtv5.SigningMethodRS256.Alg()}),
	)
	tok, err := parser.ParseWithClaims(tokenStr, claims, v.keyfunc.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("token verifier: parse: %w", err)
	}
	if !tok.Valid {
		return nil, errors.New("token verifier: token not valid")
	}
	return claims.toPlatform(), nil
}
