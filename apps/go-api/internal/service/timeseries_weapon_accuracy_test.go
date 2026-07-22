package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// timeseries_weapon_accuracy_test.go — verrouille l'exposition de la précision par arme
// (weapon_accuracy) sur le scope Timeseries, MIROIR EXACT du câblage Sessions
// (session_page_frag_distribution_test.go). GetPage peuple resp.WeaponAccuracy via le
// builder partagé buildWeaponAccuracy quand le repo précision renvoie des rows ; nil
// sinon (repo absent / capability absente sur Infinite). Réutilise le stub
// fakeSessionWeaponAccuracyRepo (même port, même package — pas de duplication).

// newTimeseriesWithAccuracy assemble un TimeseriesService de test sur le path canonical
// (miroir des tests GetPage existants) + un repo weapon_accuracy optionnel. Les
// StatsMatchRow fixent le scope (match_ids) transmis au repo précision.
func newTimeseriesWithAccuracy(rows []legacymatch.StatsMatchRow, accRepo port.WeaponAccuracyRepository) *TimeseriesService {
	svc := NewTimeseriesService(&mockTimeseriesRepo{}).
		WithPlayerMatchesRepo(newStatsMockFromRows(rows, nil), "halo_5", "GT")
	if accRepo != nil {
		svc = svc.WithWeaponAccuracyRepo(accRepo)
	}
	return svc
}

// twoTimeseriesMatches : 2 matchs minimaux (scope m1+m2) pour peupler `matches` dans
// GetPage — sans quoi le scope est vide et le repo précision n'est pas appelé.
func twoTimeseriesMatches() []legacymatch.StatsMatchRow {
	return []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: time.Now(), Kills: 10, Deaths: 4},
		{MatchID: "m2", StartTime: time.Now().Add(time.Hour), Kills: 8, Deaths: 6},
	}
}

// TestTimeseriesGetPage_WeaponAccuracyPopulated : quand le repo précision renvoie des
// rows, GetPage peuple resp.WeaponAccuracy via buildWeaponAccuracy (tri précision desc,
// rows sans label ignorées) et transmet le scope de la période (match_ids + gamertag)
// au repo — MIROIR du câblage Sessions.
func TestTimeseriesGetPage_WeaponAccuracyPopulated(t *testing.T) {
	accRepo := &fakeSessionWeaponAccuracyRepo{
		rows: []port.WeaponAccuracyRow{
			{WeaponID: 1, Label: "BR75", ShotsFired: 100, ShotsLanded: 40},  // 0.40
			{WeaponID: 2, Label: "Magnum", ShotsFired: 50, ShotsLanded: 45}, // 0.90
			{WeaponID: 3, Label: "", ShotsFired: 80, ShotsLanded: 80},       // sans label → ignoré
		},
	}
	svc := newTimeseriesWithAccuracy(twoTimeseriesMatches(), accRepo)

	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if len(resp.WeaponAccuracy) != 2 {
		t.Fatalf("WeaponAccuracy len = %d, want 2 (%+v)", len(resp.WeaponAccuracy), resp.WeaponAccuracy)
	}
	// Tri par précision décroissante : Magnum (0.90) devant BR75 (0.40).
	if resp.WeaponAccuracy[0].Label != "Magnum" || resp.WeaponAccuracy[1].Label != "BR75" {
		t.Errorf("ordre = [%q, %q], want [Magnum, BR75] (tri précision desc)",
			resp.WeaponAccuracy[0].Label, resp.WeaponAccuracy[1].Label)
	}
	// Le repo reçoit le scope de la période (match_ids filtrés + gamertag).
	if accRepo.gotFilters.Gamertag != "GT" || len(accRepo.gotFilters.MatchIDs) != 2 {
		t.Errorf("filtres reçus = %+v, want {MatchIDs:[m1 m2] Gamertag:GT}", accRepo.gotFilters)
	}
}

// TestTimeseriesGetPage_WeaponAccuracyNilWhenRepoAbsent : sans repo précision (câblage
// Infinite absent), resp.WeaponAccuracy reste nil (champ omis, le front retombe sur
// « Outils de destruction »).
func TestTimeseriesGetPage_WeaponAccuracyNilWhenRepoAbsent(t *testing.T) {
	svc := newTimeseriesWithAccuracy(twoTimeseriesMatches(), nil)
	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if resp.WeaponAccuracy != nil {
		t.Errorf("repo absent : WeaponAccuracy = %+v, want nil", resp.WeaponAccuracy)
	}
}

// TestTimeseriesGetPage_WeaponAccuracyCapabilityAbsent : sur un titre sans table
// weapon_accuracy (Infinite), le repo renvoie games.ErrCapabilityNotSupported →
// resp.WeaponAccuracy reste nil (best-effort, jamais de panic).
func TestTimeseriesGetPage_WeaponAccuracyCapabilityAbsent(t *testing.T) {
	accRepo := &fakeSessionWeaponAccuracyRepo{err: games.ErrCapabilityNotSupported}
	svc := newTimeseriesWithAccuracy(twoTimeseriesMatches(), accRepo)
	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if resp.WeaponAccuracy != nil {
		t.Errorf("capability absente : WeaponAccuracy = %+v, want nil", resp.WeaponAccuracy)
	}
}

// TestLoadTimeseriesWeaponAccuracy_Guards : nil si repo absent, gamertag vide ou scope
// vide (aucun appel repo — évite un rejet Validate loggé en Warn). Miroir de
// TestLoadSessionWeaponAccuracy_Guards.
func TestLoadTimeseriesWeaponAccuracy_Guards(t *testing.T) {
	ctx := context.Background()
	rows := []port.WeaponAccuracyRow{{WeaponID: 1, Label: "BR75", ShotsFired: 10, ShotsLanded: 5}}

	// Repo nil.
	svc := &TimeseriesService{titleSlug: "halo_5", gamertag: "GT"}
	if got := svc.loadTimeseriesWeaponAccuracy(ctx, []string{"m1"}); got != nil {
		t.Errorf("repo nil : got %+v, want nil", got)
	}

	// Gamertag vide.
	repo := &fakeSessionWeaponAccuracyRepo{rows: rows}
	svc = &TimeseriesService{titleSlug: "halo_5", gamertag: "", weaponAccuracyRepo: repo}
	if got := svc.loadTimeseriesWeaponAccuracy(ctx, []string{"m1"}); got != nil {
		t.Errorf("gamertag vide : got %+v, want nil", got)
	}

	// Scope vide.
	svc = &TimeseriesService{titleSlug: "halo_5", gamertag: "GT", weaponAccuracyRepo: repo}
	if got := svc.loadTimeseriesWeaponAccuracy(ctx, nil); got != nil {
		t.Errorf("scope vide : got %+v, want nil", got)
	}
}
