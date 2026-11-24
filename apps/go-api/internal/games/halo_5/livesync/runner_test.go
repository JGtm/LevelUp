package livesync

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
)

func viewer() canonical.PlayerIdentity {
	return canonical.PlayerIdentity{Gamertag: "JGtm", XUID: "xJG"}
}

// batchWith construit un MatchBatch minimal (registry + nParts participants + nMedals médailles).
func batchWith(matchID string, nParts, nMedals int) *persist.MatchBatch {
	b := persist.NewBatchBuilder("halo_5", "JGtm", "xJG", "test").
		SetMatch(&domain.MatchRegistryRow{MatchID: matchID})
	if nParts > 0 {
		parts := make([]domain.MatchParticipantRow, nParts)
		for i := range parts {
			parts[i] = domain.MatchParticipantRow{MatchID: matchID, XUID: "x"}
		}
		b.AddParticipants(parts)
	}
	if nMedals > 0 {
		medals := make([]domain.MedalRow, nMedals)
		for i := range medals {
			medals[i] = domain.MedalRow{MatchID: matchID}
		}
		b.AddMedals(medals)
	}
	return b.Build()
}

// captureReturning : CaptureFunc qui ignore la source et renvoie des batches canned.
func captureReturning(batches []*persist.MatchBatch, stats halo5.CaptureStats, err error) CaptureFunc {
	return func(context.Context, halo5.CaptureSource, canonical.PlayerIdentity, func(string) string, func(string) bool, halo5.CaptureOptions) ([]*persist.MatchBatch, halo5.CaptureStats, error) {
		return batches, stats, err
	}
}

func TestRunDelta_HappyPath(t *testing.T) {
	deps := Deps{
		NewSource: func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
		Capture: captureReturning(
			[]*persist.MatchBatch{batchWith("m1", 8, 3), batchWith("m2", 8, 2)},
			halo5.CaptureStats{MatchesSeen: 2, MatchesCollected: 2}, nil),
		Viewer:     viewer(),
		LoadKnown:  func(context.Context) (map[string]bool, error) { return map[string]bool{}, nil },
		PersistAll: func(_ context.Context, b []*persist.MatchBatch) ([]*persist.MatchBatch, []string) { return b, nil },
	}
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{MaxMatches: 50})
	if err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if res.MatchesInserted != 2 || res.ParticipantsDone != 16 || res.MedalsInserted != 5 {
		t.Errorf("res = %+v, want inserted2 / participants16 / medals5", res)
	}
	if res.Status() != "success" || len(res.InsertedMatchIDs) != 2 {
		t.Errorf("status=%s ids=%v", res.Status(), res.InsertedMatchIDs)
	}
}

func TestRunDelta_SourceUnavailableDegrades(t *testing.T) {
	deps := Deps{
		NewSource: func(context.Context) (halo5.CaptureSource, error) { return nil, errors.New("SpartanToken absent") },
		Viewer:    viewer(),
	}
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil {
		t.Fatalf("RunDelta ne doit pas renvoyer d'erreur dure: %v", err)
	}
	// re-auth → failure (0 inséré, erreur remontée), aucune panique.
	if res.Status() != "failure" || len(res.Errors) == 0 {
		t.Errorf("source KO → failure + erreur, got %+v", res)
	}
}

func TestRunDelta_PartialPersist(t *testing.T) {
	deps := Deps{
		NewSource: func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
		Capture: captureReturning(
			[]*persist.MatchBatch{batchWith("m1", 4, 0), batchWith("m2", 4, 0)},
			halo5.CaptureStats{MatchesSeen: 2, MatchesCollected: 2}, nil),
		Viewer: viewer(),
		PersistAll: func(_ context.Context, b []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
			return b[:1], []string{"persist m2 échec"} // m1 OK, m2 échoue
		},
	}
	res, _ := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if res.MatchesInserted != 1 || res.MatchesSkipped != 1 || res.Status() != "partial_success" {
		t.Errorf("partial: inserted=%d skipped=%d status=%s", res.MatchesInserted, res.MatchesSkipped, res.Status())
	}
}

func TestRunDelta_KnownSetErrorBestEffort(t *testing.T) {
	called := false
	deps := Deps{
		NewSource: func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
		LoadKnown: func(context.Context) (map[string]bool, error) { return nil, errors.New("DB KO") },
		Capture: func(_ context.Context, _ halo5.CaptureSource, _ canonical.PlayerIdentity, _ func(string) string, isKnown func(string) bool, _ halo5.CaptureOptions) ([]*persist.MatchBatch, halo5.CaptureStats, error) {
			called = true
			if isKnown("anything") {
				t.Error("known-set KO → isKnown doit tout retourner false (pas de delta)")
			}
			return nil, halo5.CaptureStats{}, nil
		},
		Viewer:     viewer(),
		PersistAll: func(_ context.Context, b []*persist.MatchBatch) ([]*persist.MatchBatch, []string) { return b, nil },
	}
	res, _ := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if !called {
		t.Error("la collecte doit tourner malgré known-set KO (best-effort)")
	}
	if len(res.Errors) == 0 {
		t.Error("l'erreur known-set doit être remontée")
	}
}

// TestRunDelta_NotifyFirstSync : le hook « titre prêt » (MT-19 / axe E) est appelé
// dès que le titre a des matchs (known non vide OU insert>0), jamais s'il n'en a
// aucun. L'idempotence durable vit côté notifier (watermark), PAS dans ce hook —
// d'où l'appel en steady-state (retry-until-watermark si la 1re émission a échoué).
func TestRunDelta_NotifyFirstSync(t *testing.T) {
	makeDeps := func(known map[string]bool, batches []*persist.MatchBatch, notify func(context.Context, int)) Deps {
		return Deps{
			NewSource:       func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
			Capture:         captureReturning(batches, halo5.CaptureStats{MatchesSeen: len(batches), MatchesCollected: len(batches)}, nil),
			Viewer:          viewer(),
			LoadKnown:       func(context.Context) (map[string]bool, error) { return known, nil },
			PersistAll:      func(_ context.Context, b []*persist.MatchBatch) ([]*persist.MatchBatch, []string) { return b, nil },
			NotifyFirstSync: notify,
		}
	}

	// (1) 1er sync : known vide + 1 match inséré → notifié 1× avec inserted=1.
	var calls1, got1 int
	d1 := makeDeps(map[string]bool{}, []*persist.MatchBatch{batchWith("m1", 8, 0)},
		func(_ context.Context, inserted int) { calls1++; got1 = inserted })
	if _, err := NewRunner(d1, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if calls1 != 1 || got1 != 1 {
		t.Errorf("1er sync : appels=%d inserted=%d, want 1/1", calls1, got1)
	}

	// (2) Aucun match (known vide + rien inséré) → jamais notifié (pas de faux positif).
	var calls2 int
	d2 := makeDeps(map[string]bool{}, nil, func(context.Context, int) { calls2++ })
	if _, err := NewRunner(d2, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if calls2 != 0 {
		t.Errorf("aucun match : appels=%d, want 0", calls2)
	}

	// (3) Steady-state (known non vide, 0 nouveau) → notifié (le hook ne dédup pas ;
	// le watermark côté notifier rend l'émission idempotente).
	var calls3 int
	d3 := makeDeps(map[string]bool{"m1": true}, nil, func(context.Context, int) { calls3++ })
	if _, err := NewRunner(d3, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if calls3 != 1 {
		t.Errorf("steady-state : appels=%d, want 1", calls3)
	}

	// (4) nil-safe : Deps.NotifyFirstSync absent → aucun panic (runner sans notif).
	d4 := makeDeps(map[string]bool{}, []*persist.MatchBatch{batchWith("m1", 1, 0)}, nil)
	if _, err := NewRunner(d4, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta nil notify: %v", err)
	}
}

// TestRunner_Signature : Runner satisfait le contrat DeltaRunner (sans importer
// scheduler, pour éviter le cycle scheduler→livesync).
func TestRunner_Signature(t *testing.T) {
	var _ interface {
		RunDelta(context.Context, domain.SyncOptions) (domain.SyncResult, error)
	} = (*Runner)(nil)
}

// TestShouldRunPostScore couvre la décision PURE du gate post-score (filet de
// convergence à 0 insert). insert>0 → toujours ; 0 insert → seulement si la sonde
// de backlog renvoie true ; sonde nil/false à 0 insert → ne rien faire.
func TestShouldRunPostScore(t *testing.T) {
	yes := func(context.Context) bool { return true }
	no := func(context.Context) bool { return false }
	cases := []struct {
		name     string
		inserted int
		backlog  func(context.Context) bool
		want     bool
	}{
		{"insert>0 sans sonde", 2, nil, true},
		{"insert>0 sonde false (sonde ignorée)", 1, no, true},
		{"0 insert sonde nil", 0, nil, false},
		{"0 insert backlog présent", 0, yes, true},
		{"0 insert pas de backlog", 0, no, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRunPostScore(context.Background(), tc.inserted, tc.backlog); got != tc.want {
				t.Errorf("shouldRunPostScore(inserted=%d) = %v, want %v", tc.inserted, got, tc.want)
			}
		})
	}
}

// TestRunDelta_PostScoreConvergenceZeroInsert : à 0 match inséré, PostScore se
// déclenche QUAND la sonde de backlog d'enrichment renvoie true (match du titre
// inséré en shared par un coéquipier, jamais enrichi chez ce joueur) — et JAMAIS
// quand elle renvoie false. Mirroir du filet de convergence Infinite.
func TestRunDelta_PostScoreConvergenceZeroInsert(t *testing.T) {
	makeDeps := func(backlog func(context.Context) bool, postScore func(context.Context, halo5.CaptureSource, []string) error) Deps {
		return Deps{
			NewSource: func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
			// 0 match collecté → 0 inséré : on isole le filet de convergence.
			Capture:              captureReturning(nil, halo5.CaptureStats{}, nil),
			Viewer:               viewer(),
			LoadKnown:            func(context.Context) (map[string]bool, error) { return map[string]bool{"m1": true}, nil },
			PersistAll:           func(_ context.Context, b []*persist.MatchBatch) ([]*persist.MatchBatch, []string) { return b, nil },
			HasEnrichmentBacklog: backlog,
			PostScore:            postScore,
		}
	}

	// (1) Backlog présent → PostScore appelé malgré 0 insert.
	calledWithBacklog := false
	d1 := makeDeps(
		func(context.Context) bool { return true },
		func(context.Context, halo5.CaptureSource, []string) error { calledWithBacklog = true; return nil },
	)
	if _, err := NewRunner(d1, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if !calledWithBacklog {
		t.Error("0 insert + backlog → PostScore doit être appelé (filet de convergence)")
	}

	// (2) Pas de backlog → PostScore PAS appelé (comportement strict insert>0).
	calledNoBacklog := false
	d2 := makeDeps(
		func(context.Context) bool { return false },
		func(context.Context, halo5.CaptureSource, []string) error { calledNoBacklog = true; return nil },
	)
	if _, err := NewRunner(d2, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if calledNoBacklog {
		t.Error("0 insert + pas de backlog → PostScore ne doit PAS être appelé")
	}

	// (3) Sonde nil → jamais appelé à 0 insert (runner sans sonde).
	calledNilProbe := false
	d3 := makeDeps(nil, func(context.Context, halo5.CaptureSource, []string) error { calledNilProbe = true; return nil })
	if _, err := NewRunner(d3, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if calledNilProbe {
		t.Error("0 insert + sonde nil → PostScore ne doit PAS être appelé")
	}
}

// TestRunDelta_RunAchievements : le hook achievements (parité Infinite) est appelé
// à CHAQUE post-sync, indépendamment de l'insertion de matchs (les achievements
// évoluent hors match). Une erreur reste best-effort (cycle non avorté, remontée
// dans Errors). nil → pas d'appel (runner unit-testable sans réseau Xbox).
func TestRunDelta_RunAchievements(t *testing.T) {
	baseDeps := func(achievements func(context.Context) error) Deps {
		return Deps{
			NewSource:       func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
			Capture:         captureReturning(nil, halo5.CaptureStats{}, nil),
			Viewer:          viewer(),
			LoadKnown:       func(context.Context) (map[string]bool, error) { return map[string]bool{}, nil },
			PersistAll:      func(_ context.Context, b []*persist.MatchBatch) ([]*persist.MatchBatch, []string) { return b, nil },
			RunAchievements: achievements,
		}
	}

	// (1) Appelé même à 0 match inséré (les achievements ne dépendent pas des matchs).
	calls := 0
	d1 := baseDeps(func(context.Context) error { calls++; return nil })
	res, err := NewRunner(d1, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if calls != 1 {
		t.Errorf("achievements appels=%d à 0 insert, want 1 (parité Infinite)", calls)
	}
	if len(res.Errors) != 0 {
		t.Errorf("succès achievements ne doit pas remonter d'erreur: %v", res.Errors)
	}

	// (2) Erreur best-effort : cycle non avorté, erreur remontée dans le SyncResult.
	d2 := baseDeps(func(context.Context) error { return errors.New("XSTS KO") })
	res2, err := NewRunner(d2, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil {
		t.Fatalf("RunDelta ne doit pas renvoyer d'erreur dure: %v", err)
	}
	if len(res2.Errors) == 0 {
		t.Error("l'erreur achievements doit être remontée dans Errors (best-effort)")
	}

	// (3) nil-safe : Deps.RunAchievements absent → aucun panic.
	d3 := baseDeps(nil)
	if _, err := NewRunner(d3, nil).RunDelta(context.Background(), domain.SyncOptions{}); err != nil {
		t.Fatalf("RunDelta nil achievements: %v", err)
	}
}
