package middleware

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	BASIC_SCHEMA  string = "Basic "
	BEARER_SCHEMA string = "Bearer "
)

// JWTMiddleware returns an http middleware that validates the
// Authorization: Bearer ... access token using the supplied
// TokenVerifier (OIDC + JWKS). On success the handler is invoked with
// the parsed PlatformClaims and the canonical tenant string.
//
// Tenant identification: the request must carry either an
// "X-Tenant" or (legacy) "tenant" header. The value must be present
// in claims.tenant (case-insensitive); otherwise the request is
// rejected with 403.
func JWTMiddleware(verifier *TokenVerifier) func(JWTHandlerFunc) http.HandlerFunc {
	if verifier == nil {
		log.Fatal("JWTMiddleware: nil TokenVerifier")
	}
	return func(handler JWTHandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if len(authz) == 0 {
				log.Printf("No Authorization header in request")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !strings.HasPrefix(authz, BEARER_SCHEMA) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			tokenStr := authz[len(BEARER_SCHEMA):]

			claims, err := verifier.Verify(tokenStr)
			if err != nil {
				log.Printf("Token verification failed: %s", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			tenant := tenantHeader(r)
			if !contains(claims.Tenants, tenant) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			handler(w, r, claims, strings.ToUpper(tenant))
		}
	}
}

// tenantHeader returns the tenant identifier from the request, preferring
// the canonical X-Tenant header and falling back to the legacy lowercase
// "tenant" header for backward compatibility with existing clients.
func tenantHeader(r *http.Request) string {
	if v := r.Header.Get("X-Tenant"); v != "" {
		return v
	}
	return r.Header.Get("tenant")
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if strings.EqualFold(v, str) {
			return true
		}
	}
	return false
}
