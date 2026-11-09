package wire

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/auth"
)

// xuidCapturingProvider note le xuid que le contexte porte à l'entrée de
// TryOAuthRefreshWithRotation, là où SISUProvider le transmet à l'échange OAuth.
type xuidCapturingProvider struct {
	fakeTokenProvider
	xuids []string
}

func (p *xuidCapturingProvider) TryOAuthRefreshWithRotation(ctx context.Context, rt string) (string, string, error) {
	p.xuids = append(p.xuids, auth.RefreshXUIDFromContext(ctx))
	return p.fakeTokenProvider.TryOAuthRefreshWithRotation(ctx, rt)
}

// TestTryRefreshFromAuthStore_PoseLeXUID : le chemin serveur du registre porte le xuid du joueur
// jusqu'au fournisseur, donc jusqu'à l'échange OAuth.
func TestTryRefreshFromAuthStore_PoseLeXUID(t *testing.T) {
	prov := &xuidCapturingProvider{}
	reg := &ServiceRegistry{
		authStore: &mockUserTokenStore{user: &auth.UserTokens{XUID: "2533000000000023", OAuthRefreshToken: "rt-old"}},
		provider:  prov,
	}
	if result, _ := reg.tryRefreshFromAuthStore(context.Background(), "2533000000000023"); result == nil {
		t.Fatal("résultat nil inattendu")
	}
	if len(prov.xuids) != 1 || prov.xuids[0] != "2533000000000023" {
		t.Errorf("xuids vus par le fournisseur = %q, attendu [2533000000000023]", prov.xuids)
	}
}
