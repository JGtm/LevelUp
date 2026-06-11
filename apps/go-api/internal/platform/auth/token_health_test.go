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
		name                            string
		u                               UserTokens
		wantRefresh, wantMSAL, wantXSTS string
	}{
		{
			name:        "tout absent",
			u:           UserTokens{},
			wantRefresh: TokenAbsent, wantMSAL: TokenAbsent, wantXSTS: TokenAbsent,
		},
		{
			name: "tout sain",
			u: UserTokens{
				OAuthRefreshToken: "rt", MSALCacheJSON: "{}",
				AccessToken: "at", OAuthExpiresAt: future,
				XSTSToken: "xs", XSTSExpiresAt: future,
			},
			wantRefresh: TokenOK, wantMSAL: TokenOK, wantXSTS: TokenOK,
		},
		{
			name: "reauth requis (RT mort)",
			u: UserTokens{
				OAuthRefreshToken: "rt", ReauthRequired: true,
				MSALCacheJSON: "{}", OAuthExpiresAt: future,
				XSTSToken: "xs", XSTSExpiresAt: future,
			},
			wantRefresh: TokenReauth, wantMSAL: TokenOK, wantXSTS: TokenOK,
		},
		{
			name: "xsts expiré, msal expire bientôt",
			u: UserTokens{
				OAuthRefreshToken: "rt", MSALCacheJSON: "{}", OAuthExpiresAt: soon,
				XSTSToken: "xs", XSTSExpiresAt: past,
			},
			wantRefresh: TokenOK, wantMSAL: TokenExpiring, wantXSTS: TokenExpired,
		},
		{
			name:        "cache MSAL présent sans expiry connu → ok",
			u:           UserTokens{MSALCacheJSON: "{}"},
			wantRefresh: TokenOK, wantMSAL: TokenOK, wantXSTS: TokenAbsent,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := c.u.Health(now, margin)
			if h.Refresh != c.wantRefresh || h.MSAL != c.wantMSAL || h.XSTS != c.wantXSTS {
				t.Fatalf("Health = %+v ; want refresh=%s msal=%s xsts=%s",
					h, c.wantRefresh, c.wantMSAL, c.wantXSTS)
			}
		})
	}
}
