package middleware

import (
	"context"
	"strings"
	"testing"
)

func TestNewTokenVerifierValidates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cfg     OIDCConfig
		wantErr string
	}{
		{
			name:    "missing issuer",
			cfg:     OIDCConfig{Audience: "a"},
			wantErr: "IssuerURL is required",
		},
		{
			name:    "missing audience",
			cfg:     OIDCConfig{IssuerURL: "https://x/realms/X"},
			wantErr: "Audience is required",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewTokenVerifier(context.Background(), c.cfg)
			if err == nil {
				t.Fatalf("want error, got nil")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %q, must contain %q", err, c.wantErr)
			}
		})
	}
}

func TestVerifyNilReceiver(t *testing.T) {
	t.Parallel()
	var v *TokenVerifier
	_, err := v.Verify("anything")
	if err == nil {
		t.Fatal("want error on nil receiver")
	}
}
