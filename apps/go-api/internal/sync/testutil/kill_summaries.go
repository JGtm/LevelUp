//go:build integration

package testutil

// kill_summaries.go — LES CONSTATS DES RÉSUMÉS PAR MATCH DU JOURNAL (ADR 0039), partagés par les
// tests d'intégration de chaque écrivain (sync primaire, complétion, collecteur, BatchBuilder,
// backfill) : un seul énoncé de « la tête résume la passe courante », pas une copie par paquet.

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis/killsummary"
)

// etatResumeSQL : pour un match, la passe courante du journal et ce que les vues de résumé en
// disent. Chaque vue est bornée par sa propre constante (`match_id = ?`).
const etatResumeSQL = `
	SELECT
		(SELECT any_value(decode_pass) FROM match_kill_events_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ? AND publishable),
		(SELECT COUNT(*) FROM match_kill_heads_latest WHERE match_id = ?),
		(SELECT any_value(src_decode_pass) FROM match_kill_heads_latest WHERE match_id = ?),
		(SELECT any_value(summary_rev) FROM match_kill_heads_latest WHERE match_id = ?),
		(SELECT any_value(n_rows) FROM match_kill_heads_latest WHERE match_id = ?),
		(SELECT COALESCE(SUM(kills), 0) FROM match_kill_pairs_latest WHERE match_id = ?)`

// EtatResume : l'état lu par [LireEtatResume].
type EtatResume struct {
	PasseJournal  string
	LignesJournal int
	Publiables    int
	Tetes         int
	PasseTete     string
	RevTete       string
	LignesTete    int
	FragsS1       int
}

// LireEtatResume lit l'état du journal et de ses résumés pour un match.
func LireEtatResume(t *testing.T, db *sql.DB, matchID string) EtatResume {
	t.Helper()
	var (
		e                     EtatResume
		passe, passeTete, rev sql.NullString
		lignesTete            sql.NullInt64
	)
	args := make([]any, 8)
	for i := range args {
		args[i] = matchID
	}
	if err := db.QueryRowContext(context.Background(), etatResumeSQL, args...).Scan(&passe, &e.LignesJournal, &e.Publiables,
		&e.Tetes, &passeTete, &rev, &lignesTete, &e.FragsS1); err != nil {
		t.Fatalf("état des résumés %s : %v", matchID, err)
	}
	e.PasseJournal, e.PasseTete, e.RevTete = passe.String, passeTete.String, rev.String
	e.LignesTete = int(lignesTete.Int64)
	return e
}

// ExigerResumeDeLaPasseCourante : le match a une passe au journal, UNE tête la résume à la
// révision courante, et S1 compte exactement les morts de cette passe.
func ExigerResumeDeLaPasseCourante(t *testing.T, db *sql.DB, matchID string) EtatResume {
	t.Helper()
	e := LireEtatResume(t, db, matchID)
	switch {
	case e.LignesJournal == 0:
		t.Fatalf("%s : aucune ligne au journal, le test ne mesure rien", matchID)
	case e.Tetes != 1:
		t.Fatalf("%s : %d tête(s) servie(s), attendu 1", matchID, e.Tetes)
	case e.PasseTete != e.PasseJournal:
		t.Fatalf("%s : la tête résume la passe %q, la passe courante du journal est %q",
			matchID, e.PasseTete, e.PasseJournal)
	case e.RevTete != killsummary.Rev:
		t.Fatalf("%s : tête à la révision %q, attendu %q", matchID, e.RevTete, killsummary.Rev)
	case e.LignesTete != e.LignesJournal || e.FragsS1 != e.LignesJournal:
		t.Fatalf("%s : tête n_rows=%d, somme des frags S1=%d, journal=%d — un résumé compte "+
			"exactement les morts de sa passe", matchID, e.LignesTete, e.FragsS1, e.LignesJournal)
	}
	return e
}

// ExigerAucunePasse : ni ligne au journal ni paire de résumé pour ce match — une transaction
// annulée n'a rien laissé. Lit les tables brutes (aucune génération, même périmée) ; la table des
// têtes n'est pas lue, [CasserLaTableDesTetes] l'a retirée.
func ExigerAucunePasse(t *testing.T, db *sql.DB, matchID string) {
	t.Helper()
	var journal, paires int
	if err := db.QueryRowContext(context.Background(), `SELECT
		(SELECT COUNT(*) FROM match_kill_events WHERE match_id = ?),
		(SELECT COUNT(*) FROM match_kill_pairs WHERE match_id = ?)`, matchID, matchID).
		Scan(&journal, &paires); err != nil {
		t.Fatalf("relecture %s : %v", matchID, err)
	}
	if journal != 0 || paires != 0 {
		t.Fatalf("%s : %d ligne(s) au journal et %d paire(s) après un échec des résumés — la passe "+
			"et ses résumés tombent ensemble", matchID, journal, paires)
	}
}

// CasserLaTableDesTetes retire `match_kill_heads` : l'insertion de la tête échoue alors en base,
// APRÈS celle du journal dans la même transaction. Sert l'injection d'erreur de chaque écrivain.
func CasserLaTableDesTetes(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `DROP TABLE match_kill_heads`); err != nil {
		t.Fatalf("retrait de match_kill_heads : %v", err)
	}
}

// ExigerCoherenceDesResumes : pour CHAQUE match du journal, la tête résume la passe courante et S1
// en compte exactement les morts. Rend le nombre de matchs vérifiés.
func ExigerCoherenceDesResumes(t *testing.T, db *sql.DB) int {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		`SELECT DISTINCT match_id FROM match_kill_events_latest ORDER BY match_id`)
	if err != nil {
		t.Fatalf("matchs du journal : %v", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan : %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("matchs du journal : %v", err)
	}
	_ = rows.Close()
	for _, id := range ids {
		ExigerResumeDeLaPasseCourante(t, db, id)
	}
	return len(ids)
}
