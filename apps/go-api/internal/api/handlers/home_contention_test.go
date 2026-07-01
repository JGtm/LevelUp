package handlers

import (
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestIsSharedSwapContention : les erreurs sentinelles du SharedDBProvider (swap RW
// en cours / provider en récupération) sont reconnues → 503 Retry-After côté home,
// y compris quand elles sont wrappées via %w. Une erreur quelconque ne l'est pas.
func TestIsSharedSwapContention(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"swap_timeout", sharedprovider.ErrSwapTimeout, true},
		{"swap_failed", sharedprovider.ErrSwapFailed, true},
		{"provider_closed", sharedprovider.ErrProviderClosed, true},
		{"wrapped_swap_timeout", fmt.Errorf("PlayerMatchesRepo.Load: %w", sharedprovider.ErrSwapTimeout), true},
		{"random", errors.New("boom"), false},
		{"random_wrapped", fmt.Errorf("ctx: %w", errors.New("deadline")), false},
	}
	for _, c := range cases {
		if got := isSharedSwapContention(c.err); got != c.want {
			t.Errorf("%s: isSharedSwapContention(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}
