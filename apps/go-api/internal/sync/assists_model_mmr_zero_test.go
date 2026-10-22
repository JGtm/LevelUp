//go:build integration

package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/sync/testutil"
)

// Le régresseur « écart de MMR » du modèle d'assistances attendues vaut team_mmr - enemy_mmr
// quand LES DEUX MMR sont réels, et 0 (neutre) sinon. Un MMR à 0 (match classé abandonné) est
// une valeur absente (analysis.MMRKnown) : sans cette règle, « 1500 - 0 » entrerait dans la
// régression comme un écart de 1500 points et fausserait le coefficient de tout le mode.
func TestLoadAssistsSamples_MMRDeltaIgnoresZeroMMR(t *testing.T) {
	db := testutil.NewInMemoryShared(t)
	const xuid = "xuid_mmr_zero"
	for _, s := range []string{
		`INSERT INTO match_registry (match_id, start_time, start_time_utc, end_time, duration_seconds,
			game_variant_id, pair_name) VALUES
			('m_real', '2026-09-01 10:00:00', '2026-09-01 10:00:00+00', '2026-09-01 10:10:00', 600, 'aaaaaaaa-0000-0000-0000-000000000001', 'Ranked:Slayer'),
			('m_void', '2026-09-02 10:00:00', '2026-09-02 10:00:00+00', '2026-09-02 10:10:00', 600, 'aaaaaaaa-0000-0000-0000-000000000001', 'Ranked:Slayer'),
			('m_half', '2026-09-03 10:00:00', '2026-09-03 10:00:00+00', '2026-09-03 10:10:00', 600, 'aaaaaaaa-0000-0000-0000-000000000001', 'Ranked:Slayer'),
			('m_none', '2026-09-04 10:00:00', '2026-09-04 10:00:00+00', '2026-09-04 10:10:00', 600, 'aaaaaaaa-0000-0000-0000-000000000001', 'Ranked:Slayer')`,
		`INSERT INTO match_participants (match_id, xuid, team_id, outcome, kills, deaths, assists,
			damage_dealt, damage_taken, time_played_seconds, kda, accuracy, team_mmr, enemy_mmr) VALUES
			('m_real', '` + xuid + `', 0, 2, 5, 1, 2, 500, 100, 600, 1.0, 0.5, 1500, 1400),
			('m_void', '` + xuid + `', 0, 2, 5, 1, 2, 500, 100, 600, 1.0, 0.5, 0, 0),
			('m_half', '` + xuid + `', 0, 2, 5, 1, 2, 500, 100, 600, 1.0, 0.5, 1500, 0),
			('m_none', '` + xuid + `', 0, 2, 5, 1, 2, 500, 100, 600, 1.0, 0.5, NULL, NULL)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("fixture : %v\n%s", err, s)
		}
	}

	samples, err := loadAssistsSamples(context.Background(), db, xuid)
	if err != nil {
		t.Fatalf("loadAssistsSamples: %v", err)
	}
	if len(samples) != 4 {
		t.Fatalf("4 échantillons attendus, got %d", len(samples))
	}
	// Ordre SQL : ORDER BY game_variant_name (identique ici) — on indexe par le delta attendu
	// plutôt que par la position : un seul échantillon porte 100, les trois autres 0.
	var withGap, neutral int
	for _, s := range samples {
		switch s.mmrDelta {
		case 100:
			withGap++
		case 0:
			neutral++
		default:
			t.Errorf("écart de MMR inattendu %v (un MMR à 0 ne doit pas entrer comme un niveau)", s.mmrDelta)
		}
	}
	if withGap != 1 || neutral != 3 {
		t.Errorf("attendu 1 écart réel (100) et 3 neutres, got %d et %d", withGap, neutral)
	}
}
