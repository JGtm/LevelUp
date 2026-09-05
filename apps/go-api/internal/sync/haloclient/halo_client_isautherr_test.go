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
		// Ronde 1 de revue du volet C (2026-09-05) : doPlayerGatedGet passe par
		// IsAuthError depuis que l'ex-prédicat textuel isAuthErr est mort. Le CDN
		// des films est PUBLIC : son 403 ne dit rien de nos tokens, et son message
		// contient pourtant « HTTP 403 » — le typage est ce qui les sépare.
		{"blob 403 (CDN public)", &BlobHTTPError{StatusCode: 403, URL: "u", Attempts: 1}, false},
		{"texte HTTP 401", errors.New("HTTP 401: https://economy"), false},
	}
	for _, c := range cases {
		if got := IsAuthError(c.err); got != c.want {
			t.Errorf("%s: IsAuthError = %v, want %v", c.name, got, c.want)
		}
	}
}
