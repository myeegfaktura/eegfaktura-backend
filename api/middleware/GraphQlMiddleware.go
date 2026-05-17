package middleware

import (
	"context"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

var tenantCtxKey = &contextKey{"tenant"}

type contextKey struct {
	name string
}

// GQLMiddleware wires the OIDC TokenVerifier into the GraphQL endpoint
// pipeline. It validates the bearer token, checks the tenant header,
// and propagates the canonical tenant string through the request
// context (ForContextTenant).
func GQLMiddleware(verifier *TokenVerifier) func(http.Handler) http.Handler {
	if verifier == nil {
		log.Fatal("GQLMiddleware: nil TokenVerifier")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			ctx := context.WithValue(r.Context(), tenantCtxKey, strings.ToUpper(tenant))
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func ForContextTenant(ctx context.Context) string {
	raw, _ := ctx.Value(tenantCtxKey).(string)
	return raw
}
