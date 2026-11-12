package dbfault

import (
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestIsSharedSwapContention : les erreurs sentinelles du SharedDBProvider (swap RW
// en cours / provider en récupération) sont reconnues → 503 Retry-After côté HTTP,
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
		if got := IsSharedSwapContention(c.err); got != c.want {
			t.Errorf("%s: IsSharedSwapContention(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}

// TestIsHandleClosedOrInvalidated : une poignée fermée ou invalidée se retente (503) ; une erreur
// quelconque non.
func TestIsHandleClosedOrInvalidated(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"fermée", errors.New("sql: database is closed"), true},
		{"fermée encapsulée", fmt.Errorf("load: %w", errors.New("sql: database is closed")), true},
		{"invalidée", errors.New("database has been invalidated because of a previous fatal error"), true},
		{"quelconque", errors.New("boom"), false},
	}
	for _, c := range cases {
		if got := IsHandleClosedOrInvalidated(c.err); got != c.want {
			t.Errorf("%s: IsHandleClosedOrInvalidated(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}
