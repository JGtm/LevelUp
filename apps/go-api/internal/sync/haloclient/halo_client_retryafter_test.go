package haloclient

import (
	"net/http"
	"testing"
	"time"
)

// TestParseRetryAfter couvre les deux formes RFC 7231 (delta-seconds + date HTTP)
// et le clamp [0, maxRetryAfter]. Revue lot A (Retry-After 429/503).
func TestParseRetryAfter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	cases := []struct {
		name string
		h    string
		want time.Duration
	}{
		{"empty", "", 0},
		{"seconds", "120", 120 * time.Second},
		{"zero", "0", 0},
		{"negative", "-5", 0},
		{"garbage", "abc", 0},
		{"clamp_over_max", "99999", maxRetryAfter},
		{"http_date_future", now.Add(90 * time.Second).Format(http.TimeFormat), 90 * time.Second},
		{"http_date_past", now.Add(-time.Minute).Format(http.TimeFormat), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseRetryAfter(tc.h, now); got != tc.want {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tc.h, got, tc.want)
			}
		})
	}
}
