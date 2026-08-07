package sync

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis"
)

// ensureObjectiveEventsTable crée match_objective_events si absente. Miroir du DDL
// de la migration shared_objective_events_v1 (TargetShared) : EnsureSharedSchema ne
// crée pas cette table (net-new film v3), on l'ajoute localement comme
// ensureDominanceFlagColumn le fait pour la colonne dominance_flag.
func ensureObjectiveEventsTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS match_objective_events (
			match_id       VARCHAR NOT NULL,
			seq            INTEGER NOT NULL,
			time_ms        INTEGER,
			objective_type VARCHAR,
			event_type     VARCHAR,
			team_id        INTEGER,
			objective_id   INTEGER,
			value          INTEGER,
			source         VARCHAR,
			confidence     VARCHAR,
			details        VARCHAR,
			written_at     TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (match_id, seq)
		)`); err != nil {
		t.Fatalf("create match_objective_events: %v", err)
	}
}

// seedCapture insère un event de capture CTF (objective_type='flag', value=1).
func seedCapture(t *testing.T, db *sql.DB, matchID string, seq int, timeMS int64, teamID int) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO match_objective_events
			(match_id, seq, time_ms, objective_type, event_type, team_id, value, source, confidence)
		VALUES (?, ?, ?, 'flag', 'capture', ?, 1, 'burst', 'exact')`,
		matchID, seq, timeMS, teamID); err != nil {
		t.Fatalf("insert capture seq=%d: %v", seq, err)
	}
}

// TestBackfillDominanceFlags_CTFRemontadaFromCaptures — un match CTF où l'équipe
// adverse (team1) mène tôt (0-3) puis le joueur (team0, Win) recolle et gagne 4-3.
// La courbe vient des captures objectif, pas des kills ni de Steaktacular.
// Attendu : REMONTADA (3).
func TestBackfillDominanceFlags_CTFRemontadaFromCaptures(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	ensureObjectiveEventsTable(t, sharedDB)

	const matchID, myXUID = "m_ctf_rem", "me"
	// team0 = moi (Win), team1 = adversaire.
	seedComebackMatch(t, sharedDB, matchID, "CTF:Arena", myXUID, 0, 2 /* Win */, 1)

	// team1 prend 3 drapeaux d'avance, puis team0 recolle et passe devant 4-3.
	seedCapture(t, sharedDB, matchID, 0, 10_000, 1) // 0-1
	seedCapture(t, sharedDB, matchID, 1, 20_000, 1) // 0-2
	seedCapture(t, sharedDB, matchID, 2, 30_000, 1) // 0-3
	seedCapture(t, sharedDB, matchID, 3, 40_000, 0) // 1-3
	seedCapture(t, sharedDB, matchID, 4, 50_000, 0) // 2-3
	seedCapture(t, sharedDB, matchID, 5, 60_000, 0) // 3-3
	seedCapture(t, sharedDB, matchID, 6, 70_000, 0) // 4-3 (final)

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagRemontada {
		t.Errorf("CTF remontada -> flag %d (Remontada) attendu, got %d",
			analysis.DominanceFlagRemontada, got)
	}
}

// TestBackfillDominanceFlags_CTFNoCapturesFallsBackToHistorical — un match CTF
// sans capture exploitable ne produit PAS un 0 définitif : la courbe objectif
// étant vide, le chemin historique reprend la main et voit la Steaktacular.
//
// Ce test remplace TestBackfillDominanceFlags_CTFNoCapturesNoFlag, qui exigeait
// flag=0 dans cette situation. Sa prémisse — « la source objectif est peuplée,
// donc une courbe vide signifie vraiment aucune capture » — est fausse en
// production : rien dans la sync n'alimente match_objective_events (audit
// 2026-08-06, P0). L'intention d'origine (ne pas mélanger deux sources sur la
// même courbe) est conservée par TestBackfillDominanceFlags_CTFRemontadaFromCaptures :
// quand la courbe existe, la Steaktacular reste ignorée.
func TestBackfillDominanceFlags_CTFNoCapturesFallsBackToHistorical(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	ensureObjectiveEventsTable(t, sharedDB)

	const matchID, myXUID = "m_ctf_empty", "me"
	seedComebackMatch(t, sharedDB, matchID, "Ranked:CTF", myXUID, 0, 2, 1)
	seedSteaktacularMedal(t, sharedDB, matchID, myXUID)

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagDomination {
		t.Errorf("CTF sans capture -> repli historique (Domination %d) attendu, got %d",
			analysis.DominanceFlagDomination, got)
	}
}

// TestBackfillDominanceFlags_ZoneModeFallsBackToHistorical — un mode zone
// (Strongholds) ne tente jamais la courbe objectif (score au tick, pas à
// l'event : on ne fabrique pas de fausse courbe depuis des counts) et part
// directement sur le chemin historique.
//
// Remplace TestBackfillDominanceFlags_ZoneModeReturnsZero, même correction de
// prémisse que le test ci-dessus : le court-circuit à 0 gelait Strongholds,
// KOTH et Oddball en plus du CTF.
func TestBackfillDominanceFlags_ZoneModeFallsBackToHistorical(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	ensureObjectiveEventsTable(t, sharedDB)

	const matchID, myXUID = "m_zone", "me"
	seedComebackMatch(t, sharedDB, matchID, "Strongholds:Arena", myXUID, 0, 2, 1)
	seedSteaktacularMedal(t, sharedDB, matchID, myXUID)

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagDomination {
		t.Errorf("mode zone -> repli historique (Domination %d) attendu, got %d",
			analysis.DominanceFlagDomination, got)
	}
}

// TestBackfillDominanceFlags_ObjectiveWithoutAnySignalStaysZero — le repli ne
// fabrique pas de dominance : sans capture, sans Steaktacular et sans kill event,
// un match à objectif reste à 0. Garde-fou contre un faux positif introduit par
// le repli lui-même.
func TestBackfillDominanceFlags_ObjectiveWithoutAnySignalStaysZero(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	ensureObjectiveEventsTable(t, sharedDB)

	const matchID, myXUID = "m_ctf_silent", "me"
	seedComebackMatch(t, sharedDB, matchID, "Ranked:CTF", myXUID, 0, 2, 1)

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagNone {
		t.Errorf("CTF sans aucun signal -> flag 0 attendu, got %d", got)
	}
}

// TestBackfillDominanceFlags_ObjectiveTableMissingFallsBack — la table
// match_objective_events absente (migration shared_objective_events_v1 non
// passée) fait échouer la lecture : l'erreur est tracée et le repli couvre, au
// lieu de persister un 0 muet et définitif (audit 2026-08-06, P1).
func TestBackfillDominanceFlags_ObjectiveTableMissingFallsBack(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	// Volontairement PAS de ensureObjectiveEventsTable.

	const matchID, myXUID = "m_ctf_no_table", "me"
	seedComebackMatch(t, sharedDB, matchID, "Ranked:CTF", myXUID, 0, 2, 1)
	seedSteaktacularMedal(t, sharedDB, matchID, myXUID)

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagDomination {
		t.Errorf("table objectif absente -> repli historique (Domination %d) attendu, got %d",
			analysis.DominanceFlagDomination, got)
	}
}
