// Package api — post_sync_rival_encounters_test.go : tests de l'émission
// « rival croisé » (lot relations-E) via un détecteur factice + recordingEmitter.
package wire

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// fakeRivalDetector : détecteur factice. Capture l'appel (since) et renvoie les
// encounters programmés — permet d'asserter le SKIP watermark (jamais appelé).
type fakeRivalDetector struct {
	encounters []domain.RivalEncounter
	err        error
	called     bool
	gotSince   time.Time
}

func (f *fakeRivalDetector) DetectRivalEncounters(_ context.Context, since time.Time) ([]domain.RivalEncounter, error) {
	f.called = true
	f.gotSince = since
	return f.encounters, f.err
}

func TestEmitRivalEncounters_NewDuelEmitsNotification(t *testing.T) {
	before := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)}
	after := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)}
	det := &fakeRivalDetector{
		encounters: []domain.RivalEncounter{
			{XUID: "rival-1", Gamertag: "Nemesis", MatchID: "m-new", Outcome: "win", KillsOnRival: 12, DeathsByRival: 5},
		},
	}
	em := &recordingEmitter{}

	emitRivalEncounters(context.Background(), em, det, "player-slug", before, after)

	if !det.called {
		t.Fatal("le détecteur aurait dû être appelé (nouveau match présent)")
	}
	if !det.gotSince.Equal(before.LastMatchStartTime) {
		t.Fatalf("since attendu = watermark before %v, obtenu %v", before.LastMatchStartTime, det.gotSince)
	}
	if len(em.emitted) != 1 {
		t.Fatalf("attendu 1 notification émise, obtenu %d", len(em.emitted))
	}
	in := em.emitted[0]
	if in.Category != notifications.CategoryRivalEncounter {
		t.Errorf("catégorie attendue %q, obtenue %q", notifications.CategoryRivalEncounter, in.Category)
	}
	if in.Severity != notifications.SeveritySuccess {
		t.Errorf("duel gagné → severity success attendue, obtenue %q", in.Severity)
	}
	if in.TargetRoute != "/players/player-slug/matches/m-new" {
		t.Errorf("target_route match view attendue, obtenue %q", in.TargetRoute)
	}
	if in.Source != postSyncSource {
		t.Errorf("source attendue %q, obtenue %q", postSyncSource, in.Source)
	}
	for k, want := range map[string]any{
		"gamertag": "Nemesis", "outcome": "win", "kills": 12, "deaths": 5, "match_id": "m-new",
	} {
		if in.Params[k] != want {
			t.Errorf("param %q attendu %v, obtenu %v", k, want, in.Params[k])
		}
	}
}

func TestEmitRivalEncounters_LossIsInfoSeverity(t *testing.T) {
	before := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)}
	after := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)}
	det := &fakeRivalDetector{
		encounters: []domain.RivalEncounter{
			{Gamertag: "Nemesis", MatchID: "m-loss", Outcome: "loss", KillsOnRival: 3, DeathsByRival: 9},
		},
	}
	em := &recordingEmitter{}

	emitRivalEncounters(context.Background(), em, det, "player-slug", before, after)

	if len(em.emitted) != 1 || em.emitted[0].Severity != notifications.SeverityInfo {
		t.Fatalf("duel perdu → 1 notif severity info attendue, obtenu %+v", em.emitted)
	}
}

func TestEmitRivalEncounters_NoNewMatchSkipsDetection(t *testing.T) {
	ts := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	before := &PlayerSnapshot{LastMatchStartTime: ts}
	after := &PlayerSnapshot{LastMatchStartTime: ts} // watermark inchangé → aucune sync utile
	det := &fakeRivalDetector{
		encounters: []domain.RivalEncounter{{MatchID: "should-not-be-used"}},
	}
	em := &recordingEmitter{}

	emitRivalEncounters(context.Background(), em, det, "player-slug", before, after)

	if det.called {
		t.Error("aucun nouveau match → le détecteur ne doit PAS être appelé")
	}
	if len(em.emitted) != 0 {
		t.Errorf("aucun nouveau match → 0 émission attendue, obtenu %d", len(em.emitted))
	}
}

// TestEmitRivalEncounters_DetectorErrorIsBestEffort : une erreur de détection ne
// doit JAMAIS émettre ni faire échouer le flux (best-effort strict). Le détecteur
// est bien appelé (watermark avancé) mais son erreur est loguée et absorbée.
func TestEmitRivalEncounters_DetectorErrorIsBestEffort(t *testing.T) {
	before := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)}
	after := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)}
	det := &fakeRivalDetector{err: errInjected}
	em := &recordingEmitter{}

	emitRivalEncounters(context.Background(), em, det, "slug", before, after)

	if !det.called {
		t.Fatal("nouveau match présent → le détecteur doit être appelé avant l'erreur")
	}
	if len(em.emitted) != 0 {
		t.Fatalf("erreur de détection best-effort → 0 émission attendue, obtenu %d", len(em.emitted))
	}
}

// TestEmitRivalEncounters_EmitErrorIsBestEffort : si l'émetteur échoue sur un duel,
// la boucle ne panique pas et poursuit (branche `continue`). Ici failOn cible la
// catégorie rival → les deux émissions échouent, aucune n'est enregistrée, mais le
// traitement des duels suivants n'est pas interrompu.
func TestEmitRivalEncounters_EmitErrorIsBestEffort(t *testing.T) {
	before := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)}
	after := &PlayerSnapshot{LastMatchStartTime: time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)}
	det := &fakeRivalDetector{
		encounters: []domain.RivalEncounter{
			{Gamertag: "A", MatchID: "m1", Outcome: "win"},
			{Gamertag: "B", MatchID: "m2", Outcome: "loss"},
		},
	}
	em := &recordingEmitter{failOn: notifications.CategoryRivalEncounter}

	emitRivalEncounters(context.Background(), em, det, "slug", before, after)

	if len(em.emitted) != 0 {
		t.Fatalf("émetteur en échec → 0 notification enregistrée, obtenu %d", len(em.emitted))
	}
}

// TestNewRivalDetectorForPDB_NilReturnsNil : une player DB invalide (nil ou sans
// Player) produit un détecteur nil → l'émission est proprement sautée en amont
// (jamais de construction de service sur une DB absente).
func TestNewRivalDetectorForPDB_NilReturnsNil(t *testing.T) {
	if d := newRivalDetectorForPDB(nil); d != nil {
		t.Error("pdb nil → détecteur nil attendu")
	}
	if d := newRivalDetectorForPDB(&duckdb.PlayerDB{}); d != nil {
		t.Error("pdb.Player nil → détecteur nil attendu")
	}
}

func TestEmitRivalEncounters_NilGuards(t *testing.T) {
	ts := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	before := &PlayerSnapshot{LastMatchStartTime: ts.Add(-time.Hour)}
	after := &PlayerSnapshot{LastMatchStartTime: ts}
	em := &recordingEmitter{}

	// Détecteur nil (pdb invalide) → no-op, aucune panique.
	emitRivalEncounters(context.Background(), em, nil, "slug", before, after)
	if len(em.emitted) != 0 {
		t.Fatalf("détecteur nil → 0 émission, obtenu %d", len(em.emitted))
	}
}
