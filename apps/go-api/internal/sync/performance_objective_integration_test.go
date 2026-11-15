//go:build integration

// performance_objective_integration_test.go — la note à objectif de bout en bout dans le batch de
// notes : un match classé de famille objectif qui arrive reçoit la chaîne `ranked_objectif` ET une
// note qui tient compte de l'objectif et du soutien de l'équipe (lignes de match_objective_stats
// de TOUS les joueurs du match, une requête pour tout l'historique), sans intervention manuelle ;
// sans lignes d'objectif (Halo 5, match jamais enrichi) la métrique est absente et la note est le
// combat renormalisé. Fixtures uniquement — aucune DB réelle n'est touchée.

package sync

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
)

// objectiveColumns : les colonnes lues par le chargeur (famille, poids, durées), toutes numériques.
func objectiveColumns() []string {
	seen := map[string]bool{}
	var cols []string
	add := func(c string) {
		if !seen[c] {
			seen[c] = true
			cols = append(cols, c)
		}
	}
	for _, c := range []string{
		"flag_grabs", "skull_scoring_ticks", "zone_scoring_ticks", "power_seed_carriers_killed",
		"successful_extractions", "kills_as_vip", narrative.ObjectiveColTimesSelectedAsVIP,
	} {
		add(c)
	}
	for _, fam := range narrative.AllObjectiveFamilies() {
		for c := range narrative.ObjectiveFamilyActionWeights[fam] {
			add(c)
		}
		for _, c := range narrative.ObjectiveFamilyHoldColumns[fam] {
			add(c)
		}
	}
	return cols
}

// enableObjectiveFixture dote la fixture de la table `match_objective_stats_latest` (la vue de
// lecture, en table pour la fixture) et de l'équipe des participants.
func enableObjectiveFixture(t *testing.T, db *sql.DB, withStats bool) {
	t.Helper()
	mustExec(t, db, `ALTER TABLE match_participants ADD COLUMN team_id INTEGER`)
	if !withStats {
		return
	}
	var defs []string
	for _, c := range objectiveColumns() {
		defs = append(defs, c+" DOUBLE")
	}
	mustExec(t, db, `CREATE TABLE match_objective_stats_latest (match_id VARCHAR, xuid VARCHAR, `+strings.Join(defs, ", ")+`)`)
}

// seedObjectiveMatch insère un match classé de Bases (`Ranked:Strongholds`) : xuid1 et un coéquipier
// (équipe 0), un adversaire (équipe 1) ; `captures` est le nombre de captures de zone de xuid1.
func seedObjectiveMatch(t *testing.T, db *sql.DB, idx int, kills, deaths, damage int, captures float64, withStats bool) string {
	t.Helper()
	mid := fmt.Sprintf("obj%04d", idx)
	ts := fmt.Sprintf("2025-02-%02dT%02d:00:00Z", (idx/24)+1, idx%24)
	mustExec(t, db,
		"INSERT INTO match_registry (match_id, start_time, pair_name, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, 'Ranked:Strongholds', TRUE, FALSE)",
		mid, ts)
	for _, p := range []struct {
		xuid string
		team int
	}{{"xuid1", 0}, {"mate", 0}, {"foe", 1}} {
		mustExec(t, db,
			`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kda, accuracy, time_played_seconds, personal_score, damage_dealt, damage_taken, rank, team_mmr, enemy_mmr, kills_expected, deaths_expected, team_id)
			 VALUES (?, ?, 2, ?, ?, 3, 1.5, 30.0, 600, ?, ?, 3000.0, 1, 1500.0, 1500.0, 10.0, 5.0, ?)`,
			mid, p.xuid, kills, deaths, 800+damage/10, float64(damage), p.team)
	}
	mustExec(t, db, `INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid)
	if withStats {
		for _, s := range []struct {
			xuid     string
			captures float64
		}{{"xuid1", captures}, {"mate", 4}, {"foe", 4}} {
			mustExec(t, db, `INSERT INTO match_objective_stats_latest (match_id, xuid, zone_captures, zone_scoring_ticks) VALUES (?, ?, ?, 0)`,
				mid, s.xuid, s.captures)
		}
	}
	return mid
}

// finalMatchRun : le match final d'une fixture rejouée et ce que le batch en a stocké.
type finalMatchRun struct {
	id      string
	score   float64
	chain   string
	present bool
}

// runFinalMatch rejoue un batch complet sur une fixture neuve : 10 matchs de référence (captures
// croissantes) puis LE match final, au combat faible. tweak (peut être nil) modifie la fixture du match
// final avant le batch.
func runFinalMatch(t *testing.T, captures float64, withStats bool, tweak func(db *sql.DB, final string)) finalMatchRun {
	t.Helper()
	db := openPerfDB(t)
	enableObjectiveFixture(t, db, withStats)
	const population = MinMatchesPerChainForRelative
	for i := 0; i < population; i++ {
		seedObjectiveMatch(t, db, i, 3+i, 20-i, 1000+300*i, float64(i), withStats)
	}
	final := seedObjectiveMatch(t, db, population, 4, 15, 1500, captures, withStats)
	if tweak != nil {
		tweak(db, final)
	}
	if _, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false); err != nil {
		t.Fatalf("batch: %v", err)
	}
	score, chain, present := storedPerfScore(t, db, final)
	return finalMatchRun{id: final, score: score, chain: chain, present: present}
}

// scoreFinalMatch rend la note et la chaîne STOCKÉES du match final ; échoue s'il n'a reçu aucune note.
func scoreFinalMatch(t *testing.T, captures float64, withStats bool) (float64, string) {
	t.Helper()
	r := runFinalMatch(t, captures, withStats, nil)
	if !r.present {
		t.Fatalf("le match final %s n'a reçu aucune note (chaîne stockée %q)", r.id, r.chain)
	}
	return r.score, r.chain
}

// Le match classé de famille objectif est rattaché à `ranked_objectif`, noté sans intervention, et
// sa note suit son objectif : le jumeau qui n'a rien capturé note moins que celui qui a tout pris,
// alors que rien d'autre ne les distingue.
func TestBatchPerformance_ObjectifEtSoutienEntrentDansLaNote(t *testing.T) {
	active, chain := scoreFinalMatch(t, 12, true)
	if chain != PerfChainRankedObjectif {
		t.Errorf("chaîne stockée = %q, attendu %q", chain, PerfChainRankedObjectif)
	}
	inactive, _ := scoreFinalMatch(t, 0, true)
	if active == inactive {
		t.Fatalf("même note (%v) avec et sans objectif : les lignes d'objectif n'entrent pas dans le calcul du batch", active)
	}
	if active <= inactive {
		t.Errorf("note du joueur actif à l'objectif = %v, jumeau inactif = %v : celui qui joue l'objectif doit noter plus haut", active, inactive)
	}
}

// Sans lignes d'objectif (titre sans match_objective_stats, match jamais enrichi) : la métrique est
// absente, la note existe quand même (combat renormalisé) et n'est ni récompensée ni pénalisée.
func TestBatchPerformance_SansLignesDObjectifLaMetriqueEstAbsente(t *testing.T) {
	uncovered, chain := scoreFinalMatch(t, 0, false)
	if chain != PerfChainRankedObjectif {
		t.Errorf("chaîne stockée = %q, attendu %q", chain, PerfChainRankedObjectif)
	}
	active, _ := scoreFinalMatch(t, 12, true)
	inactive, _ := scoreFinalMatch(t, 0, true)
	if uncovered >= active || uncovered <= inactive {
		t.Errorf("sans lignes d'objectif %v : attendu entre le jumeau inactif (%v) et le jumeau actif (%v)", uncovered, inactive, active)
	}
}

// setOutcome fixe l'issue d'un participant du match final (4 = non terminé).
func setOutcome(t *testing.T, db *sql.DB, matchID, xuid string, outcome int) {
	t.Helper()
	mustExec(t, db, "UPDATE match_participants SET outcome = ? WHERE match_id = ? AND xuid = ?", outcome, matchID, xuid)
}

// setCaptures fixe les captures de zone d'un participant du match final.
func setCaptures(t *testing.T, db *sql.DB, matchID, xuid string, captures float64) {
	t.Helper()
	mustExec(t, db, "UPDATE match_objective_stats_latest SET zone_captures = ? WHERE match_id = ? AND xuid = ?", captures, matchID, xuid)
}

// Un coéquipier parti (match non terminé) reste dans le total de l'équipe : sa part de l'objectif pèse
// sur celle du joueur exactement comme s'il avait fini, et plus il en a pris, plus la part du joueur baisse.
func TestBatchPerformance_CoequipierNonTermineCompteDansLeTotalDEquipe(t *testing.T) {
	mate := func(outcome int, captures float64) func(*sql.DB, string) {
		return func(db *sql.DB, final string) {
			setOutcome(t, db, final, "mate", outcome)
			setCaptures(t, db, final, "mate", captures)
		}
	}
	gone := runFinalMatch(t, 6, true, mate(4, 40))
	finished := runFinalMatch(t, 6, true, mate(2, 40))
	goneIdle := runFinalMatch(t, 6, true, mate(4, 0))
	for _, r := range []finalMatchRun{gone, finished, goneIdle} {
		if !r.present {
			t.Fatalf("le match final %s n'a reçu aucune note", r.id)
		}
	}
	if gone.score != finished.score {
		t.Errorf("coéquipier parti = %v, coéquipier arrivé au bout = %v : la présence à la fin ne change pas le total d'équipe", gone.score, finished.score)
	}
	if gone.score >= goneIdle.score {
		t.Errorf("coéquipier parti qui a tout pris = %v, inactif = %v : ses points doivent baisser la part du joueur", gone.score, goneIdle.score)
	}
}

// Le match non terminé DU JOUEUR n'est jamais noté, quelle que soit sa durée (nulle ou non).
func TestBatchPerformance_MatchNonTermineDuJoueurJamaisNote(t *testing.T) {
	for name, duration := range map[string]string{"durée connue": "600", "durée nulle": "0", "durée absente": "NULL"} {
		r := runFinalMatch(t, 12, true, func(db *sql.DB, final string) {
			setOutcome(t, db, final, "xuid1", 4)
			mustExec(t, db, "UPDATE match_participants SET time_played_seconds = "+duration+" WHERE match_id = ? AND xuid = 'xuid1'", final)
		})
		if r.present {
			t.Errorf("%s : le match non terminé du joueur a reçu une note (%v)", name, r.score)
		}
	}
}

// Une durée absente (NULL) ou nulle vaut perfFallbackDurationSeconds PARTOUT : le chargement SQL de
// l'historique et les calculs par minute lisent la même constante, la note est celle d'une durée égale.
func TestBatchPerformance_DureeAbsenteOuNulleVautLaDureeDeRepli(t *testing.T) {
	withDuration := func(duration string) finalMatchRun {
		return runFinalMatch(t, 8, true, func(db *sql.DB, final string) {
			mustExec(t, db, "UPDATE match_participants SET time_played_seconds = "+duration+" WHERE match_id = ? AND xuid = 'xuid1'", final)
		})
	}
	want := withDuration(fmt.Sprintf("%g", perfFallbackDurationSeconds))
	for name, duration := range map[string]string{"NULL": "NULL", "0": "0"} {
		got := withDuration(duration)
		if !got.present || !want.present {
			t.Fatalf("durée %s : note absente (%+v / %+v)", name, got, want)
		}
		if got.score != want.score {
			t.Errorf("durée %s : note %v, durée de repli %v : le repli doit être le même partout", name, got.score, want.score)
		}
	}
}
