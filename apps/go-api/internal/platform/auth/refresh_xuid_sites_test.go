package auth

import (
	"context"
	"testing"
	"time"
)

// xuidCapturingProvider note le xuid que le contexte porte à l'entrée de
// TryOAuthRefreshWithRotation, là où SISUProvider le transmet à l'échange OAuth.
type xuidCapturingProvider struct {
	stubProvider
	xuids []string
}

func (p *xuidCapturingProvider) TryOAuthRefreshWithRotation(ctx context.Context, rt string) (string, string, error) {
	p.xuids = append(p.xuids, RefreshXUIDFromContext(ctx))
	return "at-capture", "", nil
}

// TestEnsureWatcherAccessToken_PoseLeXUIDDuStore : le rafraîchissement du watcher porte le xuid de
// l'entrée trouvée par gamertag.
func TestEnsureWatcherAccessToken_PoseLeXUIDDuStore(t *testing.T) {
	multi := NewMultiUserTokenStore(tempTokenDir(t))
	seedStoreOAuth(t, multi, "2533000000000021", "Joueur", "rt-store")
	store := newStoreWithTokens(t, &StoredTokens{AccessToken: "perime", OAuthExpiresAt: time.Now().Add(-time.Hour)})
	prov := &xuidCapturingProvider{}

	if at, err := EnsureWatcherAccessToken(context.Background(), multi, store, prov, "Joueur"); err != nil || at != "at-capture" {
		t.Fatalf("(%q, %v), attendu (at-capture, nil)", at, err)
	}
	if len(prov.xuids) != 1 || prov.xuids[0] != "2533000000000021" {
		t.Errorf("xuids vus par le fournisseur = %q, attendu [2533000000000021]", prov.xuids)
	}
}

// TestResolveMSAccessTokenStoreFirst_PoseLeXUID : la résolution d'access_token depuis le magasin
// porte le xuid demandé.
func TestResolveMSAccessTokenStoreFirst_PoseLeXUID(t *testing.T) {
	multi := NewMultiUserTokenStore(tempTokenDir(t))
	seedStoreOAuth(t, multi, "2533000000000022", "Joueur", "rt-store")
	prov := &xuidCapturingProvider{}

	if at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, multi, "2533000000000022", "Joueur"); err != nil || at != "at-capture" {
		t.Fatalf("(%q, %v), attendu (at-capture, nil)", at, err)
	}
	if len(prov.xuids) != 1 || prov.xuids[0] != "2533000000000022" {
		t.Errorf("xuids vus par le fournisseur = %q, attendu [2533000000000022]", prov.xuids)
	}
}
