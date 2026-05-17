# Auth-Layer Specification

This document specifies the authentication and authorization behavior
expected by the `eegfaktura-backend` API endpoints. It is the contract
the middleware package implements.

## Scope

The backend exposes REST endpoints under `/api/...` and a GraphQL
endpoint under `/query`. All of these require a valid Keycloak-issued
JWT access token.

## Token Acquisition

Tokens are obtained out-of-band by clients (typically the Customer-Web
or Admin-Web SPAs) via OIDC Authorization Code flow against the
Keycloak realm `EEGFaktura`. The backend does **not** participate in
the token-acquisition flow — it only validates tokens presented in
request headers.

## Validation Steps

For each authenticated request the middleware MUST:

1. **Extract** the bearer token from the `Authorization: Bearer ...` header.
   Reject with `401 Unauthorized` if the header is missing or not
   `Bearer`-prefixed.

2. **Verify signature** of the JWT against the Keycloak realm's JWKS
   endpoint. The signing key is identified by the `kid` header in the
   JWT and looked up in a cached JWKS document. The cache refreshes
   asynchronously and on cache miss.

3. **Verify standard claims**:
   - `exp` (expiry) must be in the future
   - `iat` (issued-at) must be in the past
   - `iss` (issuer) must match the configured Keycloak realm URL
   - `aud` (audience) must contain the configured client identifier
     (string form per the `aud=string`-Constraint of this backend —
     see ADR-0003 in the platform repo)

4. **Extract platform claims** into a `PlatformClaims` struct:
   - `tenant` (Array<string>) — tenant identifiers the user has access to
   - `preferred_username` (string)
   - `access_groups` (Array<string>) — `/EEG_ADMIN`, `/EEG_USER`, etc.

5. **Validate tenant header**: the request MUST carry an `X-Tenant`
   header (or `tenant` header, for backward compatibility with existing
   clients). The value MUST be contained in `claims.tenant`.
   Comparison is case-insensitive. Reject with `403 Forbidden` if not.

6. **Pass through** to the protected handler with the validated
   `PlatformClaims` and the canonical tenant string (uppercase).

## Failure Responses

| Condition | Status |
|---|---|
| No `Authorization` header | 401 |
| `Authorization` not Bearer-prefixed | 400 |
| JWT signature invalid | 401 |
| JWT expired | 401 |
| JWT issuer mismatch | 401 |
| JWT audience mismatch | 401 |
| Tenant header missing | 403 |
| Tenant not in `claims.tenant` | 403 |
| Unhandled internal error | 500 |

## Configuration

Configuration is loaded from `config.yaml` (or env-var overrides via
viper). The auth-relevant fields:

```yaml
oidc:
  issuer_url: "https://auth.example.org/realms/EEGFaktura"
  audience: "at.ourproject.vfeeg.api"
  jwks_refresh_interval: "1h"
  jwks_refresh_timeout: "10s"
```

The static `pubKeyFile` setting from the public stand is retained as a
fallback for **development/offline** mode only and will be removed once
all environments use OIDC-Discovery (see Phase 2 of this initiative).

## Behavior in Dev/Test/Prod

| Aspect | Dev (STACKIT eu02) | Test (planned) | Prod (planned) |
|---|---|---|---|
| Issuer URL | `https://auth.eegfaktura-dev.duckdns.org/realms/EEGFaktura` | TBD | TBD |
| Audience | `at.ourproject.vfeeg.api` | same | same |
| Tenant Header Source | `X-Tenant` | same | same |

## Out of Scope (Future Work)

- Token introspection against Keycloak's introspection endpoint
  (currently the backend trusts local signature/claim validation only)
- Refresh-Token handling (clients do that themselves; backend only sees
  Access-Tokens)
- mTLS for backend-internal cluster traffic

## References

- AGPL-compatible go-OIDC and JWKS libraries:
  - `github.com/MicahParks/keyfunc/v3` (JWKS-cache + keyfunc)
  - `github.com/coreos/go-oidc/v3/oidc` (issuer-discovery; optional)
- Platform-level auth design: `docs/auth-and-services.md` and
  `docs/keycloak-realm-spec.md` in the corresponding platform repo.
