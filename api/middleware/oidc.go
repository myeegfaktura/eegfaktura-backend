package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/MicahParks/keyfunc/v3"
)

// OIDCConfig collects the configuration the middleware needs to validate
// JWT access tokens issued by the configured Keycloak realm.
type OIDCConfig struct {
	// IssuerURL is the realm's issuer URL, e.g.
	// "https://auth.example.org/realms/EEGFaktura".
	IssuerURL string

	// Audience is the expected `aud` claim value (single string — this
	// backend treats `aud` as `string`, see auth-layer-spec.md).
	Audience string

	// RefreshInterval is how often the JWKS cache asks the issuer for
	// updated keys in the background.
	RefreshInterval time.Duration

	// RefreshTimeout is the max time a synchronous JWKS refresh may
	// take when triggered by a cache miss (unknown kid).
	RefreshTimeout time.Duration

	// Clock skew tolerance applied when validating `exp` / `iat`.
	ClockSkew time.Duration
}

// Defaults returns sensible defaults for fields the caller has not set.
func (c OIDCConfig) Defaults() OIDCConfig {
	out := c
	if out.RefreshInterval == 0 {
		out.RefreshInterval = time.Hour
	}
	if out.RefreshTimeout == 0 {
		out.RefreshTimeout = 10 * time.Second
	}
	if out.ClockSkew == 0 {
		out.ClockSkew = 5 * time.Second
	}
	return out
}

// JWKSURL derives the standard Keycloak JWKS endpoint URL from the
// issuer URL by appending "/protocol/openid-connect/certs".
// This is the canonical path for Keycloak realms; if a future deployment
// uses a non-standard layout, switch to OIDC discovery via the
// /.well-known/openid-configuration endpoint.
func (c OIDCConfig) JWKSURL() (string, error) {
	if c.IssuerURL == "" {
		return "", errors.New("oidc: IssuerURL must be set")
	}
	u, err := url.Parse(c.IssuerURL)
	if err != nil {
		return "", fmt.Errorf("oidc: invalid IssuerURL: %w", err)
	}
	// Conservative path-join: rely on Keycloak's documented layout.
	jwks := *u
	jwks.Path = jwks.Path + "/protocol/openid-connect/certs"
	return jwks.String(), nil
}

// NewKeyfunc constructs a JWKS-backed keyfunc with background refresh
// against the configured issuer. The returned keyfunc.Keyfunc can be
// passed to jwt.Parse / jwt.ParseWithClaims.
func (c OIDCConfig) NewKeyfunc(ctx context.Context) (keyfunc.Keyfunc, error) {
	cfg := c.Defaults()
	jwksURL, err := cfg.JWKSURL()
	if err != nil {
		return nil, err
	}
	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("oidc: build keyfunc: %w", err)
	}
	return k, nil
}
