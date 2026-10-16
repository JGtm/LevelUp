package pool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// sharingPool construit un pool sur trois sources A, B, C ; `excluded` liste celles dont le
// propriétaire n'a pas consenti à la mise en commun.
func sharingPool(t *testing.T, excluded ...string) Pool {
	t.Helper()
	sources := testSlotEnv(3)
	for i := range sources {
		for _, gt := range excluded {
			if sources[i].Gamertag == gt {
				sources[i].PoolExcluded = true
			}
		}
	}
	p, err := NewPool(context.Background(), &testResolver{resolved: map[string]*ResolvedTokens{}},
		sources, PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

// servedBy acquiert n jetons en PolicyAnyPublic et retourne l'ensemble des gamertags servis.
func servedBy(t *testing.T, p Pool, n int) map[string]bool {
	t.Helper()
	served := map[string]bool{}
	for i := 0; i < n; i++ {
		lease, err := p.Acquire(context.Background(), PolicyAnyPublic, "")
		if err != nil {
			t.Fatalf("Acquire #%d: %v", i, err)
		}
		served[lease.Gamertag] = true
		lease.Release()
	}
	return served
}

func TestAcquireAnyPublic_NeverServesExcludedSlot(t *testing.T) {
	p := sharingPool(t, "B")

	served := servedBy(t, p, 12)

	if served["B"] {
		t.Fatal("le jeton d'un joueur sans consentement ne doit jamais sortir du round-robin public")
	}
	if !served["A"] || !served["C"] {
		t.Fatalf("les jetons partagés doivent tous servir: %v", served)
	}
}

func TestAcquireAnyPublic_AllExcludedReportsNoHealthySlot(t *testing.T) {
	p := sharingPool(t, "A", "B", "C")

	_, err := p.Acquire(context.Background(), PolicyAnyPublic, "")

	if !errors.Is(err, ErrNoHealthySlot) {
		t.Fatalf("want ErrNoHealthySlot, got %v", err)
	}
}

func TestAcquirePinned_StillServesExcludedSlot(t *testing.T) {
	p := sharingPool(t, "B")

	lease, err := p.Acquire(context.Background(), PolicyPinnedPlayer, "B")

	if err != nil {
		t.Fatalf("un joueur sans consentement garde l'usage de son propre jeton: %v", err)
	}
	if lease.Gamertag != "B" {
		t.Fatalf("lease.Gamertag = %q, want B", lease.Gamertag)
	}
}

func TestIsWithheld(t *testing.T) {
	p := sharingPool(t, "B")

	if p.IsWithheld("A") {
		t.Fatal("A partage, il ne retient rien")
	}
	if !p.IsWithheld("B") {
		t.Fatal("B n'a pas consenti")
	}
	if p.IsWithheld("inconnu") {
		t.Fatal("un joueur sans slot n'a rien retenu")
	}
}

func TestSetShared_AppliesImmediately(t *testing.T) {
	p := sharingPool(t)

	if !servedBy(t, p, 12)["B"] {
		t.Fatal("précondition: B sert avant le retrait")
	}

	if !p.SetShared("B", false) {
		t.Fatal("SetShared sur un slot existant doit retourner true")
	}
	if servedBy(t, p, 12)["B"] {
		t.Fatal("après un retrait, B ne doit plus servir personne")
	}

	if !p.SetShared("B", true) {
		t.Fatal("SetShared sur un slot existant doit retourner true")
	}
	if !servedBy(t, p, 12)["B"] {
		t.Fatal("après un nouvel accord, B sert de nouveau")
	}

	if p.SetShared("inconnu", true) {
		t.Fatal("SetShared sur un gamertag sans slot doit retourner false")
	}
}

func TestAddOrUpdateSource_PropagatesExclusion(t *testing.T) {
	p := sharingPool(t)
	src := testSlotEnv(3)[1] // B

	src.PoolExcluded = true
	if err := p.AddOrUpdateSource(context.Background(), src); err != nil {
		t.Fatalf("AddOrUpdateSource: %v", err)
	}
	if servedBy(t, p, 12)["B"] {
		t.Fatal("un re-scan qui constate un retrait doit exclure le slot")
	}

	src.PoolExcluded = false
	if err := p.AddOrUpdateSource(context.Background(), src); err != nil {
		t.Fatalf("AddOrUpdateSource: %v", err)
	}
	if !servedBy(t, p, 12)["B"] {
		t.Fatal("un re-scan qui constate un accord remet le slot en service")
	}
}

// Discovery est le seul producteur de CredentialSource en production : sans décision
// enregistrée, la source doit sortir exclue.
func TestDiscoveryScan_PoolExcludedFollowsConsent(t *testing.T) {
	cases := []struct {
		name     string
		decide   func(*auth.MultiUserTokenStore, string) error
		wantExcl bool
	}{
		{"aucune décision", func(*auth.MultiUserTokenStore, string) error { return nil }, true},
		{"accord", func(s *auth.MultiUserTokenStore, x string) error {
			return s.SetPoolSharing(x, true, auth.PoolSharingBasisUserChoice)
		}, false},
		{"retrait", func(s *auth.MultiUserTokenStore, x string) error {
			return s.SetPoolSharing(x, false, auth.PoolSharingBasisUserChoice)
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := fakeConfigWithPlayers(t, "Alice", "111")
			storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
			if err := os.MkdirAll(storeDir, 0o700); err != nil {
				t.Fatal(err)
			}
			store := auth.NewMultiUserTokenStore(storeDir)
			if err := store.Upsert(&auth.UserTokens{XUID: "111", Gamertag: "Alice", OAuthRefreshToken: "rt"}); err != nil {
				t.Fatal(err)
			}
			if err := tc.decide(store, "111"); err != nil {
				t.Fatal(err)
			}

			d := NewDiscoveryWithStore(cfg, titlePkg.NewPathResolver(cfg.RepoRoot), titlePkg.DefaultSlug, store)
			sources, err := d.Scan(context.Background())
			if err != nil || len(sources) != 1 {
				t.Fatalf("Scan: %v, %d sources", err, len(sources))
			}
			if sources[0].PoolExcluded != tc.wantExcl {
				t.Fatalf("PoolExcluded = %v, want %v", sources[0].PoolExcluded, tc.wantExcl)
			}
		})
	}
}
