//go:build integration

// Package duckdb — match_view_kv_pairs_bot_test.go : le scan Q20 survit aux lignes de BOT.
//
// RÉGRESSION COUVERTE (2026-08-03 → 2026-09-02) : la bascule des lecteurs sur la canonique
// `match_kill_events_latest` a introduit des xuid NULL (bots), que le scan recevait dans des
// `string` nus — la PREMIÈRE ligne de bot faisait échouer tout le chargement, et l'erreur
// était avalée en best-effort : section Duels vide sur 245 matchs Infinite qui avaient la
// donnée. Le scan passe par sql.NullString (NULL → "") ; les agrégateurs écartent "".
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/killscope"
)

// insertKVKill pose une mort dans match_kill_events avec les DEUX xuids contrôlables
// ("" = NULL = bot). insertKill (fixture killsource) fige victim_xuid à NULL — ici le
// scénario exige aussi des paires joueur↔joueur complètes.
func insertKVKill(t *testing.T, pdb *PlayerDB, killerXUID, killerGT, victimXUID, victimGT string, timeMS int) {
	t.Helper()
	var killer any
	if killerXUID != "" {
		killer = killerXUID
	}
	var victim any
	if victimXUID != "" {
		victim = victimXUID
	}
	_, err := pdb.Shared.Exec(context.Background(), `
		INSERT INTO match_kill_events
			(match_id, decode_pass, decoder_rev, publishable, time_ms,
			 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
			 feed_present, assist_known, read_path, read_origin)
		VALUES (?, 'pass_kv', 'rev_test', TRUE, ?, ?, ?, ?, ?, TRUE, FALSE, ?, 'credit-concordant')`,
		kscMatchID, timeMS, victimGT, victim, killerGT, killer, killscope.ReadPathFilmWalk)
	if err != nil {
		t.Fatalf("insert kv kill: %v", err)
	}
}

func TestGetMatchKVPairs_LignesDeBotNeCassentPasLeScan(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)

	// Une paire joueur↔joueur, une mort infligée PAR un bot (killer NULL), une mort DE bot
	// (victim NULL) : avant le correctif, la ligne 2 faisait échouer le scan entier.
	insertKVKill(t, pdb, "2533274000000001", "Alice", "2533274000000002", "Bob", 1000)
	insertKVKill(t, pdb, "", "343 Oscar [bot]", "2533274000000001", "Alice", 2000)
	insertKVKill(t, pdb, "2533274000000002", "Bob", "", "343 Guilty [bot]", 3000)

	pairs, err := repo.GetMatchKVPairs(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("GetMatchKVPairs: %v", err)
	}
	if len(pairs) != 3 {
		t.Fatalf("attendu 3 lignes (les lignes de bot sont SERVIES, pas jetées), obtenu %d", len(pairs))
	}
	// NULL → "" côté Go, gamertags intacts (le journal nomme les bots).
	if pairs[1].KillerXUID != "" || pairs[1].KillerGT != "343 Oscar [bot]" {
		t.Errorf("ligne bot-tueur inattendue : %+v", pairs[1])
	}
	if pairs[2].VictimXUID != "" || pairs[2].VictimGT != "343 Guilty [bot]" {
		t.Errorf("ligne bot-victime inattendue : %+v", pairs[2])
	}
	if pairs[0].KillerXUID != "2533274000000001" || pairs[0].VictimXUID != "2533274000000002" {
		t.Errorf("paire joueur↔joueur inattendue : %+v", pairs[0])
	}
}
