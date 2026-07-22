package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// fakeSessionWeaponAccuracyRepo est un stub de port.WeaponAccuracyRepository : il
// renvoie des rows OU une erreur configurée, et capture les filtres reçus pour
// vérifier le scope (matchIDs + gamertag).
type fakeSessionWeaponAccuracyRepo struct {
	rows       []port.WeaponAccuracyRow
	err        error
	gotFilters port.WeaponAccuracyFilters
}

func (f *fakeSessionWeaponAccuracyRepo) LoadWeaponAccuracyAggregated(
	_ context.Context, _ string, filters port.WeaponAccuracyFilters,
) ([]port.WeaponAccuracyRow, error) {
	f.gotFilters = filters
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

// TestAttachSessionFragDistribution_WeaponAccuracy : quand le repo précision renvoie
// des rows, attachSessionFragDistribution peuple entry.WeaponAccuracy via le builder
// partagé buildWeaponAccuracy (tri précision desc), et transmet le scope de session
// (matchIDs + gamertag) au repo.
func TestAttachSessionFragDistribution_WeaponAccuracy(t *testing.T) {
	repo := &fakeSessionWeaponAccuracyRepo{
		rows: []port.WeaponAccuracyRow{
			{WeaponID: 1, Label: "BR75", ShotsFired: 100, ShotsLanded: 40},  // 0.40
			{WeaponID: 2, Label: "Magnum", ShotsFired: 50, ShotsLanded: 45}, // 0.90
			{WeaponID: 3, Label: "", ShotsFired: 80, ShotsLanded: 80},       // sans label → ignoré
		},
	}
	svc := &SessionPageService{titleSlug: "halo_5", gamertag: "GT", weaponAccuracyRepo: repo}
	entry := &domain.SessionCompareEntry{}

	svc.attachSessionFragDistribution(context.Background(), entry, nil, []string{"m1", "m2"})

	if len(entry.WeaponAccuracy) != 2 {
		t.Fatalf("WeaponAccuracy len = %d, want 2 (%+v)", len(entry.WeaponAccuracy), entry.WeaponAccuracy)
	}
	// Tri par précision décroissante : Magnum (0.90) devant BR75 (0.40).
	if entry.WeaponAccuracy[0].Label != "Magnum" || entry.WeaponAccuracy[1].Label != "BR75" {
		t.Errorf("ordre = [%q, %q], want [Magnum, BR75] (tri précision desc)",
			entry.WeaponAccuracy[0].Label, entry.WeaponAccuracy[1].Label)
	}
	// Le repo reçoit bien le scope de la session (matchIDs + gamertag).
	if repo.gotFilters.Gamertag != "GT" || len(repo.gotFilters.MatchIDs) != 2 {
		t.Errorf("filtres reçus = %+v, want {MatchIDs:[m1 m2] Gamertag:GT}", repo.gotFilters)
	}
}

// TestAttachSessionFragDistribution_WeaponAccuracyCapabilityAbsent : sur un titre sans
// table weapon_accuracy (Infinite), le repo renvoie games.ErrCapabilityNotSupported →
// WeaponAccuracy reste nil (champ omis, le front retombe sur « Détails des frags »).
func TestAttachSessionFragDistribution_WeaponAccuracyCapabilityAbsent(t *testing.T) {
	repo := &fakeSessionWeaponAccuracyRepo{err: games.ErrCapabilityNotSupported}
	svc := &SessionPageService{titleSlug: "halo_infinite", gamertag: "GT", weaponAccuracyRepo: repo}
	entry := &domain.SessionCompareEntry{}

	svc.attachSessionFragDistribution(context.Background(), entry, nil, []string{"m1"})

	if entry.WeaponAccuracy != nil {
		t.Errorf("capability absente : WeaponAccuracy = %+v, want nil", entry.WeaponAccuracy)
	}
}

// TestLoadSessionWeaponAccuracy_Guards : nil si repo absent, gamertag vide ou scope
// vide (aucun appel au repo — évite un rejet Validate loggé en Warn).
func TestLoadSessionWeaponAccuracy_Guards(t *testing.T) {
	ctx := context.Background()
	rows := []port.WeaponAccuracyRow{{WeaponID: 1, Label: "BR75", ShotsFired: 10, ShotsLanded: 5}}

	// Repo nil.
	svc := &SessionPageService{titleSlug: "halo_5", gamertag: "GT"}
	if got := svc.loadSessionWeaponAccuracy(ctx, []string{"m1"}); got != nil {
		t.Errorf("repo nil : got %+v, want nil", got)
	}

	// Gamertag vide.
	repo := &fakeSessionWeaponAccuracyRepo{rows: rows}
	svc = &SessionPageService{titleSlug: "halo_5", gamertag: "", weaponAccuracyRepo: repo}
	if got := svc.loadSessionWeaponAccuracy(ctx, []string{"m1"}); got != nil {
		t.Errorf("gamertag vide : got %+v, want nil", got)
	}

	// Scope vide.
	svc = &SessionPageService{titleSlug: "halo_5", gamertag: "GT", weaponAccuracyRepo: repo}
	if got := svc.loadSessionWeaponAccuracy(ctx, nil); got != nil {
		t.Errorf("scope vide : got %+v, want nil", got)
	}
}
