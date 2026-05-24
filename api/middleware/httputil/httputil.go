// Package httputil provides small HTTP-client helpers shared across
// middleware code (especially Keycloak token/admin flows). Mirrors the
// httputil/ subpackage in prod-vfeeg-backend v0.3.05 per ADR-0006.
//
// Today there is no in-fork consumer yet — the package is mirrored for
// parity-completeness; future ports of Keycloak admin / token-refresh
// logic will use these helpers instead of inlining the patterns.
package httputil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ErrUnanticipatedResponse is returned when an HTTP call succeeded
// transport-wise but the response status indicates an unexpected
// result. Callers should use errors.As to extract StatusCode and
// (truncated) Body for diagnostics.
type ErrUnanticipatedResponse struct {
	StatusCode int
	Body       string
}

// Error implements the error interface.
func (e *ErrUnanticipatedResponse) Error() string {
	return fmt.Sprintf("unanticipated HTTP response: status=%d body=%q", e.StatusCode, e.Body)
}

// NewErrUnanticipatedResponse reads up to 4 KiB of the response body
// (truncating beyond that) and returns an *ErrUnanticipatedResponse
// carrying the status code and the captured body. The response body
// is closed before returning.
func NewErrUnanticipatedResponse(resp *http.Response) error {
	defer func() { _ = resp.Body.Close() }()
	const maxBody = 4 << 10
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	return &ErrUnanticipatedResponse{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}

// DecodeJSONResponse decodes the JSON body of resp into target,
// closing resp.Body on return. If decoding fails, the underlying
// error is returned unwrapped (callers can errors.Is against
// io.ErrUnexpectedEOF etc.).
func DecodeJSONResponse(resp *http.Response, target any) error {
	defer func() { _ = resp.Body.Close() }()
	return json.NewDecoder(resp.Body).Decode(target)
}
