package pool

import (
	"context"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/platform/auth"
)

// xuidProvider note le xuid que le contexte porte à l'entrée de TryOAuthRefreshWithRotation, là
// où SISUProvider le transmet à l'échange OAuth.
type xuidProvider struct {
	mockTokenProvider
	mu   sync.Mutex
	vues []string
}

func (p *xuidProvider) TryOAuthRefreshWithRotation(ctx context.Context, refreshToken string) (string, string, error) {
	p.mu.Lock()
	p.vues = append(p.vues, auth.RefreshXUIDFromContext(ctx))
	p.mu.Unlock()
	return p.mockTokenProvider.TryOAuthRefreshWithRotation(ctx, refreshToken)
}

// TestResolver_RefreshPorteLeXUIDDeLaSource : le pool pose le xuid de la source sur le contexte du
// rafraîchissement ; une source sans xuid laisse le contexte sans xuid.
func TestResolver_RefreshPorteLeXUIDDeLaSource(t *testing.T) {
	provider := &xuidProvider{mockTokenProvider: mockTokenProvider{
		oauthRefreshResult: "access-token",
		exchangeResult:     &auth.ExchangeResult{},
	}}
	r := NewResolver(provider, time.Hour, nil)

	for _, src := range []CredentialSource{
		{Gamertag: "Avec", XUID: "2533000000000099", RefreshToken: "rt-1"},
		{Gamertag: "Sans", RefreshToken: "rt-2"},
	} {
		if _, err := r.Resolve(context.Background(), src); err != nil {
			t.Fatalf("Resolve(%s) : %v", src.Gamertag, err)
		}
	}
	if len(provider.vues) != 2 || provider.vues[0] != "2533000000000099" || provider.vues[1] != "" {
		t.Errorf("xuids vus par le fournisseur = %q, attendu [2533000000000099 \"\"]", provider.vues)
	}
}
