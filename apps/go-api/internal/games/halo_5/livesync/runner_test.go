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

// TestRunner_Signature : Runner satisfait le contrat DeltaRunner (sans importer
// scheduler, pour éviter le cycle scheduler→livesync).
func TestRunner_Signature(t *testing.T) {
	var _ interface {
		RunDelta(context.Context, domain.SyncOptions) (domain.SyncResult, error)
	} = (*Runner)(nil)
}
