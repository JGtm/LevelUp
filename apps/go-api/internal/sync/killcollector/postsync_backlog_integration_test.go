//go:build integration

package killcollector

// postsync_backlog_integration_test.go — LA REQUETE DE BACKLOG, EXECUTEE POUR DE VRAI.
//
// POURQUOI CE FICHIER EXISTE. La premiere version de ce lot ne verifiait la requete que par
// `strings.Contains` sur son texte. Une virgule manquante, un `?` de trop, un renommage de la
// vue : toutes ces fautes gardent les sous-chaines presentes, donc le test restait vert
// pendant qu a l execution `QueryContext` echouait -> WARN -> backlog vide -> etape no-op
// permanente. C est-a-dire EXACTEMENT le mode de panne silencieuse que ce lot corrige,
// reproduit dans son propre garde-rail.
//
// Ici la requete tourne sur une vraie base migree. Elle ne peut plus mentir sur sa syntaxe,
// ni sur l ensemble qu elle selectionne, ni sur son ordre.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync/matchflags"
)

func baseBacklog(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "backlog.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

func inscrireMatch(t *testing.T, db *sql.DB, id string, quand time.Time, bits int64) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO match_registry (match_id, start_time, start_time_utc, backfill_completed)
		 VALUES (?, ?, ?, ?)`, id, quand, quand, bits)
	if err != nil {
		t.Fatalf("insert registre %s: %v", id, err)
	}
}

// inscrirePasseFilm ecrit UNE passe. `decode_pass` est l unite de production (une passe, un
// match) et la vue `_latest` ne retient que la plus recente : le passer explicitement est ce
// qui permet d ecrire une passe PERIMEE puis une passe courante sur le meme match.
func inscrirePasseFilm(t *testing.T, db *sql.DB, id, rev, voie, passe string, quand time.Time) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO match_kill_events
		   (match_id, decode_pass, decoder_rev, time_ms, victim_gamertag, feed_present,
		    assist_known, publishable, read_path, read_origin, written_at)
		 VALUES (?, ?, ?, 1000, 'V', TRUE, TRUE, TRUE, ?, 'concordant-test', ?)`,
		id, passe, rev, voie, quand)
	if err != nil {
		t.Fatalf("insert kill event %s: %v", id, err)
	}
}

// TestBacklogAJour_SelectionOrdreEtJauge : les quatre proprietes de la selection, sur une base
// reelle.
//
//  1. le marqueur terminal EXCLUT       un match declare « film absent » par l etape 1.55 ne
//     revient jamais — sans cela il occupe la liste a vie
//     (mesure : 581 des 999 candidats du 2026-08-29).
//  2. une passe de FILM a jour EXCLUT   c est le travail deja fait.
//  3. une passe CREDIT n exclut PAS     toute la base en est couverte ; l exclure ferait
//     croire l etape a jour partout.
//  4. l ordre est du PLUS RECENT au plus vieux, et l horizon borne la LISTE sans fausser la
//     JAUGE — un compteur tronque mentirait sur le retard.
func TestBacklogAJour_SelectionOrdreEtJauge(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	inscrireMatch(t, db, "vieux-nofilm", t0, int64(matchflags.MBitFilmAbsent)) // (1)
	inscrireMatch(t, db, "deja-decode", t0.AddDate(0, 1, 0), 0)                // (2)
	inscrirePasseFilm(t, db, "deja-decode", KillSourceDecoderRev, killscope.ReadPathFilmWalk, "p1", t0)
	inscrireMatch(t, db, "via-credit", t0.AddDate(0, 2, 0), 0) // (3)
	inscrirePasseFilm(t, db, "via-credit", KillSourceDecoderRev, killscope.ReadPathCreditBackfill, "p1", t0)
	inscrireMatch(t, db, "vierge-ancien", t0.AddDate(0, 3, 0), 0)
	inscrireMatch(t, db, "vierge-recent", t0.AddDate(0, 4, 0), 0)

	ids, total := backlogAJour(context.Background(), db, 10)

	if total != 3 {
		t.Errorf("taille du backlog = %d, attendu 3 (via-credit, vierge-ancien, vierge-recent)", total)
	}
	attendu := []string{"vierge-recent", "vierge-ancien", "via-credit"}
	if len(ids) != len(attendu) {
		t.Fatalf("liste = %v, attendu %v", ids, attendu)
	}
	for i, id := range attendu {
		if ids[i] != id {
			t.Errorf("liste[%d] = %q, attendu %q — l ordre doit aller du plus RECENT au plus vieux "+
				"(les films recuperables d abord)", i, ids[i], id)
		}
	}

	// L horizon borne la liste, JAMAIS la jauge.
	bornee, totalBorne := backlogAJour(context.Background(), db, 1)
	if len(bornee) != 1 || bornee[0] != "vierge-recent" {
		t.Errorf("liste bornee = %v, attendu [vierge-recent]", bornee)
	}
	if totalBorne != 3 {
		t.Errorf("jauge sous horizon = %d, attendu 3 — une jauge tronquee decrirait l horizon, "+
			"pas le retard", totalBorne)
	}
}

// TestBacklogAJour_PasseAncienneNeComptePas : la lecture passe par la vue `_latest` (ADR 0026).
// Une passe de film a une revision PERIMEE ne doit pas faire sauter le match — sinon un
// changement de decodeur ne rejouerait jamais.
func TestBacklogAJour_PasseAncienneNeComptePas(t *testing.T) {
	db := baseBacklog(t)
	inscrireMatch(t, db, "revision-perimee", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), 0)
	inscrirePasseFilm(t, db, "revision-perimee", "killsource-2020-01-01", killscope.ReadPathFilmWalk, "p1",
		time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))

	ids, total := backlogAJour(context.Background(), db, 10)
	if total != 1 || len(ids) != 1 || ids[0] != "revision-perimee" {
		t.Errorf("liste = %v (total %d) ; un match decode a une revision perimee doit rester "+
			"candidat", ids, total)
	}
}

// TestBacklogAJour_PasseCouranteSupplanteLaPerimee : deux passes sur le meme match. La vue
// `_latest` ne retient que la plus recente ; si la plus recente est a jour, le match sort du
// backlog — meme si une passe perimee traine derriere lui dans la table.
func TestBacklogAJour_PasseCouranteSupplanteLaPerimee(t *testing.T) {
	db := baseBacklog(t)
	quand := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	inscrireMatch(t, db, "redecode", quand, 0)
	inscrirePasseFilm(t, db, "redecode", "killsource-2020-01-01", killscope.ReadPathFilmWalk, "p1", quand)
	inscrirePasseFilm(t, db, "redecode", KillSourceDecoderRev, killscope.ReadPathFilmWalk, "p2", quand.Add(time.Hour))

	if ids, total := backlogAJour(context.Background(), db, 10); total != 0 || len(ids) != 0 {
		t.Errorf("liste = %v (total %d) ; la passe COURANTE doit faire sortir le match", ids, total)
	}
}

// TestBacklogAJour_BaseVide : aucune ligne, aucune erreur, aucune panique.
func TestBacklogAJour_BaseVide(t *testing.T) {
	ids, total := backlogAJour(context.Background(), baseBacklog(t), 10)
	if len(ids) != 0 || total != 0 {
		t.Errorf("liste = %v, total = %d ; attendu vide", ids, total)
	}
}
