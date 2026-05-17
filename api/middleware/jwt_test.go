package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestContains(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		set  []string
		need string
		want bool
	}{
		{"exact match", []string{"TE100200", "RC100130"}, "TE100200", true},
		{"case-insensitive match", []string{"te100200"}, "TE100200", true},
		{"case-insensitive needle", []string{"TE100200"}, "te100200", true},
		{"miss", []string{"TE100200"}, "RC100130", false},
		{"empty set", nil, "TE100200", false},
		{"empty needle", []string{"TE100200"}, "", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := contains(c.set, c.need)
			if got != c.want {
				t.Errorf("contains(%v, %q) = %v, want %v", c.set, c.need, got, c.want)
			}
		})
	}
}

func TestTenantHeaderPrefersXTenant(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Tenant", "RC100130")
	r.Header.Set("tenant", "TE100200")
	if got := tenantHeader(r); got != "RC100130" {
		t.Errorf("tenantHeader: want %q (X-Tenant precedence), got %q", "RC100130", got)
	}
}

func TestTenantHeaderFallsBackToLegacy(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("tenant", "TE100200")
	if got := tenantHeader(r); got != "TE100200" {
		t.Errorf("tenantHeader: want %q (legacy fallback), got %q", "TE100200", got)
	}
}

func TestTenantHeaderMissingReturnsEmpty(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	if got := tenantHeader(r); got != "" {
		t.Errorf("tenantHeader: want empty, got %q", got)
	}
}
