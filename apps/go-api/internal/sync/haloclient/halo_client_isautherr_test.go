// Package sync — halo_client_isautherr_test.go : garde-fou IsAuthError (filet 401
// du client sync, consommé par halo.RetryWithFreshTokens côté Explorer).
package haloclient

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsAuthError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"401", &HTTPError{StatusCode: 401, URL: "u"}, true},
		{"403", &HTTPError{StatusCode: 403, URL: "u"}, true},
		{"404", &HTTPError{StatusCode: 404, URL: "u"}, false},
		{"429", &HTTPError{StatusCode: 429, URL: "u"}, false},
		{"wrapped 401", fmt.Errorf("ctx: %w", &HTTPError{StatusCode: 401, URL: "u"}), true},
		{"plain error", errors.New("boom"), false},
	}
	for _, c := range cases {
		if got := IsAuthError(c.err); got != c.want {
			t.Errorf("%s: IsAuthError = %v, want %v", c.name, got, c.want)
		}
	}
}
