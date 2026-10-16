package domain

import "testing"

func TestErrorMessageForCode(t *testing.T) {
	cases := map[string]string{
		"player_not_found": "player not found",
		"invalid_body":     "invalid body",
		"sync_failed":      "sync failed",
		"forbidden":        "forbidden",
		"":                 "error",
	}
	for code, want := range cases {
		if got := ErrorMessageForCode(code); got != want {
			t.Errorf("ErrorMessageForCode(%q) = %q, attendu %q", code, got, want)
		}
	}
}

func TestNewJobError_MessageIsFixedByCode(t *testing.T) {
	e := NewJobError("transcode_failed", true)
	if e.Code != "transcode_failed" || e.Message != "transcode failed" || !e.Retryable {
		t.Errorf("NewJobError = %+v", e)
	}
}
