//go:build integration

// performance_ospm_integration_test.go — lot 3 de .ai/PLAN_PERF_NOTE_OBJECTIFS.md
// (B3.1 loader, B3.5 test « futur match »).
//
// EXIGENCE UTILISATEUR du 2026-08-27 : un match d'objectif classé qui arrive APRÈS
// ce chantier doit ressortir du batch avec la chaîne `ranked_objectif` ET une note
// calculée AVEC la participation à l'objectif, sans aucune intervention manuelle.
//
// Le chemin de production est le même que celui exercé ici : le post-sync appelle
// batchComputePerformanceScores (internal/sync/engine_postsync_scoring.go:83), qui
// charge lui-même les awards depuis la DB joueur — d'où l'absence de changement de
// signature, et donc la couverture des 5 call-sites du batch.
//
// Fixtures uniquement — aucune DB réelle n'est touchée par ces tests.

package sync

import (
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/migration"
)

// awardCategoryObjective — la catégorie que le loader somme. Le littéral est celui
// de la source (Personal Scores API) ; la constante de production vit dans
// internal/sync/skill (non exportée), on la recopie ici côté test.
const awardCategoryObjective = "objective"

// enablePSAFixture dote la DB de fixture de personal_score_awards ET de sa vue
// `_latest`, exactement comme le boot d'une player DB : DDL canonique
// (migration.PlayerPersonalScoreAwardsDDL) puis conversion append-only idempotente
// qui (re)crée la vue. Sans cet appel, la table existe sans la vue et le loader
// dégrade — c'est d'ailleurs le cas de TOUTES les autres fixtures de perf, qui
// vérifient ainsi le chemin « pas de PSA » à chaque run.
func enablePSAFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := execScript(t.Context(), db, migration.PlayerPersonalScoreAwardsDDL); err != nil {
		t.Fatalf("DDL personal_score_awards: %v", err)
	}
	if err := migration.EnsurePersonalScoreAwardsAppendOnly(db); err != nil {
		t.Fatalf("EnsurePersonalScoreAwardsAppendOnly: %v", err)
	}
}

// seedAward insère une ligne d'award pour xuid1 dans une génération donnée.
func seedAward(t *testing.T, db *sql.DB, matchID, category string, score int, generation int) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO personal_score_awards (match_id, xuid, award_name, award_category, award_count, award_score, generation_id, is_tombstone)
		 VALUES (?, 'xuid1', ?, ?, 1, ?, ?, FALSE)`,
		matchID, category+"_award", category, score, generation)
}

// seedRankedObjectifMatch insère un match classé de sous-mode objectif
// (`Ranked:Strongholds` + is_ranked) avec des stats de combat pilotées, plus la row
// enrichment socle posée à l'ingestion en production.
func seedRankedObjectifMatch(t *testing.T, db *sql.DB, idx, kills, deaths, damage, accuracy int) string {
	t.Helper()
	mid := fmt.Sprintf("obj%04d", idx)
	ts := fmt.Sprintf("2025-02-%02dT%02d:00:00Z", (idx/24)+1, idx%24)
	mustExec(t, db,
		"INSERT INTO match_registry (match_id, start_time, pair_name, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, 'Ranked:Strongholds', TRUE, FALSE)",
		mid, ts)
	mustExec(t, db,
		`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kda, accuracy, time_played_seconds, personal_score, damage_dealt, damage_taken, rank, team_mmr, enemy_mmr, kills_expected, deaths_expected)
		 VALUES (?, 'xuid1', 2, ?, ?, 3, 1.5, ?, 600, ?, ?, 3000.0, 1, 1500.0, 1500.0, 10.0, 5.0)`,
		mid, kills, deaths, float64(accuracy), 800+damage/10, float64(damage))
	mustExec(t, db, `INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid)
	return mid
}

// finalMatchPSA — la seule différence entre les matchs jumeaux du test B3.5.
type finalMatchPSA int

const (
	psaObjectiveActive finalMatchPSA = iota // couvert, forte activité à l'objectif
	psaCombatOnly                           // couvert, AUCUN point d'objectif
	psaAbsent                               // aucune ligne PSA : match non couvert
)

// seedFutureMatchFixture bâtit l'univers du test : 10 matchs classés d'objectif
// (population de référence, couverts PSA avec des points d'objectif croissants et un
// combat étalé), puis LE match final — le « futur match » — au combat faible.
// Retourne son match_id.
func seedFutureMatchFixture(t *testing.T, db *sql.DB, variant finalMatchPSA) string {
	t.Helper()
	enablePSAFixture(t, db)

	const population = MinMatchesPerChainForRelative // 10 : le seuil de la chaîne
	for i := 0; i < population; i++ {
		mid := seedRankedObjectifMatch(t, db, i, 3+i, 20-i, 1000+300*i, 25+i)
		seedAward(t, db, mid, awardCategoryObjective, 50+i*10, 1)
		seedAward(t, db, mid, "combat", 500, 1)
	}

	final := seedRankedObjectifMatch(t, db, population, 4, 15, 1500, 30)
	switch variant {
	case psaObjectiveActive:
		seedAward(t, db, final, awardCategoryObjective, 400, 1)
		seedAward(t, db, final, "combat", 500, 1)
	case psaCombatOnly:
		seedAward(t, db, final, "combat", 500, 1)
	case psaAbsent:
		// Aucune ligne : le match n'a pas de couverture PSA.
	}
	return final
}

// scoreFutureMatch joue un batch complet sur une fixture neuve et rend la note et la
// chaîne STOCKÉES du match final.
func scoreFutureMatch(t *testing.T, variant finalMatchPSA) (float64, string) {
	t.Helper()
	db := openPerfDB(t)
	final := seedFutureMatchFixture(t, db, variant)
	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("batch: %v", err)
	}
	score, chain, present := storedPerfScore(t, db, final)
	if !present {
		t.Fatalf("le futur match %s n'a reçu aucune note (chaîne stockée %q)", final, chain)
	}
	return score, chain
}

// TestBatchPerformance_FutureRankedObjectiveMatchIsScoredWithOSPM est le test B3.5.
//
// Il vérifie les trois moitiés de l'exigence :
//  1. le match classé d'objectif est bien rattaché à `ranked_objectif` (scission D-A)
//     et noté, sans intervention ;
//  2. sa note INTÈGRE ospm — prouvé par le match JUMEAU, identique en tout point sauf
//     ses awards d'objectif : si la métrique n'était pas branchée, les deux notes
//     seraient égales (rien d'autre ne les distingue) ;
//  3. le sens est le bon : celui qui joue l'objectif note plus haut.
func TestBatchPerformance_FutureRankedObjectiveMatchIsScoredWithOSPM(t *testing.T) {
	active, chain := scoreFutureMatch(t, psaObjectiveActive)
	if chain != PerfChainRankedObjectif {
		t.Errorf("chaîne stockée = %q, attendu %q (le futur match doit adopter le nouveau régime seul)",
			chain, PerfChainRankedObjectif)
	}

	inactive, inactiveChain := scoreFutureMatch(t, psaCombatOnly)
	if inactiveChain != PerfChainRankedObjectif {
		t.Errorf("jumeau : chaîne stockée = %q, attendu %q", inactiveChain, PerfChainRankedObjectif)
	}
	if active == inactive {
		t.Fatalf("le match actif à l'objectif et son jumeau inactif ont la MÊME note (%v) : "+
			"ospm n'entre pas dans le calcul du batch", active)
	}
	if active <= inactive {
		t.Errorf("note du match actif à l'objectif = %v, celle du jumeau inactif = %v — "+
			"celui qui joue l'objectif doit noter plus haut", active, inactive)
	}

	// Troisième variante : sans AUCUNE donnée PSA, la métrique est absente et son
	// poids redistribué. La note doit se situer entre les deux (ni récompense, ni
	// pénalité) — et surtout différer du jumeau couvert-à-0, sinon « pas de données »
	// et « rien fait » seraient confondus (D-J).
	uncovered, _ := scoreFutureMatch(t, psaAbsent)
	if uncovered == inactive {
		t.Errorf("match non couvert (%v) et match couvert sans point d'objectif (%v) : "+
			"mêmes notes, les deux cas sont confondus", uncovered, inactive)
	}
}

// TestLoadObjectiveParticipation_CoverageAndDedup couvre le loader B3.1 sur les
// quatre cas qui décident de la valeur : dédup par génération via la vue `_latest`,
// couverture sans point d'objectif, tombstone, et lignes d'un autre joueur.
func TestLoadObjectiveParticipation_CoverageAndDedup(t *testing.T) {
	db := openPerfDB(t)
	enablePSAFixture(t, db)

	// m1 : ré-extraction — la génération 2 REMPLACE la 1 (et non s'y ajoute).
	seedAward(t, db, "m1", awardCategoryObjective, 400, 1)
	seedAward(t, db, "m1", awardCategoryObjective, 10, 2)
	seedAward(t, db, "m1", "combat", 700, 2)
	// m2 : couvert, mais aucun award d'objectif → 0 est une VALEUR.
	seedAward(t, db, "m2", "combat", 500, 1)
	// m3 : extraction vide → tombstone en dernière génération → non couvert.
	seedAward(t, db, "m3", awardCategoryObjective, 300, 1)
	mustExec(t, db,
		`INSERT INTO personal_score_awards (match_id, xuid, award_name, award_category, award_count, award_score, generation_id, is_tombstone)
		 VALUES ('m3', 'xuid1', '', NULL, 0, 0, 2, TRUE)`)
	// m4 : awards d'un AUTRE joueur → invisible pour xuid1 (filtre xuid strict,
	// aligné sur le lecteur canonique platform/duckdb/personal_score_awards_repo.go).
	mustExec(t, db,
		`INSERT INTO personal_score_awards (match_id, xuid, award_name, award_category, award_count, award_score, generation_id, is_tombstone)
		 VALUES ('m4', 'xuid_autre', 'objective_award', 'objective', 1, 250, 1, FALSE)`)

	got := loadObjectiveParticipation(t.Context(), db, "xuid1")

	if v, ok := got["m1"]; !ok || v == nil || *v != 10 {
		t.Errorf("m1 : %v (présent=%v), attendu 10 — seule la dernière génération compte", deref(v), ok)
	}
	if v, ok := got["m2"]; !ok || v == nil || *v != 0 {
		t.Errorf("m2 : %v (présent=%v), attendu une présence à 0 (couvert, aucun point d'objectif)", deref(v), ok)
	}
	if v, ok := got["m3"]; ok {
		t.Errorf("m3 : présent avec %v, attendu ABSENT (dernière génération tombstonée = plus de couverture)", deref(v))
	}
	if v, ok := got["m4"]; ok {
		t.Errorf("m4 : présent avec %v, attendu ABSENT (awards d'un autre joueur)", deref(v))
	}
	if len(got) != 2 {
		t.Errorf("map de couverture : %d entrées, attendu 2 (m1, m2)", len(got))
	}
}

// TestLoadObjectiveParticipation_NoTableDegradesGracefully : sur une DB sans
// personal_score_awards (autre titre, DB legacy), le loader rend une map vide sans
// erreur — les notes sont alors calculées sans ospm, comme avant ce lot.
func TestLoadObjectiveParticipation_NoTableDegradesGracefully(t *testing.T) {
	db := openPerfDB(t) // volontairement SANS enablePSAFixture
	got := loadObjectiveParticipation(t.Context(), db, "xuid1")
	if len(got) != 0 {
		t.Errorf("map de couverture = %v, attendue vide (table absente)", got)
	}
}

func deref(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}
