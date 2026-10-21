package duckdb

// resumes_fixture_test.go — L'AIDE DE TEST UNIQUE DES RÉSUMÉS PAR MATCH DU JOURNAL (ADR 0039)
// pour les lecteurs de ce paquet.
//
// Les lecteurs d'agrégats du journal lisent `match_kill_heads_latest`, `match_kill_pairs_latest`
// et `match_assist_pairs_latest`. Une fixture qui pose des morts dans `match_kill_events` doit
// donc poser aussi leurs résumés. `persist` les écrit avec la passe, mais ce paquet ne peut pas
// l'importer (persist importe platform/duckdb) : l'aide relit la passe courante de chaque match et
// la résume par la fonction pure `analysis/killsummary`, la MÊME que l'écrivain réel — aucune
// règle de résumé n'est recopiée ici. La parité avec l'écrivain réel est tenue par
// resumes_parite_test.go (paquet externe, qui écrit par `persist`).

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/killsummary"
	"levelup/go-api/internal/migration"
)

// ouvrirSharedMigree ouvre un shared `:memory:` avec les VRAIES migrations (journal, résumés et
// leurs vues `_latest`), borné à une connexion comme en production.
func ouvrirSharedMigree(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	return db
}

// generationResumes numérote les générations de résumés écrites par l'aide (identifiant unique).
var generationResumes atomic.Int64

func nullSiVide(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ecrireResumes écrit, pour chaque match donné (tous les matchs du journal si aucun), une
// génération de résumés de sa passe COURANTE. Appelable plusieurs fois (une fixture qui pose ses
// morts une à une) : chaque appel remplace les résumés du match.
func ecrireResumes(t *testing.T, db *sql.DB, matchIDs ...string) {
	t.Helper()
	ecrireResumesOpts(t, db, false, matchIDs...)
}

func ecrireResumesOpts(t *testing.T, db *sql.DB, garder bool, matchIDs ...string) {
	t.Helper()
	if len(matchIDs) == 0 {
		rows, err := db.Query(`SELECT DISTINCT match_id FROM match_kill_events_latest ORDER BY match_id`)
		if err != nil {
			t.Fatalf("matchs du journal : %v", err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatalf("scan : %v", err)
			}
			matchIDs = append(matchIDs, id)
		}
		_ = rows.Close()
	}
	for _, id := range matchIDs {
		ecrireResumesDuMatch(t, db, id, garder, fmt.Sprintf("%s-gen%d", id, generationResumes.Add(1)))
	}
}

func ecrireResumesDuMatch(t *testing.T, db *sql.DB, matchID string, garder bool, gen string) {
	t.Helper()
	rows, err := db.Query(`
		SELECT decode_pass, written_at, publishable, victim_xuid, feed_killer_xuid, feed_killer_gamertag,
		       assist_xuid, assist_gamertag, assist_known, source_tag, killer_damage_pct, assist_damage_pct
		FROM match_kill_events_latest WHERE match_id = ?`, matchID)
	if err != nil {
		t.Fatalf("relecture de la passe %s : %v", matchID, err)
	}
	var (
		pass      string
		at        time.Time
		publiable bool
		lues      []killsummary.Row
	)
	for rows.Next() {
		var (
			r                               killsummary.Row
			victimX, killerX, killerGT      sql.NullString
			assistX, assistGT               sql.NullString
			sourceTag, killerPct, assistPct sql.NullInt64
		)
		var rowAt time.Time
		var pub bool
		if err := rows.Scan(&pass, &rowAt, &pub, &victimX, &killerX, &killerGT, &assistX, &assistGT,
			&r.AssistKnown, &sourceTag, &killerPct, &assistPct); err != nil {
			t.Fatalf("scan de la passe %s : %v", matchID, err)
		}
		if len(lues) > 0 && pub != publiable {
			t.Fatalf("match %s : publiabilité mixte dans la passe courante — impossible en production "+
				"(`KillSourceBatch.Publishable` est constant sur la passe) ; scinder la fixture", matchID)
		}
		publiable = pub
		if rowAt.After(at) {
			at = rowAt // la passe porte l'horodatage de sa dernière ligne ; la tête suit la passe
		}
		r.VictimXUID, r.KillerXUID, r.KillerGamertag = victimX.String, killerX.String, killerGT.String
		r.AssistXUID, r.AssistGamertag = assistX.String, assistGT.String
		r.SourceTag = uint32(sourceTag.Int64)
		if killerPct.Valid {
			v := uint8(killerPct.Int64)
			r.KillerDamagePct = &v
		}
		if assistPct.Valid {
			v := uint8(assistPct.Int64)
			r.AssistDamagePct = &v
		}
		lues = append(lues, r)
	}
	_ = rows.Close()
	if len(lues) == 0 {
		return
	}
	s := killsummary.Summarize(publiable, lues)
	ctx := context.Background()
	// Une fixture qui pose ses morts une à une réécrit les résumés du match : l'aide ne laisse
	// qu'UNE génération (comme une passe écrite en ligne), sans quoi les fenêtres des vues
	// compteraient les générations intermédiaires de la fixture.
	for _, table := range []string{"match_assist_pairs", "match_kill_pairs", "match_kill_heads"} {
		if garder {
			break
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM `+table+` WHERE match_id = ?`, matchID); err != nil {
			t.Fatalf("remplacement des résumés %s %s : %v", table, matchID, err)
		}
	}
	h := s.Head
	if _, err := db.ExecContext(ctx, `INSERT INTO match_kill_heads
		(match_id, summary_pass, summary_rev, src_decode_pass, written_at,
		 publishable, n_rows, n_assist_known, n_killer_xuid, n_src_killer)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, matchID, gen, s.Rev, pass, at,
		h.Publishable, h.Rows, h.AssistKnown, h.KillerXUID, h.SourceAndKiller); err != nil {
		t.Fatalf("tête %s : %v", matchID, err)
	}
	for _, p := range s.Pairs {
		var tag any
		if p.SourceTag != 0 {
			tag = int64(p.SourceTag)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO match_kill_pairs
			(match_id, summary_pass, summary_rev, src_decode_pass, written_at,
			 feed_killer_xuid, victim_xuid, source_tag, kills)
			VALUES (?, ?, ?, ?, ?, ?, ?, CAST(? AS UINTEGER), ?)`, matchID, gen, s.Rev, pass, at,
			nullSiVide(p.KillerXUID), nullSiVide(p.VictimXUID), tag, p.Kills); err != nil {
			t.Fatalf("paire %s : %v", matchID, err)
		}
	}
	for _, a := range s.Assists {
		if _, err := db.ExecContext(ctx, `INSERT INTO match_assist_pairs
			(match_id, summary_pass, summary_rev, src_decode_pass, written_at,
			 assist_xuid, assist_gamertag, feed_killer_xuid, feed_killer_gamertag, victim_xuid,
			 n, n_stolen, sum_assist_pct, n_assist_pct, n_low, n_mid, n_high)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, matchID, gen, s.Rev, pass, at,
			nullSiVide(a.AssistXUID), nullSiVide(a.AssistGamertag), nullSiVide(a.KillerXUID),
			nullSiVide(a.KillerGamertag), nullSiVide(a.VictimXUID),
			a.N, a.Stolen, a.PctSum, a.PctCount, a.Low, a.Mid, a.High); err != nil {
			t.Fatalf("appui %s : %v", matchID, err)
		}
	}
}

// tetesDuJournal rend le nombre de lignes de la table des têtes (une par match et génération) :
// la borne de la fenêtre de la tête quand une vue fille (S1, S2) est lue sous une liste de matchs
// (ADR 0039 décision 7 : la liste ne traverse pas la jointure, la fenêtre de la tête voit toutes
// les têtes, jamais les lignes des filles).
func tetesDuJournal(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_heads`).Scan(&n); err != nil {
		t.Fatalf("têtes du journal : %v", err)
	}
	return n
}
