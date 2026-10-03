package pool

// resolver_family_test.go — provenance du token (préfixe RpsTicket) : le pool la
// pose avant l'échange et persiste celle qui a été MESURÉE.
//
// Régression du bruit de boot du 2026-09-20 : 5 WARN « RpsTicket refusé (401) —
// retry sur l'autre préfixe » à CHAQUE démarrage, avec family="" — le pool ne
// lisait ni n'écrivait jamais la provenance, donc chaque boot repartait de « d= »
// pour des comptes qui n'acceptent que « t= ».

import (
	"context"
	"sync"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// familyProvider : double d'échangeur qui n'accepte qu'une provenance et
// renseigne la mesure comme le fait le chokepoint XBL réel.
type familyProvider struct {
	mockTokenProvider

	accepted string // provenance que l'endpoint accepte

	mu       sync.Mutex
	seenCtx  []string // provenance posée en ctx à chaque Exchange
	measured int      // nombre d'échanges ayant mesuré une provenance
}

func (p *familyProvider) Exchange(ctx context.Context, _ string) (*auth.ExchangeResult, error) {
	p.mu.Lock()
	p.seenCtx = append(p.seenCtx, auth.TokenClientFamilyFromContext(ctx))
	p.measured++
	p.mu.Unlock()
	// Le vrai chokepoint essaie le préfixe déduit de la provenance du ctx, et
	// retombe sur l'autre en 401 ; dans les deux cas il enregistre CELUI QUI A
	// MARCHÉ. On modélise le résultat, pas les allers-retours HTTP.
	auth.RecordObservedTokenFamily(ctx, p.accepted)
	return &auth.ExchangeResult{
		Tokens: &domain.HaloTokens{SpartanToken: "spartan", ClearanceToken: "clearance"},
	}, nil
}

func (p *familyProvider) snapshot() ([]string, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.seenCtx...), p.measured
}

// TestResolver_ProvenanceMesureePersistee : une provenance inconnue au scan mais
// mesurée à l'échange remonte au callback de persistance, puis est REPOSÉE dans le
// ctx de l'échange suivant (c'est ce qui supprime le 401 au boot d'après).
func TestResolver_ProvenanceMesureePersistee(t *testing.T) {
	provider := &familyProvider{accepted: auth.TokenFamilyXboxNative}
	provider.oauthRefreshResult = "access_token"

	type persisted struct{ gamertag, xuid, family string }
	var got []persisted
	r := NewResolverWithCallbacks(provider, 0, ResolverCallbacks{
		OnFamilyObserved: func(_ context.Context, gamertag, xuid, family string) error {
			got = append(got, persisted{gamertag, xuid, family})
			return nil
		},
	})

	src := CredentialSource{
		Gamertag: "Alice", XUID: "111",
		RefreshToken: "rt", Source: "watcher_oauth",
		// Provenance inconnue : l'état du store avant ce lot.
	}
	if _, err := r.Resolve(context.Background(), src); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("callback de persistance appelé %d fois, want 1 — la mesure serait perdue au prochain boot", len(got))
	}
	if got[0].family != auth.TokenFamilyXboxNative || got[0].xuid != "111" || got[0].gamertag != "Alice" {
		t.Errorf("provenance persistée = %+v, want {Alice 111 %s}", got[0], auth.TokenFamilyXboxNative)
	}

	// Échange suivant (cache invalidé) : la provenance mesurée est REPOSÉE.
	if _, err := r.Refresh(context.Background(), "Alice"); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	seen, measured := provider.snapshot()
	if measured != 2 {
		t.Fatalf("%d échanges, want 2", measured)
	}
	if seen[0] != "" {
		t.Errorf("1er échange : provenance en ctx = %q, want vide (inconnue)", seen[0])
	}
	if seen[1] != auth.TokenFamilyXboxNative {
		t.Errorf("2e échange : provenance en ctx = %q, want %q — le pool n'a pas réutilisé la mesure",
			seen[1], auth.TokenFamilyXboxNative)
	}
	if len(got) != 1 {
		t.Errorf("callback rappelé %d fois — une mesure inchangée ne doit pas réécrire le store", len(got))
	}
}

// TestResolver_ProvenanceDejaConnueNonReecrite : une provenance déjà portée par la
// source est posée en ctx dès le premier échange, et ne déclenche aucune écriture.
func TestResolver_ProvenanceDejaConnueNonReecrite(t *testing.T) {
	provider := &familyProvider{accepted: auth.TokenFamilyAzure}
	provider.oauthRefreshResult = "access_token"

	calls := 0
	r := NewResolverWithCallbacks(provider, 0, ResolverCallbacks{
		OnFamilyObserved: func(context.Context, string, string, string) error {
			calls++
			return nil
		},
	})

	src := CredentialSource{
		Gamertag: "Bob", XUID: "222", RefreshToken: "rt", Source: "watcher_oauth",
		TokenClientFamily: auth.TokenFamilyAzure,
	}
	if _, err := r.Resolve(context.Background(), src); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	seen, _ := provider.snapshot()
	if seen[0] != auth.TokenFamilyAzure {
		t.Errorf("provenance en ctx = %q, want %q dès le premier échange", seen[0], auth.TokenFamilyAzure)
	}
	if calls != 0 {
		t.Errorf("callback appelé %d fois pour une provenance inchangée, want 0", calls)
	}
}
