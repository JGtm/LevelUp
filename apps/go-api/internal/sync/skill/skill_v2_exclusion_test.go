//go:build cgo

package skill

// skill_v2_exclusion_test.go — la v2 saute les matchs is_excluded (D-6, plan backlog
// 2026-10-16). Avant ce lot, seule la v1 appelait LoadExcludedMatchIDs : un match exclu
// entrait dans la note v2 et recevait sa ligne canonique.

import (
	"context"
	"testing"
	"time"
)

func TestRunLUSRV2Shadow_Canonical_SkipsExcludedMatches(t *testing.T) {

	sharedDB := openShadowTestDB(t)
	playerDB := openCanonicalPlayerTestDB(t)
	// Table lue par LoadExcludedMatchIDs ; le schéma réel la sert par la vue _latest.
	if _, err := playerDB.Exec(`CREATE TABLE player_match_enrichment_latest (match_id VARCHAR, is_excluded BOOLEAN)`); err != nil {
		t.Fatalf("DDL player_match_enrichment_latest: %v", err)
	}
	if _, err := playerDB.Exec(`INSERT INTO player_match_enrichment_latest VALUES ('m2', TRUE), ('m1', FALSE)`); err != nil {
		t.Fatalf("seed exclusion: %v", err)
	}
	for i, id := range []string{"m1", "m2", "m3"} {
		seedCanonical2v2(t, sharedDB, id, time.Date(2025, 9, 1, 12+i, 0, 0, 0, time.UTC))
	}

	processed, err := RunLUSRV2Shadow(context.Background(), playerDB, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}
	if processed != 2 {
		t.Errorf("processed = %d, attendu 2 (m2 exclu)", processed)
	}

	var rows int
	if err := playerDB.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id = 'm2'`).Scan(&rows); err != nil {
		t.Fatalf("count m2: %v", err)
	}
	if rows != 0 {
		t.Errorf("le match exclu a reçu %d ligne(s) de note, attendu 0", rows)
	}
	var experience int
	if err := sharedDB.QueryRow(
		`SELECT experience FROM player_skill_state_v2_latest WHERE xuid = 'owner'`).Scan(&experience); err != nil {
		t.Fatalf("lecture de l'état v2: %v", err)
	}
	if experience != 2 {
		t.Errorf("expérience v2 = %d, attendu 2 (le match exclu n'entre pas dans la note)", experience)
	}
}
