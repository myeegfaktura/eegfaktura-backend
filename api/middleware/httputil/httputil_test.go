package httputil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestErrUnanticipatedResponse_Error(t *testing.T) {
	e := &ErrUnanticipatedResponse{StatusCode: 503, Body: "service down"}
	got := e.Error()
	if !strings.Contains(got, "503") || !strings.Contains(got, "service down") {
		t.Errorf("Error() = %q, want status + body substrings", got)
	}
}

func TestNewErrUnanticipatedResponse_TruncatesLongBody(t *testing.T) {
	const N = 8 << 10 // 8 KiB; helper caps at 4 KiB
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", N))),
	}
	err := NewErrUnanticipatedResponse(resp)
	var u *ErrUnanticipatedResponse
	if !errors.As(err, &u) {
		t.Fatalf("expected *ErrUnanticipatedResponse, got %T", err)
	}
	if u.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", u.StatusCode)
	}
	if got := len(u.Body); got > 4<<10 {
		t.Errorf("Body length = %d, want ≤ 4096 (truncated)", got)
	}
}

func TestDecodeJSONResponse_ClosesBodyOnSuccess(t *testing.T) {
	cb := &closeCounter{Reader: strings.NewReader(`{"name":"test","n":7}`)}
	resp := &http.Response{StatusCode: 200, Body: cb}
	var target struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	if err := DecodeJSONResponse(resp, &target); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if target.Name != "test" || target.N != 7 {
		t.Errorf("decoded = %+v", target)
	}
	if cb.closed != 1 {
		t.Errorf("body.Close() called %d times, want 1", cb.closed)
	}
}

func TestPostFormUrlencoded_HeadersAndBody(t *testing.T) {
	var gotContentType, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", "tok-123")

	resp, err := PostFormUrlencoded(context.Background(), srv.URL, values, "bearer-xyz")
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()

	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotAuth != "Bearer bearer-xyz" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if !strings.Contains(gotBody, "grant_type=refresh_token") || !strings.Contains(gotBody, "refresh_token=tok-123") {
		t.Errorf("body = %q", gotBody)
	}
}

func TestPostFormUrlencoded_NoAuthWhenBearerEmpty(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := PostFormUrlencoded(context.Background(), srv.URL, url.Values{}, "")
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty when bearer omitted", gotAuth)
	}
}

// closeCounter wraps a Reader and counts Close() invocations.
type closeCounter struct {
	io.Reader
	closed int
}

func (c *closeCounter) Close() error { c.closed++; return nil }
