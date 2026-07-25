package service

// explorer_service_target_status_test.go — Lot A3 (fin de la dégradation muette,
// .ai/V7.1/PLAN_EXPLORER_LIVE_REPAIR_2026-07.md) : couvre le statut
// (domain.ExplorerLiveSectionStatus) renvoyé par les 4 fetchs live de l'encart
// "Profil joueur cible" — identity, career (service record), season_csrs,
// combat_live. Chaque fonction est testée directement (sans passer par
// GetCommonMatches) pour isoler la logique de statut des autres sous-systèmes
// déjà couverts par explorer_service_test.go.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// --- fetchTargetIdentityRaw ---

func TestFetchTargetIdentityRaw_StatusOK_LocalIdentity(t *testing.T) {
	t.Parallel()
	local := &mockLocalIdentityResolver{identity: &domain.HomeSpartanIdentityRow{RankNumber: 10}}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: local})

	id, status := svc.fetchTargetIdentityRaw(context.Background(), "target-xuid", "Target", false)
	if id == nil {
		t.Fatal("identité locale attendue non-nil")
	}
	if status != domain.ExplorerLiveOK {
		t.Errorf("status = %q, want ok (identité locale résolue)", status)
	}
}

func TestFetchTargetIdentityRaw_StatusOK_LiveIdentity(t *testing.T) {
	t.Parallel()
	local := &mockLocalIdentityResolver{identity: nil} // cible non suivie
	live := &mockLiveIdentity{identity: &domain.HomeSpartanIdentityRow{RankNumber: 5}}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: local, LiveIdentity: live})

	id, status := svc.fetchTargetIdentityRaw(context.Background(), "target-xuid", "Target", true)
	if id == nil {
		t.Fatal("identité live attendue non-nil")
	}
	if status != domain.ExplorerLiveOK {
		t.Errorf("status = %q, want ok (identité live résolue)", status)
	}
}

// TestFetchTargetIdentityRaw_StatusNoAuth : cible non-locale, pas d'auth → le
// fetch live n'est JAMAIS tenté (mockLiveIdentity.called doit rester false).
// L'identité peut rester non-nil (repli bannière pool) mais le statut explique
// pourquoi (no_auth), jamais silencieux.
func TestFetchTargetIdentityRaw_StatusNoAuth(t *testing.T) {
	t.Parallel()
	local := &mockLocalIdentityResolver{identity: nil}
	live := &mockLiveIdentity{identity: &domain.HomeSpartanIdentityRow{RankNumber: 5}}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: local, LiveIdentity: live})

	_, status := svc.fetchTargetIdentityRaw(context.Background(), "target-xuid", "Target", false)
	if live.called {
		t.Error("FetchLiveIdentity ne doit JAMAIS être appelé sans auth")
	}
	if status != domain.ExplorerLiveNoAuth {
		t.Errorf("status = %q, want no_auth", status)
	}
}

// TestFetchTargetIdentityRaw_StatusFailed_LiveError : cible non-locale, auth
// disponible, mais le fetch live échoue → failed (même si le repli bannière pool
// produit tout de même une identité minimale, jamais nil sans raison).
func TestFetchTargetIdentityRaw_StatusFailed_LiveError(t *testing.T) {
	t.Parallel()
	local := &mockLocalIdentityResolver{identity: nil}
	live := &mockLiveIdentity{err: errors.New("halo api down")}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: local, LiveIdentity: live})

	_, status := svc.fetchTargetIdentityRaw(context.Background(), "target-xuid", "Target", true)
	if !live.called {
		t.Error("FetchLiveIdentity devait être tenté (auth disponible)")
	}
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status = %q, want failed", status)
	}
}

// TestFetchTargetIdentityRaw_StatusLocalPartial_NoLiveProvider : auth disponible
// mais aucun LiveIdentity câblé (dépendance absente) → local_partial (dégradation
// structurelle, pas un échec réseau).
func TestFetchTargetIdentityRaw_StatusLocalPartial_NoLiveProvider(t *testing.T) {
	t.Parallel()
	local := &mockLocalIdentityResolver{identity: nil}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: local})

	_, status := svc.fetchTargetIdentityRaw(context.Background(), "target-xuid", "Target", true)
	if status != domain.ExplorerLiveLocalPartial {
		t.Errorf("status = %q, want local_partial", status)
	}
}

// --- fetchTargetServiceRecord (career + top médailles + saisons) ---

func TestFetchTargetServiceRecord_StatusNoAuth(t *testing.T) {
	t.Parallel()
	remote := &mockRemoteStatsProvider{stats: &domain.NormalizedPlayerStats{Matches: 10}}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RemoteStats: remote})

	stats, _, _, _, status := svc.fetchTargetServiceRecord(context.Background(), "Target", false)
	if stats != nil {
		t.Errorf("stats attendues nil sans auth, got %+v", stats)
	}
	if remote.called {
		t.Error("FetchServiceRecord ne doit JAMAIS être appelé sans auth")
	}
	if status != domain.ExplorerLiveNoAuth {
		t.Errorf("status = %q, want no_auth", status)
	}
}

func TestFetchTargetServiceRecord_StatusFailed_NoProvider(t *testing.T) {
	t.Parallel()
	svc := NewExplorerService(&mockExplorerRepo{}, "self") // RemoteStats non câblé

	stats, _, _, _, status := svc.fetchTargetServiceRecord(context.Background(), "Target", true)
	if stats != nil {
		t.Errorf("stats attendues nil sans provider, got %+v", stats)
	}
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status = %q, want failed", status)
	}
}

func TestFetchTargetServiceRecord_StatusFailed_Error(t *testing.T) {
	t.Parallel()
	remote := &mockRemoteStatsProvider{err: errors.New("waypoint 500")}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RemoteStats: remote})

	_, _, _, _, status := svc.fetchTargetServiceRecord(context.Background(), "Target", true)
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status = %q, want failed", status)
	}
}

func TestFetchTargetServiceRecord_StatusOK(t *testing.T) {
	t.Parallel()
	remote := &mockRemoteStatsProvider{stats: &domain.NormalizedPlayerStats{Matches: 42}}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RemoteStats: remote})

	stats, _, _, _, status := svc.fetchTargetServiceRecord(context.Background(), "Target", true)
	if stats == nil || stats.Matches != 42 {
		t.Fatalf("stats attendues non-nil (Matches=42), got %+v", stats)
	}
	if status != domain.ExplorerLiveOK {
		t.Errorf("status = %q, want ok", status)
	}
}

// --- fetchTargetCSR ---

func TestFetchTargetCSR_StatusNoAuth(t *testing.T) {
	t.Parallel()
	called := false
	csr := CSRProviderFunc(func(context.Context, string, string) ([]domain.CareerPlaylistCSR, error) {
		called = true
		return nil, nil
	})
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{CSR: csr, CurrentSeasonID: "s1"})

	_, status := svc.fetchTargetCSR(context.Background(), "target-xuid", false)
	if called {
		t.Error("SeasonCSRs ne doit JAMAIS être appelé sans auth")
	}
	if status != domain.ExplorerLiveNoAuth {
		t.Errorf("status = %q, want no_auth", status)
	}
}

func TestFetchTargetCSR_StatusFailed_NoProvider(t *testing.T) {
	t.Parallel()
	svc := NewExplorerService(&mockExplorerRepo{}, "self") // CSR non câblé

	_, status := svc.fetchTargetCSR(context.Background(), "target-xuid", true)
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status = %q, want failed", status)
	}
}

func TestFetchTargetCSR_StatusFailed_Error(t *testing.T) {
	t.Parallel()
	csr := CSRProviderFunc(func(context.Context, string, string) ([]domain.CareerPlaylistCSR, error) {
		return nil, errors.New("skill api down")
	})
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{CSR: csr, CurrentSeasonID: "s1"})

	_, status := svc.fetchTargetCSR(context.Background(), "target-xuid", true)
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status = %q, want failed", status)
	}
}

// TestFetchTargetCSR_StatusOK_EmptyIsOK : une réponse VIDE sans erreur (joueur non
// classé) est un état valide → ok, pas failed (ne pas confondre absence de rang
// avec un échec de fetch).
func TestFetchTargetCSR_StatusOK_EmptyIsOK(t *testing.T) {
	t.Parallel()
	csr := CSRProviderFunc(func(context.Context, string, string) ([]domain.CareerPlaylistCSR, error) {
		return nil, nil
	})
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{CSR: csr, CurrentSeasonID: "s1"})

	csrs, status := svc.fetchTargetCSR(context.Background(), "target-xuid", true)
	if csrs != nil {
		t.Errorf("csrs attendues nil (joueur non classé), got %+v", csrs)
	}
	if status != domain.ExplorerLiveOK {
		t.Errorf("status = %q, want ok (liste vide != échec)", status)
	}
}

// --- computeTargetCombatProfileLive ---

func TestComputeTargetCombatProfileLive_StatusNoAuth(t *testing.T) {
	t.Parallel()
	live := &mockRecentMatches{rows: []domain.ExplorerTargetRecentMatch{{MatchID: "m1"}}}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RecentMatches: live})

	rows, status := svc.computeTargetCombatProfileLive(context.Background(), "target-xuid", false)
	if rows != nil {
		t.Errorf("rows attendues nil sans auth, got %+v", rows)
	}
	if live.calls != 0 {
		t.Error("FetchRecentMatches ne doit JAMAIS être appelé sans auth")
	}
	if status != domain.ExplorerLiveNoAuth {
		t.Errorf("status = %q, want no_auth", status)
	}
}

func TestComputeTargetCombatProfileLive_StatusFailed_NoProvider(t *testing.T) {
	t.Parallel()
	svc := NewExplorerService(&mockExplorerRepo{}, "self") // RecentMatches non câblé

	rows, status := svc.computeTargetCombatProfileLive(context.Background(), "target-xuid", true)
	if rows != nil {
		t.Errorf("rows attendues nil sans provider, got %+v", rows)
	}
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status = %q, want failed", status)
	}
}

func TestComputeTargetCombatProfileLive_StatusFailed_Error(t *testing.T) {
	t.Parallel()
	live := &mockRecentMatches{err: errors.New("live fetch timeout")}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RecentMatches: live})

	_, status := svc.computeTargetCombatProfileLive(context.Background(), "target-xuid", true)
	if status != domain.ExplorerLiveFailed {
		t.Errorf("status = %q, want failed", status)
	}
}

func TestComputeTargetCombatProfileLive_StatusOK(t *testing.T) {
	t.Parallel()
	live := &mockRecentMatches{rows: []domain.ExplorerTargetRecentMatch{{MatchID: "m1"}, {MatchID: "m2"}}}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RecentMatches: live})

	rows, status := svc.computeTargetCombatProfileLive(context.Background(), "target-xuid", true)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if status != domain.ExplorerLiveOK {
		t.Errorf("status = %q, want ok", status)
	}
}

// --- buildTargetProfile : LiveStatus renseigné en bout de chaîne ---

// TestBuildTargetProfile_LiveStatusPopulated prouve que GetCommonMatches (chemin
// public) renseigne LiveStatus pour les 5 sections, jamais de valeur vide — cf.
// domain.ExplorerLiveStatus. Aucune dépendance câblée → tout en échec/no-auth
// selon la disponibilité de l'auth (le point à vérifier est l'ABSENCE de statut
// zéro-valeur, pas la valeur précise de chaque section).
func TestBuildTargetProfile_LiveStatusPopulated(t *testing.T) {
	t.Parallel()
	repo := &mockExplorerRepo{xuid: "target-xuid"}
	svc := NewExplorerService(repo, "self") // aucun provider câblé

	resp, err := svc.GetCommonMatches(ctxAuth(false, "self"), "Target", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil")
	}
	ls := tp.LiveStatus
	for name, got := range map[string]domain.ExplorerLiveSectionStatus{
		"identity":    ls.Identity,
		"career":      ls.Career,
		"season_csrs": ls.SeasonCSRs,
		"seasons":     ls.Seasons,
		"combat_live": ls.CombatLive,
	} {
		if got == "" {
			t.Errorf("live_status.%s vide — chaque section DOIT être statuée (jamais silencieuse)", name)
		}
	}
	// Sans auth, les 4 sections dépendant de tokens sont no_auth (seasons peut
	// aussi être no_auth via le fallback local — cf. computeSeasonBreakdown).
	if ls.Career != domain.ExplorerLiveNoAuth {
		t.Errorf("live_status.career = %q, want no_auth", ls.Career)
	}
	if ls.CombatLive != domain.ExplorerLiveNoAuth {
		t.Errorf("live_status.combat_live = %q, want no_auth", ls.CombatLive)
	}
	if ls.Seasons != domain.ExplorerLiveNoAuth {
		t.Errorf("live_status.seasons = %q, want no_auth", ls.Seasons)
	}
}
