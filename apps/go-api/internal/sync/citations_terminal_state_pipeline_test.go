//go:build integration

// Package sync — citations_terminal_state_pipeline_test.go : état terminal des
// citations, de bout en bout à travers BackfillMatchCitations sur la fixture de
// pipeline (jeton posé, sortie du pool, rattrapage réel par force=true,
// non-régression du chemin « events présents », inertie sur schéma non migré).
//
// La décision isolée (readEventsEmpty, isEventsEmptyDefinitive — citations.go) est
// couverte par citations_terminal_state_test.go.
//
// OÙ CES TESTS TOURNENT : job CI couverture/intégration (CGO_ENABLED=1,
// -tags=integration ./...), check_test_baseline.sh, et en local. Le job CI `unit`
// (CGO_ENABLED=0, domain/analysis/contracttest) ne les exécute PAS.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	termPendingMatch = "match-terminal-film-attendu"
	termHealedMatch  = "match-terminal-events-ok"
)

// addEventsVerdictColumns ajoute au registre de la fixture les deux colonnes que
// le pipeline events utilise pour rendre son verdict. buildSharedDDL ne les porte
// PAS : sans events_loaded, isEventsLoaded répond true par best-effort et la
// branche testée n'est jamais atteinte ; sans events_empty, la nouvelle condition
// est inerte (cas couvert par le test dédié, qui n'appelle pas ce helper). Les
// lignes existantes héritent du DEFAULT.
func addEventsVerdictColumns(t *testing.T, shared *sql.DB) {
	t.Helper()
	mustExec(t, shared, `ALTER TABLE match_registry ADD COLUMN events_loaded BOOLEAN DEFAULT FALSE`)
	mustExec(t, shared, `ALTER TABLE match_registry ADD COLUMN events_empty BOOLEAN`)
}

// setEventsVerdict fixe le verdict du pipeline events pour un match existant.
// eventsEmpty accepte nil (NULL = aucun verdict, le film est encore attendu).
func setEventsVerdict(t *testing.T, shared *sql.DB, matchID string, eventsLoaded bool, eventsEmpty any) {
	t.Helper()
	mustExec(t, shared,
		`UPDATE match_registry SET events_loaded = ?, events_empty = ? WHERE match_id = ?`,
		eventsLoaded, eventsEmpty, matchID)
}

// insertCitationCandidate crée un match candidat aux citations : ligne registre,
// participant pour fixXUID, ligne player_match_enrichment. Aucune médaille → 0
// delta, le cas qui déclenche la décision d'état terminal.
func insertCitationCandidate(t *testing.T, f *pipelineFixture, matchID string, eventsLoaded bool, eventsEmpty any) {
	t.Helper()
	mustExec(t, f.shared, `
		INSERT INTO match_registry
			(match_id, start_time, playlist_name, game_variant_name,
			 is_ranked, duration_seconds, events_loaded, events_empty)
		VALUES (?, TIMESTAMP '2026-06-13 20:00:00', 'Ranked Arena', 'Slayer', FALSE, 600, ?, ?)`,
		matchID, eventsLoaded, eventsEmpty)
	mustExec(t, f.shared, `
		INSERT INTO match_participants
			(match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists,
			 personal_score, time_played_seconds)
		VALUES (?, ?, ?, 0, 2, 5, 4, 1, 1000, 600)`,
		matchID, fixXUID, fixGamertag)
	mustExec(t, f.player, `INSERT INTO player_match_enrichment (match_id) VALUES (?)`, matchID)
}

// runCitationsFor exécute le backfill sur un seul match.
func runCitationsFor(t *testing.T, f *pipelineFixture, matchID string) {
	t.Helper()
	if err := BackfillMatchCitations(
		context.Background(), f.metadata, f.shared, f.player, nil, fixXUID, []string{matchID},
	); err != nil {
		t.Fatalf("BackfillMatchCitations(%s): %v", matchID, err)
	}
}

// runCitationsForceRecompute rejoue la CHAÎNE force=true de RunBackfillCitations
// (citations_backfill.go) : sélection sans LEFT JOIN, recreateCitationsTable, puis
// backfill. Seule la plomberie leases/ouverture de fichiers est omise — la fixture
// travaille sur des DB in-memory.
func runCitationsForceRecompute(t *testing.T, f *pipelineFixture) {
	t.Helper()
	ctx := context.Background()
	ids, err := selectMatchesForCitations(ctx, f.player, true)
	if err != nil {
		t.Fatalf("selectMatchesForCitations(force=true): %v", err)
	}
	if err := recreateCitationsTable(ctx, f.player); err != nil {
		t.Fatalf("recreateCitationsTable: %v", err)
	}
	if err := BackfillMatchCitations(ctx, f.metadata, f.shared, f.player, nil, fixXUID, ids); err != nil {
		t.Fatalf("BackfillMatchCitations(force): %v", err)
	}
}

// countCitationRows compte les lignes physiques écrites pour un match (toutes
// générations confondues) : 0 prouve qu'aucun jeton n'a été posé.
func countCitationRows(t *testing.T, player *sql.DB, matchID string) int {
	t.Helper()
	var n int
	if err := player.QueryRow(
		`SELECT COUNT(*) FROM match_citations WHERE match_id = ?`, matchID).Scan(&n); err != nil {
		t.Fatalf("countCitationRows(%s): %v", matchID, err)
	}
	return n
}

// countSentinelRows compte les jetons "_processed" visibles pour un match.
func countSentinelRows(t *testing.T, player *sql.DB, matchID string) int {
	t.Helper()
	var n int
	if err := player.QueryRow(
		`SELECT COUNT(*) FROM match_citations_latest WHERE match_id = ? AND citation_name_norm = '_processed'`,
		matchID).Scan(&n); err != nil {
		t.Fatalf("countSentinelRows(%s): %v", matchID, err)
	}
	return n
}

// realCitationValue retourne la valeur d'une citation RÉELLE (hors jeton).
func realCitationValue(t *testing.T, player *sql.DB, matchID, nameNorm string) int {
	t.Helper()
	var v sql.NullInt64
	if err := player.QueryRow(
		`SELECT value FROM match_citations_latest WHERE match_id = ? AND citation_name_norm = ?`,
		matchID, nameNorm).Scan(&v); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("realCitationValue(%s/%s): %v", matchID, nameNorm, err)
	}
	return int(v.Int64)
}

// isCandidate indique si le match figure encore dans le pool de sélection.
func isCandidate(t *testing.T, player *sql.DB, matchID string, force bool) bool {
	t.Helper()
	ids, err := selectMatchesForCitations(context.Background(), player, force)
	if err != nil {
		t.Fatalf("selectMatchesForCitations(force=%v): %v", force, err)
	}
	for _, id := range ids {
		if id == matchID {
			return true
		}
	}
	return false
}

// TestCitationsTerminalState_VerdictEventsEmpty — cas du match ANNULÉ : 0 citation,
// events non chargés, et le pipeline events a rendu son verdict (events_empty).
// Le jeton doit être posé et le match sortir du pool force=false : fin de la
// boucle perpétuelle. Retirer la 3e condition de la branche fait rougir ce test.
func TestCitationsTerminalState_VerdictEventsEmpty(t *testing.T) {
	f := buildPipelineFixture(t)
	addEventsVerdictColumns(t, f.shared)

	// fixM3 : aucune médaille dans la fixture → 0 delta. Film récupéré mais sans
	// event exploitable, et les events du film sont purgés : le match annulé de prod.
	mustExec(t, f.shared, `DELETE FROM highlight_events WHERE match_id = ?`, fixM3)
	setEventsVerdict(t, f.shared, fixM3, false, true)

	if !isCandidate(t, f.player, fixM3, false) {
		t.Fatal("préalable : le match doit être candidat avant le backfill")
	}

	runCitationsFor(t, f, fixM3)

	if got := countSentinelRows(t, f.player, fixM3); got != 1 {
		t.Errorf("jeton _processed = %d, want 1 (verdict events_empty rendu)", got)
	}
	if isCandidate(t, f.player, fixM3, false) {
		t.Error("le match reste candidat après le jeton — la boucle perpétuelle continue")
	}
}

// TestCitationsTerminalState_ForceRecomputeRattrapeLesEvents — le jeton ne doit
// pas être une impasse. Après sa pose, on simule l'arrivée tardive des events
// (verdict corrigé + médaille présente) et on rejoue la VRAIE chaîne force=true,
// recreateCitationsTable compris : les citations doivent être recalculées.
//
// Asserter la seule présence du match dans selectMatchesForCitations(force=true)
// ne prouverait rien — cette requête retourne tout, jeton ou pas.
func TestCitationsTerminalState_ForceRecomputeRattrapeLesEvents(t *testing.T) {
	f := buildPipelineFixture(t)
	addEventsVerdictColumns(t, f.shared)
	mustExec(t, f.shared, `DELETE FROM highlight_events WHERE match_id = ?`, fixM3)
	setEventsVerdict(t, f.shared, fixM3, false, true)

	runCitationsFor(t, f, fixM3)
	if got := countSentinelRows(t, f.player, fixM3); got != 1 {
		t.Fatalf("préalable : jeton attendu, got %d", got)
	}
	if v := realCitationValue(t, f.player, fixM3, "bulltrue"); v != 0 {
		t.Fatalf("préalable : aucune citation réelle attendue avant l'arrivée des events, got %d", v)
	}

	// Arrivée tardive des events + de la médaille (fix parser, re-fetch réussi).
	setEventsVerdict(t, f.shared, fixM3, true, false)
	mustExec(t, f.shared, `INSERT INTO medals_earned VALUES (?, ?, ?, ?)`, fixM3, fixXUID, fixMedalBulltrue, 1)

	runCitationsForceRecompute(t, f)

	if v := realCitationValue(t, f.player, fixM3, "bulltrue"); v == 0 {
		t.Error("force=true n'a pas recalculé les citations d'un match jetonné — le jeton est une impasse")
	}
	if got := countSentinelRows(t, f.player, fixM3); got != 0 {
		t.Errorf("le jeton survit au recompute (%d) alors que le match a désormais une citation réelle", got)
	}
}

// TestCitationsTerminalState_SansVerdictResteCandidat — tant que le pipeline
// events n'a pas conclu (events_empty NULL : le film est encore retenté, jusqu'à
// 30 jours), le comportement Phase 4 est intact — rien n'est écrit.
func TestCitationsTerminalState_SansVerdictResteCandidat(t *testing.T) {
	f := buildPipelineFixture(t)
	addEventsVerdictColumns(t, f.shared)
	insertCitationCandidate(t, f, termPendingMatch, false, nil)

	runCitationsFor(t, f, termPendingMatch)

	if got := countCitationRows(t, f.player, termPendingMatch); got != 0 {
		t.Errorf("lignes écrites = %d, want 0 (film encore attendu : les events peuvent arriver)", got)
	}
	if !isCandidate(t, f.player, termPendingMatch, false) {
		t.Error("un match sans verdict doit RESTER candidat (régression Phase 4)")
	}
}

// TestCitationsTerminalState_EventsLoadedUnchanged — non-régression : quand les
// events sont chargés (dont le cas film 404-définitif, où MarkNoFilmDefinitive
// pose events_loaded=TRUE), le jeton est posé comme avant, sans consulter le verdict.
func TestCitationsTerminalState_EventsLoadedUnchanged(t *testing.T) {
	f := buildPipelineFixture(t)
	addEventsVerdictColumns(t, f.shared)
	insertCitationCandidate(t, f, termHealedMatch, true, nil)

	runCitationsFor(t, f, termHealedMatch)

	if got := countSentinelRows(t, f.player, termHealedMatch); got != 1 {
		t.Errorf("jeton _processed = %d, want 1 (events chargés : 0 citation est définitif)", got)
	}
	if isCandidate(t, f.player, termHealedMatch, false) {
		t.Error("un match aux events chargés doit sortir du pool (comportement existant)")
	}
}

// TestCitationsTerminalState_SchemaSansEventsEmpty — sur une DB dont le registre
// n'a pas la colonne events_empty (titre non migré), la nouvelle branche est
// INERTE : verdict illisible → le match reste candidat, exactement comme avant le
// lot. On n'appelle donc pas addEventsVerdictColumns : seul events_loaded est ajouté.
func TestCitationsTerminalState_SchemaSansEventsEmpty(t *testing.T) {
	f := buildPipelineFixture(t)
	mustExec(t, f.shared, `ALTER TABLE match_registry ADD COLUMN events_loaded BOOLEAN DEFAULT FALSE`)
	mustExec(t, f.shared, `DELETE FROM highlight_events WHERE match_id = ?`, fixM3)

	runCitationsFor(t, f, fixM3)

	if got := countCitationRows(t, f.player, fixM3); got != 0 {
		t.Errorf("lignes écrites = %d, want 0 (verdict illisible : échec sûr = rester candidat)", got)
	}
	if !isCandidate(t, f.player, fixM3, false) {
		t.Error("schéma sans events_empty : le match doit rester candidat (comportement d'avant le lot)")
	}
}
