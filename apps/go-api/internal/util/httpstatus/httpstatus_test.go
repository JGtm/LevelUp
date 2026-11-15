package httpstatus

import (
	"errors"
	"fmt"
	"testing"
)

func TestFromError(t *testing.T) {
	for _, tc := range []struct {
		nom  string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"sans statut", errors.New("connexion refusée"), 0},
		{"HTTP 500", fmt.Errorf("FetchCSRLeaderboard page 1: %w", errors.New("statut HTTP 500")), 500},
		{"HTTP 429 avec URL", errors.New("HTTP 429: https://exemple/api"), 429},
		{"trois chiffres seulement", errors.New("HTTP 5000 octets"), 0},
	} {
		if got := FromError(tc.err); got != tc.want {
			t.Errorf("%s : FromError = %d, attendu %d", tc.nom, got, tc.want)
		}
	}
	if !IsServerError(503) || IsServerError(404) || IsServerError(429) {
		t.Error("IsServerError : 503 oui, 404 et 429 non")
	}
	if !IsTooManyRequests(errors.New("HTTP 429")) || IsTooManyRequests(errors.New("HTTP 500")) {
		t.Error("IsTooManyRequests : 429 oui, 500 non")
	}
}
