package middleware

import (
	"testing"
	"time"
)

func TestOIDCConfigDefaults(t *testing.T) {
	t.Parallel()
	in := OIDCConfig{IssuerURL: "https://example/realms/X", Audience: "a"}
	out := in.Defaults()

	if out.RefreshInterval != time.Hour {
		t.Errorf("RefreshInterval: want 1h, got %v", out.RefreshInterval)
	}
	if out.RefreshTimeout != 10*time.Second {
		t.Errorf("RefreshTimeout: want 10s, got %v", out.RefreshTimeout)
	}
	if out.ClockSkew != 5*time.Second {
		t.Errorf("ClockSkew: want 5s, got %v", out.ClockSkew)
	}
	// Caller-provided values must not be overwritten.
	if out.IssuerURL != in.IssuerURL || out.Audience != in.Audience {
		t.Errorf("Defaults should not overwrite caller values")
	}
}

func TestOIDCConfigDefaultsKeepsExplicitValues(t *testing.T) {
	t.Parallel()
	in := OIDCConfig{
		IssuerURL:       "https://example/realms/X",
		Audience:        "a",
		RefreshInterval: 5 * time.Minute,
		RefreshTimeout:  3 * time.Second,
		ClockSkew:       1 * time.Second,
	}
	out := in.Defaults()

	if out.RefreshInterval != 5*time.Minute {
		t.Errorf("explicit RefreshInterval was overwritten: %v", out.RefreshInterval)
	}
	if out.RefreshTimeout != 3*time.Second {
		t.Errorf("explicit RefreshTimeout was overwritten: %v", out.RefreshTimeout)
	}
	if out.ClockSkew != 1*time.Second {
		t.Errorf("explicit ClockSkew was overwritten: %v", out.ClockSkew)
	}
}

func TestJWKSURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		issuer  string
		want    string
		wantErr bool
	}{
		{
			name:   "standard keycloak realm",
			issuer: "https://auth.example.org/realms/EEGFaktura",
			want:   "https://auth.example.org/realms/EEGFaktura/protocol/openid-connect/certs",
		},
		{
			name:   "issuer with trailing path",
			issuer: "https://auth.example.org/auth/realms/X",
			want:   "https://auth.example.org/auth/realms/X/protocol/openid-connect/certs",
		},
		{
			name:    "missing issuer",
			issuer:  "",
			wantErr: true,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cfg := OIDCConfig{IssuerURL: c.issuer}
			got, err := cfg.JWKSURL()
			if c.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("JWKSURL: got %q, want %q", got, c.want)
			}
		})
	}
}
