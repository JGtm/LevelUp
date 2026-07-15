// Package auth — token_family_test.go : tests de la classification des familles
// d'access_token Microsoft (JWT Azure AD vs compact ticket MSA natif) et du
// préfixe RpsTicket qui en découle (fix 401 SISU 2026-07-15).
package auth

import "testing"

// TestRpsTicketPrefix : "d=" pour un JWT Azure AD, "t=" pour tout le reste
// (compact tickets MSA natifs du flow SISU).
func TestRpsTicketPrefix(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"jwt_azure_ad", "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.payload.sig", "d="},
		{"msa_compact_ticket", "EwAoA8l6BAAU...", "t="},
		{"format_inconnu", "abc123", "t="},
	}
	for _, tc := range tests {
		if got := rpsTicketPrefix(tc.token); got != tc.want {
			t.Errorf("%s : rpsTicketPrefix = %q, attendu %q", tc.name, got, tc.want)
		}
	}
}

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
