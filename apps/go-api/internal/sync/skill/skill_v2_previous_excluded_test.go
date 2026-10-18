//go:build cgo

package skill

// skill_v2_previous_excluded_test.go — le « précédent » du chaînage (delta de note,
// hystérésis du palier affiché) est le dernier match NON exclu du groupe, lu par la vue
// match_skill_rank_latest_by_type. Une exclusion ne retire pas la ligne déjà écrite du
// match exclu (table append-only) : sans filtre, le match suivant la prenait pour
// précédent (revue adversariale E10.1, constat C1).

import (
	"context"
	"testing"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// TestRunLUSRV2Shadow_Rejeu_PrecedentIgnoreLeMatchExclu : trois matchs notés, puis m2
// est exclu et le joueur est rejoué. Le delta de m3 se mesure contre m1, jamais contre
// l'ancienne ligne de m2 (toujours présente dans la table).
func TestRunLUSRV2Shadow_Rejeu_PrecedentIgnoreLeMatchExclu(t *testing.T) {
	sharedDB := openShadowTestDB(t)
	playerDB := openCanonicalPlayerTestDB(t)
	if _, err := playerDB.Exec(`CREATE TABLE player_match_enrichment_latest (match_id VARCHAR, is_excluded BOOLEAN)`); err != nil {
		t.Fatalf("DDL player_match_enrichment_latest: %v", err)
	}
	seedShadowFixtures(t, sharedDB,
		shadowFixture{"m1", pairSlayer, fixtureAt(0), seats2v2(3, 5, 15)},
		shadowFixture{"m2", pairSlayer, fixtureAt(10), seats2v2(3, 4, 16)},
		shadowFixture{"m3", pairSlayer, fixtureAt(20), seats2v2(2, 15, 5)})
	ctx := context.Background()
	acc := newPinnedSharedAccessor(sharedDB)
	if n, err := RunLUSRV2ShadowOwnerOnly(ctx, playerDB, acc, "owner"); err != nil || n != 3 {
		t.Fatalf("première passe : processed=%d err=%v, attendu 3", n, err)
	}

	// Exclusion de m2, puis rejeu complet (remise à zéro du filigrane, comme le recalcul
	// d'exclusion).
	if _, err := playerDB.Exec(`INSERT INTO player_match_enrichment_latest VALUES ('m2', TRUE)`); err != nil {
		t.Fatalf("seed exclusion: %v", err)
	}
	if _, err := sharedDB.Exec(`DELETE FROM player_skill_state_v2`); err != nil {
		t.Fatalf("reset filigrane: %v", err)
	}
	if n, err := RunLUSRV2ShadowOwnerOnly(ctx, playerDB, acc, "owner"); err != nil || n != 2 {
		t.Fatalf("rejeu : processed=%d err=%v, attendu 2 (m2 exclu)", n, err)
	}

	var m1Rating, m3Rating float64
	var m3Delta *float64
	if err := playerDB.QueryRow(`SELECT rating_value FROM match_skill_rank_latest_by_type
		WHERE match_id = 'm1' AND rating_type = 'LUSR'`).Scan(&m1Rating); err != nil {
		t.Fatalf("lecture m1 : %v", err)
	}
	if err := playerDB.QueryRow(`SELECT rating_value, rating_delta FROM match_skill_rank_latest_by_type
		WHERE match_id = 'm3' AND rating_type = 'LUSR'`).Scan(&m3Rating, &m3Delta); err != nil {
		t.Fatalf("lecture m3 : %v", err)
	}
	if m3Delta == nil {
		t.Fatal("m3 sans delta : un précédent (m1) existe")
	}
	if want := m3Rating - m1Rating; *m3Delta < want-1e-3 || *m3Delta > want+1e-3 {
		t.Errorf("delta(m3) = %v, attendu %v (m3 - m1) : le précédent est le dernier match non exclu", *m3Delta, want)
	}
}

// TestLoadPrevious_IgnoreLesMatchsExclus : les deux chargeurs sautent les matchs exclus
// et rendent le plus récent restant ; aucun restant : nil / -1.
func TestLoadPrevious_IgnoreLesMatchsExclus(t *testing.T) {
	playerDB := openCanonicalPlayerTestDB(t)
	ctx := context.Background()
	if _, err := playerDB.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, tier, sub_tier, playlist_group, start_time, written_at) VALUES
		('m1', 'LUSR', 1500, 'Gold', 3, 'arena_slayer', TIMESTAMP '2025-01-01 12:00:00', TIMESTAMP '2025-01-01 12:00:00'),
		('m2', 'LUSR', 1600, 'Gold', 5, 'arena_slayer', TIMESTAMP '2025-01-01 13:00:00', TIMESTAMP '2025-01-01 13:00:00')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	t3 := time.Date(2025, 1, 1, 14, 0, 0, 0, time.UTC)
	q := previousQuery{group: "arena_slayer", matchID: "m3", start: t3, excluded: map[string]bool{"m2": true}}
	if prev := loadPreviousLUSRRating(ctx, playerDB, q); prev == nil || *prev != 1500 {
		t.Errorf("prev(m3, m2 exclu) = %v, attendu 1500 (m1)", prev)
	}
	bnd := skillv2.DefaultTierBoundaries()
	if got, want := loadPreviousDisplayedOrdinal(ctx, playerDB, q, bnd), skillv2.TierOrdinal(bnd, "Gold", 3); got != want {
		t.Errorf("ordinal(m3, m2 exclu) = %d, attendu %d (m1 Gold 3)", got, want)
	}
	q.excluded = map[string]bool{"m1": true, "m2": true}
	if prev := loadPreviousLUSRRating(ctx, playerDB, q); prev != nil {
		t.Errorf("prev(m3, tout exclu) = %v, attendu nil", *prev)
	}
	if got := loadPreviousDisplayedOrdinal(ctx, playerDB, q, bnd); got != -1 {
		t.Errorf("ordinal(m3, tout exclu) = %d, attendu -1", got)
	}
}
