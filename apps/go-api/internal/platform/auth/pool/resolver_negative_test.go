// Package pool — resolver_negative_test.go : cache négatif des échecs OAuth
// permanents + fix de la boucle boot NewPool (plan anti-bruit 2026-06-11).
package pool

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/platform/auth"
)

func (m *mockTokenProvider) oauthCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, c := range m.callLog {
		if c == "TryOAuthRefreshWithRotation" {
			n++
		}
	}
	return n
}

func configError() error {
	return &auth.OAuthExchangeError{
		ErrorCode:   "invalid_request",
		Description: "AADSTS90023: Public clients can't send a client secret.",
	}
}

// TestResolver_NegativeCache_PermanentErrorShortCircuits : un échec de classe
// config est mémorisé — le 2e Resolve ne rappelle PAS le provider et retourne
// une erreur marquée ErrPermanentAuthFailure.
func TestResolver_NegativeCache_PermanentErrorShortCircuits(t *testing.T) {
	provider := &mockTokenProvider{oauthRefreshErr: configError()}
	var gotClasses []string
	onAuthError := func(_ context.Context, _, _, class, _ string) {
		gotClasses = append(gotClasses, class)
	}
	r := NewResolverWithCallbacks(provider, time.Hour, ResolverCallbacks{OnAuthError: onAuthError})
	src := CredentialSource{Gamertag: "GT", XUID: "123", RefreshToken: "rt-bad"}

	if _, err := r.Resolve(context.Background(), src); err == nil {
		t.Fatalf("1er Resolve : erreur attendue")
	}
	if provider.oauthCallCount() != 1 {
		t.Fatalf("1er Resolve : 1 appel provider attendu, obtenu %d", provider.oauthCallCount())
	}

	_, err := r.Resolve(context.Background(), src)
	if err == nil {
		t.Fatalf("2e Resolve : erreur attendue")
	}
	if !errors.Is(err, ErrPermanentAuthFailure) {
		t.Errorf("2e Resolve : ErrPermanentAuthFailure attendu, obtenu %v", err)
	}
	if provider.oauthCallCount() != 1 {
		t.Errorf("2e Resolve : aucun nouvel appel provider attendu (cache négatif), obtenu %d", provider.oauthCallCount())
	}
	if len(gotClasses) != 1 || gotClasses[0] != string(auth.AuthErrorConfig) {
		t.Errorf("onAuthError : attendu [config], obtenu %v", gotClasses)
	}
}

// TestResolver_NegativeCache_TransientNotCached : une erreur transitoire
// (réseau) n'est PAS mémorisée — chaque Resolve retente.
func TestResolver_NegativeCache_TransientNotCached(t *testing.T) {
	provider := &mockTokenProvider{oauthRefreshErr: errors.New("dial tcp: timeout")}
	r := NewResolverWithCallbacks(provider, time.Hour, ResolverCallbacks{})
	src := CredentialSource{Gamertag: "GT", XUID: "123", RefreshToken: "rt"}

	_, _ = r.Resolve(context.Background(), src)
	_, _ = r.Resolve(context.Background(), src)
	if provider.oauthCallCount() != 2 {
		t.Errorf("erreur transitoire : 2 appels provider attendus, obtenu %d", provider.oauthCallCount())
	}
}

// TestResolver_NegativeCache_NewRTBypasses : une source au RT différent
// (re-capture) invalide l'entrée négative et retente immédiatement.
func TestResolver_NegativeCache_NewRTBypasses(t *testing.T) {
	provider := &mockTokenProvider{oauthRefreshErr: configError()}
	r := NewResolverWithCallbacks(provider, time.Hour, ResolverCallbacks{})

	_, _ = r.Resolve(context.Background(), CredentialSource{Gamertag: "GT", XUID: "123", RefreshToken: "rt-old"})
	if provider.oauthCallCount() != 1 {
		t.Fatalf("setup : 1 appel attendu, obtenu %d", provider.oauthCallCount())
	}

	_, err := r.Resolve(context.Background(), CredentialSource{Gamertag: "GT", XUID: "123", RefreshToken: "rt-new"})
	if errors.Is(err, ErrPermanentAuthFailure) {
		t.Errorf("RT différent : le cache négatif ne doit pas s'appliquer")
	}
	if provider.oauthCallCount() != 2 {
		t.Errorf("RT différent : 2e appel provider attendu, obtenu %d", provider.oauthCallCount())
	}
}

// TestResolver_NegativeCache_ClearedOnSuccess : un refresh réussi efface
// l'erreur persistée (onAuthError class="").
func TestResolver_NegativeCache_ClearedOnSuccess(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "access-token",
		exchangeResult:     &auth.ExchangeResult{},
	}
	var gotClasses []string
	onAuthError := func(_ context.Context, _, _, class, _ string) {
		gotClasses = append(gotClasses, class)
	}
	r := NewResolverWithCallbacks(provider, time.Hour, ResolverCallbacks{OnAuthError: onAuthError})

	if _, err := r.Resolve(context.Background(), CredentialSource{Gamertag: "GT", XUID: "123", RefreshToken: "rt"}); err != nil {
		t.Fatalf("Resolve : %v", err)
	}
	if len(gotClasses) != 1 || gotClasses[0] != "" {
		t.Errorf("onAuthError : effacement (class vide) attendu, obtenu %v", gotClasses)
	}
}

// perRTProvider : succès ou échec selon le refreshToken reçu — permet de
// simuler un pool multi-comptes où seuls certains RT sont morts.
type perRTProvider struct {
	mockTokenProvider
	badRT string
}

func (p *perRTProvider) TryOAuthRefreshWithRotation(_ context.Context, refreshToken string) (string, string, error) {
	if refreshToken == p.badRT {
		return "", "", configError()
	}
	return "access-token", "", nil
}

func (p *perRTProvider) Exchange(context.Context, string) (*auth.ExchangeResult, error) {
	return &auth.ExchangeResult{}, nil
}

// TestNewPool_SkipsFailingSourceAndKeepsTheRest : régression du bug de boucle
// boot (i-- + poolSize--) qui retentait le même index et ABANDONNAIT toutes
// les sources situées après la première en échec.
func TestNewPool_SkipsFailingSourceAndKeepsTheRest(t *testing.T) {
	provider := &perRTProvider{badRT: "rt-dead"}
	r := NewResolverWithCallbacks(provider, time.Hour, ResolverCallbacks{})

	sources := []CredentialSource{
		{Gamertag: "OK1", XUID: "1", RefreshToken: "rt-1"},
		{Gamertag: "DEAD", XUID: "2", RefreshToken: "rt-dead"},
		{Gamertag: "OK2", XUID: "3", RefreshToken: "rt-3"},
		{Gamertag: "OK3", XUID: "4", RefreshToken: "rt-4"},
	}
	p, err := NewPool(context.Background(), r, sources, PoolOptions{})
	if err != nil {
		t.Fatalf("NewPool : %v", err)
	}
	defer p.Close()

	impl, ok := p.(*poolImpl)
	if !ok {
		t.Fatalf("poolImpl attendu")
	}
	if len(impl.slots) != 3 {
		t.Fatalf("slots : attendu 3 (OK1, OK2, OK3 — DEAD skippé), obtenu %d", len(impl.slots))
	}
	got := map[string]bool{}
	for _, s := range impl.slots {
		got[s.gamertag] = true
	}
	for _, want := range []string{"OK1", "OK2", "OK3"} {
		if !got[want] {
			t.Errorf("slot %s manquant (sources après l'échec abandonnées ?)", want)
		}
	}
	if got["DEAD"] {
		t.Errorf("le slot DEAD ne doit pas exister")
	}
}
