//go:build integration

// performance_cleanup_integration_test.go — lot 2 de .ai/PLAN_PERF_NOTE_OBJECTIFS.md
// (B2.2 tests du batch auto-nettoyant, B2.3 garde-rail pérenne « aucune note sur
// un match non terminé »).
//
// Décisions couvertes : D-D (purge sèche des notes orphelines : score ET chaîne à
// NULL) et D-E (le nettoyage tourne à CHAQUE run, force compris).
//
// Fixtures uniquement — aucune DB réelle n'est touchée par ces tests.

package sync

import (
	"database/sql"
	"fmt"
	"testing"
)

// ── Helpers de fixture ──────────────────────────────────────────────────────

// seedPerfMatchOutcome insère 1 match complet (registry + participation + row
// enrichment socle) avec un outcome explicite — outcome=4 = non terminé (DNF),
// que loadHistoryForPerf exclut de l'univers de calcul.
func seedPerfMatchOutcome(t *testing.T, db *sql.DB, idx int, pairName string, isRanked, isFirefight bool, outcome int) string {
	t.Helper()
	mid := fmt.Sprintf("m%04d", idx)
	ts := fmt.Sprintf("2025-01-%02dT%02d:00:00Z", (idx/24)+1, idx%24)
	mustExec(t, db,
		"INSERT INTO match_registry (match_id, start_time, pair_name, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, ?, ?, ?)",
		mid, ts, pairName, isRanked, isFirefight)
	mustExec(t, db,
		`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kda, accuracy, time_played_seconds, personal_score, damage_dealt, damage_taken, rank, team_mmr, enemy_mmr, kills_expected, deaths_expected)
		 VALUES (?, 'xuid1', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mid, outcome, 10, 5, 3, 1.5, 0.5, 600, 1000, 2000.0, 500.0, 1, 1500.0, 1500.0, 10.0, 5.0)
	mustExec(t, db, `INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid)
	return mid
}

// seedStoredPerfScore pose une note DÉJÀ STOCKÉE sur un match : une row partielle
// stage='perf' (le stage propriétaire des colonnes performance_*), exactement ce
// que le batch écrit en production. chain vide = note legacy d'avant les chaînes.
func seedStoredPerfScore(t *testing.T, db *sql.DB, matchID string, score float64, chain string) {
	t.Helper()
	if chain == "" {
		mustExec(t, db,
			`INSERT INTO player_match_enrichment (match_id, performance_score, stage) VALUES (?, ?, 'perf')`,
			matchID, score)
		return
	}
	mustExec(t, db,
		`INSERT INTO player_match_enrichment (match_id, performance_score, performance_chain, stage) VALUES (?, ?, ?, 'perf')`,
		matchID, score, chain)
}

// seedExcludedFlag marque un match is_excluded (row partielle stage='exclusion',
// comme match_exclusion_service en production).
func seedExcludedFlag(t *testing.T, db *sql.DB, matchID string) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO player_match_enrichment (match_id, is_excluded, stage) VALUES (?, TRUE, 'exclusion')`,
		matchID)
}

// storedPerfScore lit la note COURANTE d'un match par la vue merge-on-read.
// present=false quand score ET chaîne sont NULL (note nettoyée ou jamais posée).
func storedPerfScore(t *testing.T, db *sql.DB, matchID string) (score float64, chain string, present bool) {
	t.Helper()
	var s sql.NullFloat64
	var c sql.NullString
	err := db.QueryRow(
		`SELECT performance_score, performance_chain FROM player_match_enrichment_latest WHERE match_id = ?`,
		matchID).Scan(&s, &c)
	if err == sql.ErrNoRows {
		return 0, "", false
	}
	if err != nil {
		t.Fatalf("lecture de la note stockée (%s): %v", matchID, err)
	}
	return s.Float64, c.String, s.Valid
}

// countNulledPerfRows compte les rows d'ANNULATION écrites par le nettoyage
// (stage='perf' avec un score NULL). Sert de témoin d'idempotence : un 2e run ne
// doit pas en ajouter une seule.
func countNulledPerfRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment WHERE stage = 'perf' AND performance_score IS NULL`).Scan(&n); err != nil {
		t.Fatalf("count des rows d'annulation: %v", err)
	}
	return n
}

func countScoredMatches(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`).Scan(&n); err != nil {
		t.Fatalf("count des matchs notés: %v", err)
	}
	return n
}

// seedCleanupFixture construit l'univers commun aux tests de nettoyage :
//
//	m0000..m0011 : 12 matchs arena_slayer terminés → 2 derniers scorables ;
//	m0100        : DNF (outcome=4) PORTANT une note stockée → doit être NULLée ;
//	m0200..m0202 : 3 matchs BTB (sous le seuil de 10), le dernier PORTE une note
//	               legacy (chaîne vide) → doit être NULLée ;
//	m0300        : match arena_slayer EXCLU portant une note → doit être NULLée.
//
// Retourne les 3 match_ids attendus comme nettoyés.
func seedCleanupFixture(t *testing.T, db *sql.DB) (dnf, below, excluded string) {
	t.Helper()
	seedPerfMatchesWithChain(t, db, 0, 12, "Arena:Slayer", false, false)

	dnf = seedPerfMatchOutcome(t, db, 100, "Arena:Slayer", false, false, 4)
	seedStoredPerfScore(t, db, dnf, 61.5, LUSRChainArenaSlayer)

	seedPerfMatchesWithChain(t, db, 200, 3, "BTB:Slayer", false, false)
	below = "m0202"
	seedStoredPerfScore(t, db, below, 47.0, "")

	excluded = seedPerfMatchOutcome(t, db, 300, "Arena:Slayer", false, false, 2)
	seedExcludedFlag(t, db, excluded)
	seedStoredPerfScore(t, db, excluded, 72.3, LUSRChainArenaSlayer)

	return dnf, below, excluded
}

// ── B2.2 — les 4 causes de nettoyage, en mode normal ────────────────────────

// TestBatchPerformance_CleansOrphanScores_NotForce couvre (a) DNF, (b) sous-seuil
// avec note legacy, (c) match qualifié conservé, (d) match exclu — en mode !force.
func TestBatchPerformance_CleansOrphanScores_NotForce(t *testing.T) {
	db := openPerfDB(t)
	dnf, below, excluded := seedCleanupFixture(t, db)

	n, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if n != 2 {
		t.Fatalf("matchs notés = %d, attendu 2 (12 arena_slayer − seuil 10)", n)
	}

	// (a) DNF : la note stockée disparaît, chaîne comprise (purge sèche D-D).
	if _, chain, present := storedPerfScore(t, db, dnf); present {
		t.Errorf("(a) DNF %s : note toujours présente (chaîne=%q), attendue NULLée", dnf, chain)
	}
	// (b) sous-seuil de chaîne avec note legacy.
	if _, _, present := storedPerfScore(t, db, below); present {
		t.Errorf("(b) sous-seuil %s : note toujours présente, attendue NULLée", below)
	}
	// (d) match exclu.
	if _, _, present := storedPerfScore(t, db, excluded); present {
		t.Errorf("(d) exclu %s : note toujours présente, attendue NULLée", excluded)
	}
	// (c) les matchs qualifiés gardent leur note ET leur chaîne.
	if got := countScoredMatches(t, db); got != 2 {
		t.Errorf("(c) matchs notés après nettoyage = %d, attendu 2", got)
	}
	if _, chain, present := storedPerfScore(t, db, "m0011"); !present || chain != LUSRChainArenaSlayer {
		t.Errorf("(c) m0011 : note=%v chaîne=%q, attendu une note en %q", present, chain, LUSRChainArenaSlayer)
	}
	if got := countNulledPerfRows(t, db); got != 3 {
		t.Errorf("rows d'annulation écrites = %d, attendu 3 (DNF + sous-seuil + exclu)", got)
	}
}

// TestBatchPerformance_CleansOrphanScores_Force : (f) le nettoyage tourne AUSSI en
// mode force — le chargement des notes stockées ne doit plus être conditionné à
// !force (D-E).
func TestBatchPerformance_CleansOrphanScores_Force(t *testing.T) {
	db := openPerfDB(t)
	dnf, below, excluded := seedCleanupFixture(t, db)

	n, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, true)
	if err != nil {
		t.Fatalf("batch force: %v", err)
	}
	if n != 2 {
		t.Fatalf("mode force : matchs notés = %d, attendu 2", n)
	}
	for _, mid := range []string{dnf, below, excluded} {
		if _, _, present := storedPerfScore(t, db, mid); present {
			t.Errorf("mode force : note orpheline %s non nettoyée", mid)
		}
	}
	if got := countScoredMatches(t, db); got != 2 {
		t.Errorf("mode force : matchs notés après nettoyage = %d, attendu 2", got)
	}
}

// TestBatchPerformance_CleanupIsIdempotent : (e) un 2e run ne re-NULLe rien — une
// note déjà NULLée ne matche plus `performance_score IS NOT NULL`, donc aucune
// nouvelle row d'annulation n'est écrite (pas de croissance non bornée de la table
// append-only).
func TestBatchPerformance_CleanupIsIdempotent(t *testing.T) {
	db := openPerfDB(t)
	seedCleanupFixture(t, db)

	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("1er run: %v", err)
	}
	after1 := countNulledPerfRows(t, db)
	scored1 := countScoredMatches(t, db)
	if after1 != 3 {
		t.Fatalf("1er run : rows d'annulation = %d, attendu 3", after1)
	}

	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("2e run: %v", err)
	}
	if after2 := countNulledPerfRows(t, db); after2 != after1 {
		t.Errorf("2e run : rows d'annulation = %d, attendu %d (aucune écriture)", after2, after1)
	}
	if scored2 := countScoredMatches(t, db); scored2 != scored1 {
		t.Errorf("2e run : matchs notés = %d, attendu %d (stabilité)", scored2, scored1)
	}

	// 3e run en force : recalcule les notes, ne rajoute aucune annulation.
	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, true); err != nil {
		t.Fatalf("3e run (force): %v", err)
	}
	if after3 := countNulledPerfRows(t, db); after3 != after1 {
		t.Errorf("3e run (force) : rows d'annulation = %d, attendu %d", after3, after1)
	}
}

// TestBatchPerformance_CleansScoreWhenChainFallsBelowThreshold : la note d'un match
// QUALIFIÉ hier, dont la chaîne repasse sous le seuil aujourd'hui (exclusions en
// amont), est nettoyée — c'est le cas où la chaîne stockée est pourtant la BONNE.
// Sans la priorité du seuil sur le skip, ce match serait considéré qualifié et sa
// note survivrait à tort (D-D).
func TestBatchPerformance_CleansScoreWhenChainFallsBelowThreshold(t *testing.T) {
	db := openPerfDB(t)
	seedPerfMatchesWithChain(t, db, 0, 12, "Arena:Slayer", false, false)

	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("1er run: %v", err)
	}
	if got := countScoredMatches(t, db); got != 2 {
		t.Fatalf("1er run : matchs notés = %d, attendu 2", got)
	}

	// L'utilisateur exclut 4 matchs : la chaîne tombe à 8 < 10.
	for i := 0; i < 4; i++ {
		seedExcludedFlag(t, db, fmt.Sprintf("m%04d", i))
	}

	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("2e run: %v", err)
	}
	if got := countScoredMatches(t, db); got != 0 {
		t.Errorf("après exclusions : matchs notés = %d, attendu 0 (chaîne sous le seuil)", got)
	}
}

// TestBatchPerformance_CleansEverythingWhenUniverseEmpty : cas dégénéré — tous les
// matchs sont exclus (le batch sort par son retour anticipé). Les notes stockées
// sont TOUTES orphelines et doivent quand même être nettoyées.
func TestBatchPerformance_CleansEverythingWhenUniverseEmpty(t *testing.T) {
	db := openPerfDB(t)
	seedPerfMatchesWithChain(t, db, 0, 12, "Arena:Slayer", false, false)
	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("1er run: %v", err)
	}
	if got := countScoredMatches(t, db); got != 2 {
		t.Fatalf("1er run : matchs notés = %d, attendu 2", got)
	}

	for i := 0; i < 12; i++ {
		seedExcludedFlag(t, db, fmt.Sprintf("m%04d", i))
	}

	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("2e run (univers vide): %v", err)
	}
	if got := countScoredMatches(t, db); got != 0 {
		t.Errorf("univers vide : matchs notés = %d, attendu 0", got)
	}
}

// ── B2.3 — garde-rail pérenne ───────────────────────────────────────────────

// TestBatchPerformance_NoStoredScoreSurvivesOnUnfinishedMatch est le garde-rail
// permanent du lot 2 (B2.3 de .ai/PLAN_PERF_NOTE_OBJECTIFS.md) : après un run de
// batch, AUCUNE ligne de player_match_enrichment_latest ne porte de
// performance_score si la participation du joueur est un non-terminé (outcome=4).
//
// L'invariant est vérifié par une jointure directe registry × participants, pas
// par la liste des match_ids attendus : il tiendra même si la fixture évolue.
// Il est vérifié dans les DEUX modes (le nettoyage tourne à chaque run — D-E).
func TestBatchPerformance_NoStoredScoreSurvivesOnUnfinishedMatch(t *testing.T) {
	assertNoScoreOnDNF := func(t *testing.T, db *sql.DB, when string) {
		t.Helper()
		var n int
		if err := db.QueryRow(`
			SELECT COUNT(*)
			  FROM player_match_enrichment_latest pme
			  JOIN match_participants mp ON mp.match_id = pme.match_id
			 WHERE pme.performance_score IS NOT NULL
			   AND mp.xuid = 'xuid1'
			   AND COALESCE(mp.outcome, 0) = 4`).Scan(&n); err != nil {
			t.Fatalf("%s: requête d'invariant: %v", when, err)
		}
		if n != 0 {
			t.Errorf("%s: %d note(s) survivent sur un match non terminé (outcome=4) — invariant D-D violé", when, n)
		}
	}

	for _, force := range []bool{false, true} {
		name := "mode_normal"
		if force {
			name = "mode_force"
		}
		t.Run(name, func(t *testing.T) {
			db := openPerfDB(t)
			seedPerfMatchesWithChain(t, db, 0, 12, "Arena:Slayer", false, false)

			// 3 DNF portant chacun une note stockée (l'état constaté en production
			// avant ce lot : 33 notes sur DNF pour le seul JGtm).
			for i := 0; i < 3; i++ {
				mid := seedPerfMatchOutcome(t, db, 100+i, "Arena:Slayer", false, false, 4)
				seedStoredPerfScore(t, db, mid, 55.0+float64(i), LUSRChainArenaSlayer)
			}
			var before int
			if err := db.QueryRow(`
				SELECT COUNT(*) FROM player_match_enrichment_latest pme
				  JOIN match_participants mp ON mp.match_id = pme.match_id
				 WHERE pme.performance_score IS NOT NULL AND COALESCE(mp.outcome, 0) = 4`).Scan(&before); err != nil {
				t.Fatal(err)
			}
			if before != 3 {
				t.Fatalf("fixture invalide : %d notes sur DNF avant le run, attendu 3", before)
			}

			if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, force); err != nil {
				t.Fatalf("batch (force=%v): %v", force, err)
			}
			assertNoScoreOnDNF(t, db, fmt.Sprintf("après run force=%v", force))
		})
	}
}
