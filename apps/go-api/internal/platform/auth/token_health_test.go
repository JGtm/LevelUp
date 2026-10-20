package auth

import (
	"testing"
	"time"
)

func TestUserTokens_Health(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	margin := 5 * time.Minute
	future := now.Add(time.Hour)
	soon := now.Add(2 * time.Minute) // dans la marge
	past := now.Add(-time.Hour)

	cases := []struct {
		name                              string
		u                                 UserTokens
		wantRefresh, wantAccess, wantXSTS string
	}{
		{
			name:        "tout absent",
			u:           UserTokens{},
			wantRefresh: TokenAbsent, wantAccess: TokenAbsent, wantXSTS: TokenAbsent,
		},
		{
			name: "tout sain",
			u: UserTokens{
				OAuthRefreshToken: "rt",
				AccessToken:       "at", OAuthExpiresAt: future,
				XSTSToken: "xs", XSTSExpiresAt: future,
			},
			wantRefresh: TokenOK, wantAccess: TokenOK, wantXSTS: TokenOK,
		},
		{
			name: "reauth requis (RT mort)",
			u: UserTokens{
				OAuthRefreshToken: "rt", ReauthRequired: true,
				AccessToken: "at", OAuthExpiresAt: future,
				XSTSToken: "xs", XSTSExpiresAt: future,
			},
			wantRefresh: TokenReauth, wantAccess: TokenOK, wantXSTS: TokenOK,
		},
		{
			name: "xsts expiré, access expire bientôt",
			u: UserTokens{
				OAuthRefreshToken: "rt", AccessToken: "at", OAuthExpiresAt: soon,
				XSTSToken: "xs", XSTSExpiresAt: past,
			},
			wantRefresh: TokenOK, wantAccess: TokenExpiring, wantXSTS: TokenExpired,
		},
		{
			name:        "access_token présent sans expiry connu → ok",
			u:           UserTokens{AccessToken: "at"},
			wantRefresh: TokenAbsent, wantAccess: TokenOK, wantXSTS: TokenAbsent,
		},
		{
			// ADR 0023 Phase 5 : le RT est la SEULE credential de refresh — un
			// compte sans RT est « absent », même s'il porte encore un access_token.
			name:        "RT absent → refresh absent",
			u:           UserTokens{AccessToken: "at", OAuthExpiresAt: future},
			wantRefresh: TokenAbsent, wantAccess: TokenOK, wantXSTS: TokenAbsent,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := c.u.Health(now, margin)
			if h.Refresh != c.wantRefresh || h.Access != c.wantAccess || h.XSTS != c.wantXSTS {
				t.Fatalf("Health = %+v ; want refresh=%s access=%s xsts=%s",
					h, c.wantRefresh, c.wantAccess, c.wantXSTS)
			}
		})
	}
}
