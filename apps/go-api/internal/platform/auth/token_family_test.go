// Package auth — token_family_test.go : tests de la classification diagnostique
// des familles d'access_token Microsoft. NOTE : le préfixe RpsTicket ne se
// déduit PAS du format (les tokens Azure ET MSA natifs sont des compact tickets
// "EwA…") — cf. requestUserToken (retry d= puis t=) et TestRequestUserToken_RetryTPrefixOn401.
package auth

import "testing"

// TestAccessTokenFormat : classification diagnostique (jamais le token loggé).
func TestAccessTokenFormat(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"eyJ0eXAiOiJKV1QifQ.x.y", "jwt_aad"},
		{"EwAoA8l6BAAU", "msa_compact"},
		{"EwB1234", "msa_compact"},
		{"autre-chose", "inconnu"},
		{"", "inconnu"},
	}
	for _, tc := range tests {
		if got := accessTokenFormat(tc.token); got != tc.want {
			t.Errorf("accessTokenFormat(%q) = %q, attendu %q", tc.token, got, tc.want)
		}
	}
}
