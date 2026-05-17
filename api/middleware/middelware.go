package middleware

import (
	"net/http"

	"github.com/golang-jwt/jwt"
)

// PlatformClaims is the canonical claim shape consumed by handlers
// downstream of the middleware. It is populated either by the OIDC
// TokenVerifier (see token_verifier.go) or by callers that already
// hold a parsed token.
type PlatformClaims struct {
	Tenants      []string `json:"tenant"`
	Username     string   `json:"preferred_username"`
	AccessGroups []string `json:"access_groups"`
	jwt.StandardClaims
}

// JWTHandlerFunc is the handler signature wired by JWTMiddleware.
// It receives the verified PlatformClaims and the canonical tenant.
type JWTHandlerFunc func(http.ResponseWriter, *http.Request, *PlatformClaims, string)

// JWTWrapperFunc is the wrapper a router (e.g. controllers) consumes
// when registering authenticated routes.
type JWTWrapperFunc func(handlerFunc JWTHandlerFunc) http.HandlerFunc
