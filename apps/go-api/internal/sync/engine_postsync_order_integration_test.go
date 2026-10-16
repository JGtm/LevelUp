//go:build integration

package sync

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/migration"
)

// evenementsArriventClient joue le film du match d'un coéquipier : les événements ne sont
// en base qu'APRÈS la convergence events (1.54). onFetch les écrit au moment du fetch.
type evenementsArriventClient struct {
	mockHaloClient
	onFetch func(matchID string)
}

func (c *evenementsArriventClient) GetHighlightEventsChunk(_ context.Context, matchID string) ([]byte, int, bool, error) {
	if c.onFetch != nil {
		c.onFetch(matchID)
	}
	return nil, 0, false, nil
}

// TestRunPostSyncPipeline_EventsBeforeDominance : un match inséré par un coéquipier
// (events_loaded=false) voit ses événements arriver à l'étape 1.54 ; la domination, calculée
// ensuite, lit ces événements et n'est donc PAS figée à 0 (une valeur 0 est définitive : le
// marqueur « jamais calculé » est NULL). Garde l'invariant « 1.54 avant 1.7 » du réordonnancement.
func TestRunPostSyncPipeline_EventsBeforeDominance(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid, matchID = "x1", "dom-1"

	// Match Slayer gagné par l'équipe du joueur, sans aucun événement en base.
	if _, err := shared.Exec(`INSERT INTO match_registry (match_id, events_loaded, backfill_completed, start_time, game_variant_name, duration_seconds)
		VALUES (?, FALSE, 0, now(), 'Slayer', 600)`, matchID); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	for _, p := range []struct {
		xuid    string
		team    int
		outcome int
	}{{xuid, 0, 2}, {"x2", 0, 2}, {"x3", 1, 3}, {"x4", 1, 3}} {
		if _, err := shared.Exec(`INSERT INTO match_participants (match_id, xuid, team_id, outcome) VALUES (?, ?, ?, ?)`,
			matchID, p.xuid, p.team, p.outcome); err != nil {
			t.Fatalf("seed participant: %v", err)
		}
	}

	// Courbe de frags attendue : l'équipe 0 mène de bout en bout (30 frags contre 6).
	var kills []analysis.KillEvent
	for i := 0; i < 36; i++ {
		team := 0
		if i%6 == 5 {
			team = 1
		}
		kills = append(kills, analysis.KillEvent{TimeMS: int64(10_000 * (i + 1)), TeamID: team})
	}
	want := analysis.ComputeDominanceFlag(analysis.BuildScoreSnapshots(kills), 0, 2, "standard", false, false, matchID)
	if want == analysis.DominanceFlagNone {
		t.Fatalf("scénario sans domination: le test ne prouverait rien (flag attendu = %d)", want)
	}

	client := &evenementsArriventClient{}
	client.onFetch = func(id string) {
		for _, k := range kills {
			killer := xuid
			if k.TeamID == 1 {
				killer = "x3"
			}
			if _, err := shared.Exec(`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'kill', ?, ?)`,
				id, k.TimeMS, killer); err != nil {
				t.Errorf("seed event: %v", err)
			}
		}
	}

	e := &SyncEngine{gamertag: "GT", xuid: xuid, titleSlug: "halo_infinite"}
	_ = e.runPostSyncPipeline(context.Background(), player, NewPinnedSharedAccess(shared), client, nil)

	var flag sql.NullInt64
	if err := player.QueryRow(`SELECT dominance_flag FROM player_match_enrichment_latest WHERE match_id = ?`, matchID).Scan(&flag); err != nil {
		t.Fatalf("lecture dominance_flag: %v", err)
	}
	if !flag.Valid || int(flag.Int64) != want {
		t.Fatalf("dominance_flag = %v, want %d (calculée APRÈS l'arrivée des événements, pas figée à 0)", flag, want)
	}
}
