//go:build integration

// Package sync — citations_terminal_state_pipeline_test.go : état terminal des
// citations, de bout en bout à travers BackfillMatchCitations sur la fixture de
// pipeline (jeton posé, sortie du pool, réversibilité par force=true,
// non-régression du chemin « events présents »).
//
// La décision isolée (matchAge, isCitationsTerminalNoEvents — citations.go) est
// couverte par citations_terminal_state_test.go.
package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	termRecentMatch = "match-terminal-recent"
	termHealedMatch = "match-terminal-events-ok"
)

// addEventsLoadedColumn ajoute events_loaded à la fixture partagée. buildSharedDDL
// ne porte PAS cette colonne : sans elle, isEventsLoaded répond true par
// best-effort et la branche testée ici n'est jamais atteinte. Les lignes déjà
// insérées héritent du DEFAULT FALSE (events non chargés).
func addEventsLoadedColumn(t *testing.T, shared *sql.DB) {
	t.Helper()
	mustExec(t, shared, `ALTER TABLE match_registry ADD COLUMN events_loaded BOOLEAN DEFAULT FALSE`)
}

// setRegistryAge repositionne le début d'un match du registre à `ago` dans le
// passé (les deux colonnes du fragment canonique) et fixe son events_loaded.
// L'âge est relatif à maintenant : le test ne dépend pas de l'horloge de la
// machine ni des dates figées de la fixture.
func setRegistryAge(t *testing.T, shared *sql.DB, matchID string, ago time.Duration, eventsLoaded bool) {
	t.Helper()
	start := time.Now().Add(-ago)
	mustExec(t, shared,
		`UPDATE match_registry SET start_time = ?, start_time_utc = ?, events_loaded = ? WHERE match_id = ?`,
		start.UTC(), start, eventsLoaded, matchID)
}

// insertCitationCandidate crée un match candidat aux citations : ligne registre
// (âge choisi), participant pour fixXUID, ligne player_match_enrichment. Aucune
// médaille → 0 delta, le cas qui déclenche la décision d'état terminal.
func insertCitationCandidate(t *testing.T, f *pipelineFixture, matchID string, ago time.Duration, eventsLoaded bool) {
	t.Helper()
	start := time.Now().Add(-ago)
	mustExec(t, f.shared, `
		INSERT INTO match_registry
			(match_id, start_time, start_time_utc, playlist_name, game_variant_name,
			 is_ranked, duration_seconds, events_loaded)
		VALUES (?, ?, ?, 'Ranked Arena', 'Slayer', FALSE, 600, ?)`,
		matchID, start.UTC(), start, eventsLoaded)
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

// TestCitationsTerminalState_OldMatchWithoutEvents — cas du match ANNULÉ : 0
// citation, events jamais chargés, âge au-delà du seuil. Le jeton doit être posé
// et le match sortir du pool force=false (fin de la boucle perpétuelle), tout en
// restant rattrapable par un recompute force=true.
func TestCitationsTerminalState_OldMatchWithoutEvents(t *testing.T) {
	f := buildPipelineFixture(t)
	addEventsLoadedColumn(t, f.shared)

	// fixM3 : aucune médaille dans la fixture → 0 delta. Les events du film sont
	// purgés et le registre les déclare non chargés : le match annulé de prod.
	mustExec(t, f.shared, `DELETE FROM highlight_events WHERE match_id = ?`, fixM3)
	setRegistryAge(t, f.shared, fixM3, citationsTerminalNoEventsAge+30*24*time.Hour, false)

	if !isCandidate(t, f.player, fixM3, false) {
		t.Fatal("préalable : le match doit être candidat avant le backfill")
	}

	runCitationsFor(t, f, fixM3)

	if got := countSentinelRows(t, f.player, fixM3); got != 1 {
		t.Errorf("jeton _processed = %d, want 1 (état terminal : events jamais arrivés)", got)
	}
	if isCandidate(t, f.player, fixM3, false) {
		t.Error("le match reste candidat après le jeton — la boucle perpétuelle continue")
	}
	// Réversibilité documentée dans citations.go : force=true ne consulte pas
	// match_citations, un match jetonné est donc retraité par un recompute.
	if !isCandidate(t, f.player, fixM3, true) {
		t.Error("force=true doit toujours re-sélectionner un match jetonné (recompute impossible sinon)")
	}
}

// TestCitationsTerminalState_RecentMatchStaysCandidate — sous le seuil, le
// comportement Phase 4 est intact : rien n'est écrit, le match attend son film.
func TestCitationsTerminalState_RecentMatchStaysCandidate(t *testing.T) {
	f := buildPipelineFixture(t)
	addEventsLoadedColumn(t, f.shared)
	insertCitationCandidate(t, f, termRecentMatch, time.Hour, false)

	runCitationsFor(t, f, termRecentMatch)

	if got := countCitationRows(t, f.player, termRecentMatch); got != 0 {
		t.Errorf("lignes écrites = %d, want 0 (match récent : les events peuvent encore arriver)", got)
	}
	if !isCandidate(t, f.player, termRecentMatch, false) {
		t.Error("un match récent sans events doit RESTER candidat (régression Phase 4)")
	}
}

// TestCitationsTerminalState_EventsLoadedUnchanged — non-régression : quand les
// events sont chargés, le jeton est posé comme avant, sans considération d'âge
// (ici un match d'une heure, très en-deçà du seuil).
func TestCitationsTerminalState_EventsLoadedUnchanged(t *testing.T) {
	f := buildPipelineFixture(t)
	addEventsLoadedColumn(t, f.shared)
	insertCitationCandidate(t, f, termHealedMatch, time.Hour, true)

	runCitationsFor(t, f, termHealedMatch)

	if got := countSentinelRows(t, f.player, termHealedMatch); got != 1 {
		t.Errorf("jeton _processed = %d, want 1 (events chargés : 0 citation est définitif)", got)
	}
	if isCandidate(t, f.player, termHealedMatch, false) {
		t.Error("un match aux events chargés doit sortir du pool (comportement existant)")
	}
}
