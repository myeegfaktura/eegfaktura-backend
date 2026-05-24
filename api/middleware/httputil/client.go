package httputil

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PostFormUrlencoded sends a POST with `application/x-www-form-urlencoded`
// body — the format Keycloak's token endpoint requires. If `bearer` is
// non-empty an `Authorization: Bearer <bearer>` header is added.
//
// The returned *http.Response is the raw client response; the caller
// owns its Body and should close it (or pass it to one of the helpers
// in httputil.go which close on their own).
func PostFormUrlencoded(ctx context.Context, endpoint string, values url.Values, bearer string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return http.DefaultClient.Do(req)
}
