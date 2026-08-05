//go:build cgo

package migration

// Tests du DDL `match_kill_events` — ils verrouillent les QUATRE decisions du schema, pas la
// syntaxe SQL :
//
//  1. la vue retient LA DERNIERE PASSE ENTIERE par match, jamais la derniere ligne par mort ;
//  2. `damage_pct_residual` vaut NULL des qu un terme manque (« NULL n est jamais zero ») et
//     n est ni borne ni positif par construction ;
//  3. la vue SUIT une colonne ajoutee par ALTER — c est ce qui rend la migration future
//     « table fille d assistants » bon marche, et le jour ou ce ne serait plus vrai, ce test
//     sonne avant la prod ;
//  4. `killer_victim_pairs` n est ni supprimee ni modifiee par cette migration.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
)

// killEventsDB : une base avec la table, l index et la vue.
func killEventsDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTmpDB(t)
	if err := applyMatchKillEvents(db); err != nil {
		t.Fatalf("applyMatchKillEvents: %v", err)
	}
	// Idempotence : rejouer la migration ne doit rien casser (le boot la rejoue a chaque
	// demarrage).
	if err := applyMatchKillEvents(db); err != nil {
		t.Fatalf("applyMatchKillEvents (2e passage): %v", err)
	}
	return db
}

// insertKillRow ecrit une ligne minimale valide. Les parametres qui portent le sens du test
// sont explicites ; le reste est du remplissage constant.
func insertKillRow(t *testing.T, db *sql.DB, matchID, pass string, timeMS int, victim string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, publishable, time_ms,
			victim_gamertag, feed_present, assist_known, read_path, read_origin
		) VALUES (?, ?, 'test-rev', TRUE, ?, ?, TRUE, TRUE, ?, 'credit-concordant')`,
		matchID, pass, timeMS, victim, killscope.ReadPathFilmWalk)
	if err != nil {
		t.Fatalf("insert (%s/%s): %v", matchID, pass, err)
	}
}

// TestVueRetientLaDernierePasseEntiere — LE test qui justifie `decode_pass`.
//
// Le scenario est celui qui a motive la conception : une passe A publie 3 morts, une passe B
// plus recente n en publie que 2. Une vue « derniere ligne par mort » laisserait survivre la
// 3e ligne de A — une mort que le decodeur courant ne voit plus. La vue doit rendre 2 lignes,
// toutes de B.
func TestVueRetientLaDernierePasseEntiere(t *testing.T) {
	db := killEventsDB(t)

	insertKillRow(t, db, "m1", "passeA", 1000, "Victime1")
	insertKillRow(t, db, "m1", "passeA", 2000, "Victime2")
	insertKillRow(t, db, "m1", "passeA", 3000, "Victime3")
	// Passe B : written_at par defaut = now(), donc posterieure ; l id l est aussi.
	insertKillRow(t, db, "m1", "passeB", 1000, "Victime1")
	insertKillRow(t, db, "m1", "passeB", 2000, "Victime2")

	var n int
	var pass string
	if err := db.QueryRow(`SELECT COUNT(*), MIN(decode_pass) FROM match_kill_events_latest
		WHERE match_id = 'm1'`).Scan(&n, &pass); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 2 {
		t.Errorf("vue = %d lignes, attendu 2 (la passe B entiere) — une vue « derniere ligne "+
			"par mort » en rendrait 3 et ferait survivre une mort que le decodeur ne publie plus", n)
	}
	if pass != "passeB" {
		t.Errorf("vue sert la passe %q, attendu passeB", pass)
	}
	if got := countRows(t, db, "match_kill_events"); got != 5 {
		t.Errorf("table = %d lignes, attendu 5 — append-only : rien n est efface", got)
	}
}

// TestVuePasseIndependanteParMatch — la selection de passe est PARTITIONNEE par match : une
// passe recente sur m2 ne doit pas masquer la passe de m1.
func TestVuePasseIndependanteParMatch(t *testing.T) {
	db := killEventsDB(t)
	insertKillRow(t, db, "m1", "p1", 1000, "V1")
	insertKillRow(t, db, "m2", "p2", 1000, "V2")

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 2 {
		t.Errorf("vue = %d lignes, attendu 2 (une par match)", n)
	}
}

// TestResiduDegatsQuatreCasNull — « NULL n est jamais zero ».
//
// Le 3e cas est celui que la premiere version du DDL avait FAUX : un assistant NOMME dont la
// part n a pas ete lue tombait dans le cas solo (total 100) au lieu de 99, donc un residu faux
// de 1, silencieux.
func TestResiduDegatsQuatreCasNull(t *testing.T) {
	db := killEventsDB(t)

	cas := []struct {
		nom        string
		killerPct  any
		assistKnwn bool
		assistName any
		assistPct  any
		attendu    any // nil = NULL
	}{
		{"part du tueur non lue", nil, true, nil, nil, nil},
		{"assistant inconnu (on ne sait pas)", 60, false, nil, nil, nil},
		{"assistant NOMME, part non lue", 60, true, "Assist", nil, nil},
		{"solo mesure : total 100", 70, true, nil, nil, 30},
		{"avec assistant : total 99", 60, true, "Assist", 30, 9},
		// La population > 100 sort telle quelle : un residu NEGATIF est une donnee, pas un bug.
		{"part > 100 (1,7 % du corpus)", 228, true, nil, nil, -128},
	}
	for i, c := range cas {
		_, err := db.Exec(`
			INSERT INTO match_kill_events (
				match_id, decode_pass, decoder_rev, publishable, time_ms,
				victim_gamertag, feed_present, assist_known, assist_gamertag,
				killer_damage_pct, assist_damage_pct, read_path, read_origin
			) VALUES (?, 'p', 'test-rev', TRUE, ?, 'V', TRUE, ?, ?,
				CAST(? AS UTINYINT), CAST(? AS UTINYINT), ?, 'credit-concordant')`,
			c.nom, i, c.assistKnwn, c.assistName, c.killerPct, c.assistPct,
			killscope.ReadPathFilmWalk)
		if err != nil {
			t.Fatalf("insert %q: %v", c.nom, err)
		}

		var residu sql.NullInt64
		if err := db.QueryRow(`SELECT damage_pct_residual FROM match_kill_events_latest
			WHERE match_id = ?`, c.nom).Scan(&residu); err != nil {
			t.Fatalf("select %q: %v", c.nom, err)
		}
		if c.attendu == nil {
			if residu.Valid {
				t.Errorf("%s : residu = %d, attendu NULL (un terme manque, "+
					"et une absence de mesure n est pas un zero)", c.nom, residu.Int64)
			}
			continue
		}
		if !residu.Valid || residu.Int64 != int64(c.attendu.(int)) {
			t.Errorf("%s : residu = %v, attendu %v", c.nom, residu, c.attendu)
		}
	}
}

// TestVueSuitColonneAjoutee — le COUT de la migration future « table fille d assistants » est
// mesure, pas suppose : la vue `SELECT e.*` suit une colonne ajoutee par ALTER sans qu il
// faille la re-creer. Si une version future de DuckDB figeait la liste de colonnes a la
// creation de la vue, ce test sonne et la parade est deja ecrite (re-executer
// ddlMatchKillEventsLatest, l unique copie de ce SQL).
func TestVueSuitColonneAjoutee(t *testing.T) {
	db := killEventsDB(t)
	insertKillRow(t, db, "m1", "p1", 1000, "V1")

	mustExec(t, db, `ALTER TABLE match_kill_events ADD COLUMN assist_gamertag_2 VARCHAR`)

	var col sql.NullString
	if err := db.QueryRow(`SELECT assist_gamertag_2 FROM match_kill_events_latest`).Scan(&col); err != nil {
		t.Fatalf("la vue ne suit PAS la colonne ajoutee (%v) — re-executer "+
			"ddlMatchKillEventsLatest apres l ALTER devient obligatoire, cf. en-tete du DDL", err)
	}
}

// TestSourceEtDivergenceNullables — la source du degat est FACULTATIVE en base. Une ligne du
// producteur « credit seul » (highlight_events) doit passer sans tag, sans categorie et sans
// divergence : exiger la source ferait d elle la condition d existence d une mort, et
// exclurait les ~28 % de matchs dont le film a expire cote serveur.
func TestSourceEtDivergenceNullables(t *testing.T) {
	db := killEventsDB(t)
	// La portée vient de `killscope`, pas d'un littéral : un seed qui figerait sa propre copie
	// continuerait de passer au vert le jour où la valeur de production change.
	_, err := db.Exec(`
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, publishable, time_ms,
			victim_gamertag, feed_killer_gamertag, feed_present, assist_known,
			read_path, read_origin
		) VALUES ('m1', 'p1', 'highlight-v1', TRUE, 1000, 'V1', 'K1', TRUE, FALSE, ?, ?)`,
		killscope.ReadPathCreditBackfill, killscope.OriginCreditOnly)
	if err != nil {
		t.Fatalf("le producteur credit-seul doit pouvoir ecrire sans source: %v", err)
	}

	var tag, cat, div sql.NullString
	if err := db.QueryRow(`SELECT source_tag, source_category, diverges
		FROM match_kill_events_latest`).Scan(&tag, &cat, &div); err != nil {
		t.Fatalf("select: %v", err)
	}
	if tag.Valid || cat.Valid || div.Valid {
		t.Errorf("source non mesuree : les trois colonnes doivent etre NULL, got tag=%v cat=%v div=%v",
			tag, cat, div)
	}
}

// TestMigrationNeToucheNiKillerVictimPairs — la remplacante n est pas un remplacement : tant
// que ses 8 lecteurs n ont pas migre, `killer_victim_pairs` reste intacte. Le test tient
// l engagement ecrit dans l en-tete du DDL.
func TestMigrationNeToucheNiKillerVictimPairs(t *testing.T) {
	db := openTmpDB(t)
	mustExec(t, db, `CREATE TABLE killer_victim_pairs (match_id VARCHAR, kill_count INTEGER)`)
	mustExec(t, db, `INSERT INTO killer_victim_pairs VALUES ('m1', 1)`)

	if err := applyMatchKillEvents(db); err != nil {
		t.Fatalf("applyMatchKillEvents: %v", err)
	}
	if got := countRows(t, db, "killer_victim_pairs"); got != 1 {
		t.Errorf("killer_victim_pairs = %d lignes, attendu 1 — la migration ne doit NI la "+
			"supprimer NI la modifier", got)
	}
}
