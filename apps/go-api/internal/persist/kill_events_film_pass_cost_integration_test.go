//go:build integration

// Package persist — kill_events_film_pass_cost_integration_test.go : LE COUT DE LA LECTURE PAR
// MATCH DE LA PASSE DE FILM, MESURE SUR LES VRAIES MIGRATIONS.
//
// # LE DEFAUT QUE CE TEST MESURE (lot 5.12, 2026-09-21)
//
// `levelup backfill-killsource` joue apres la passe des films une passe CREDIT-SEUL qui examine
// TOUS les matchs du registre et, pour chacun, relit l enrichissement deja en base
// ([FilmPassForMatch]). Mesure de production du 2026-09-21 : la passe des films a decode
// 1 598 films en 4 h 14, puis la passe credit (« 9 144 matchs a examiner ») a tourne PLUS DE
// 22 HEURES sans terminer ni journaliser une ligne de progression.
//
// La cause est dans la FORME de la lecture, pas dans le volume : la vue `match_kill_events_latest`
// choisit la derniere passe par un `QUALIFY … FIRST_VALUE() OVER (PARTITION BY match_id …)`. Une
// fenetre se calcule sur TOUTE la relation d entree ; si le `WHERE match_id = ?` du lecteur reste
// AU-DESSUS d elle, chaque match paie un balayage complet d une table append-only qui grossit de
// ~160 000 lignes a chaque backfill. Le cout total est alors quadratique : 9 144 matchs x toute la
// table.
//
// CE TEST EST LA MESURE, ET IL RESTE COMME GARDE-FOU : il peuple une table de la MEME FORME que la
// production (2 000 matchs x 3 passes x 30 morts = 180 000 lignes), colle les plans DuckDB
// (`EXPLAIN`) et exige que la lecture DE PRODUCTION tienne un budget par appel. Sans budget, la
// regression serait invisible — elle ne casse rien, elle rend seulement la passe interminable.
package persist

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Les dimensions du banc. `nbMatchsBanc` x `nbPassesBanc` x `nbMortsBanc` = 180 000 lignes, soit
// l ordre de grandeur d UNE campagne de backfill de production (~160 000 lignes ajoutees).
const (
	nbMatchsBanc = 2000
	nbPassesBanc = 3
	nbMortsBanc  = 30
	// nbLecturesBanc : le nombre d appels chronometres. 200 sort du bruit et garde le test court
	// meme quand la lecture est le defaut quadratique — qui, lui, ne tiendrait pas 9 144 appels.
	nbLecturesBanc = 200
	// budgetParAppel : LE BUDGET. Une lecture par match doit couter un acces a UN match, pas un
	// balayage de la table. 10 ms par appel = les 9 144 matchs du registre en moins de deux
	// minutes ; le defaut mesure, lui, met des dizaines de millisecondes par appel des
	// 180 000 lignes, et s aggrave a chaque backfill.
	budgetParAppel = 10 * time.Millisecond
)

// peuplerBancKillEvents remplit `match_kill_events` : `nbMatchsBanc` matchs, chacun avec
// `nbPassesBanc` passes horodatees dans l ordre, chaque passe portant `nbMortsBanc` morts de voie
// FILM (`marche`) — la population que [FilmPassForMatch] relit.
//
// L insertion est un `INSERT … SELECT` de fixture : un INSERT pur sur une table append-only, la
// seule ecriture qu ADR 0026 y autorise. Passer par le persister ferait 6 000 transactions pour
// une donnee dont seule la FORME compte ici.
func peuplerBancKillEvents(t *testing.T, db *sql.DB) {
	t.Helper()
	debut := time.Now()
	_, err := db.Exec(`
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, written_at, publishable, time_ms,
			victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid, feed_present,
			assist_known, assist_extra_count, killer_damage_pct, read_path, read_origin)
		SELECT
			printf('match-%05d', m.i),
			printf('match-%05d-passe-%d', m.i, p.i),
			'rev-banc',
			TIMESTAMP '2026-01-01 00:00:00' + to_minutes(CAST(p.i AS INTEGER)),
			TRUE,
			CAST(d.i AS INTEGER) * 1000,
			printf('victime-%02d', d.i), printf('xuid-v-%02d', d.i),
			printf('tueur-%02d', d.i), printf('xuid-t-%02d', d.i), TRUE,
			FALSE, 0, 70, ?, 'credit-concordant'
		FROM range(0, ?) AS m(i), range(0, ?) AS p(i), range(0, ?) AS d(i)`,
		FilmReadPaths[0], nbMatchsBanc, nbPassesBanc, nbMortsBanc)
	if err != nil {
		t.Fatalf("peuplement du banc: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events`).Scan(&n); err != nil {
		t.Fatalf("comptage du banc: %v", err)
	}
	if want := nbMatchsBanc * nbPassesBanc * nbMortsBanc; n != want {
		t.Fatalf("banc: %d lignes, attendu %d", n, want)
	}
	t.Logf("banc peuple : %d lignes (%d matchs x %d passes x %d morts) en %s",
		n, nbMatchsBanc, nbPassesBanc, nbMortsBanc, time.Since(debut).Round(time.Millisecond))
}

// collerPlan execute un `EXPLAIN` et colle le plan dans la sortie du test. C est LA PIECE : elle
// dit si le filtre par match est applique SOUS la fenetre (un seul match balaye) ou au-dessus
// (toute la table balayee, a chaque appel).
func collerPlan(t *testing.T, db *sql.DB, titre, requete string, args ...any) {
	t.Helper()
	rows, err := db.Query("EXPLAIN "+requete, args...)
	if err != nil {
		t.Fatalf("EXPLAIN %s: %v", titre, err)
	}
	defer func() { _ = rows.Close() }()
	var plan strings.Builder
	for rows.Next() {
		var cle, valeur string
		if err := rows.Scan(&cle, &valeur); err != nil {
			t.Fatalf("EXPLAIN %s (scan): %v", titre, err)
		}
		plan.WriteString(valeur)
		plan.WriteString("\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("EXPLAIN %s (rows): %v", titre, err)
	}
	t.Logf("PLAN — %s\n%s", titre, plan.String())
}

// chronometrer joue `nbLecturesBanc` lectures et rend la duree MOYENNE par appel.
func chronometrer(t *testing.T, lire func(matchID string) (int, error)) time.Duration {
	t.Helper()
	debut := time.Now()
	lignes := 0
	for i := 0; i < nbLecturesBanc; i++ {
		n, err := lire(fmt.Sprintf("match-%05d", i))
		if err != nil {
			t.Fatalf("lecture %d: %v", i, err)
		}
		lignes += n
	}
	total := time.Since(debut)
	if lignes != nbLecturesBanc*nbMortsBanc {
		t.Fatalf("lectures: %d lignes rendues, attendu %d", lignes, nbLecturesBanc*nbMortsBanc)
	}
	return total / nbLecturesBanc
}

// TestFilmPassForMatch_CoutParAppel — LE BUDGET DE LA LECTURE DE PRODUCTION.
//
// Rouge = la passe credit du backfill est redevenue quadratique.
func TestFilmPassForMatch_CoutParAppel(t *testing.T) {
	db := baseDeTest(t)
	peuplerBancKillEvents(t, db)
	ctx := context.Background()

	// Le plan de la lecture de production, telle qu elle est ECRITE dans le code (`filmPassQuery`,
	// la seule definition — jamais une copie).
	requete, args := filmPassQuery("match-00001")
	collerPlan(t, db, "production — FilmPassForMatch", requete, args...)

	moyenne := chronometrer(t, func(matchID string) (int, error) {
		batch, err := FilmPassForMatch(ctx, db, matchID)
		return len(batch.Deaths), err
	})
	t.Logf("FilmPassForMatch : %s par appel sur %d lignes (%d appels) — budget %s",
		moyenne.Round(time.Microsecond), nbMatchsBanc*nbPassesBanc*nbMortsBanc,
		nbLecturesBanc, budgetParAppel)
	if moyenne > budgetParAppel {
		t.Fatalf("FilmPassForMatch coute %s par appel, budget %s : la lecture par match balaie "+
			"la table append-only entiere (lot 5.12)", moyenne, budgetParAppel)
	}
}

// TestFilmPassForMatch_LeDefautDeLaVueGenerale — LA MESURE DU DEFAUT, gardee comme piece.
//
// Elle chronometre la MEME lecture ecrite sur la vue generale `match_kill_events_latest` filtree
// par match, c est-a-dire la forme qu avait la production avant le lot 5.12. Le test ne l exige
// pas lente (une version de DuckDB pourrait pousser le predicat) : il colle son plan et son temps
// A COTE de ceux de la lecture de production, pour que l ecart soit une mesure et non une croyance.
func TestFilmPassForMatch_LeDefautDeLaVueGenerale(t *testing.T) {
	db := baseDeTest(t)
	peuplerBancKillEvents(t, db)
	ctx := context.Background()

	const requete = `SELECT time_ms FROM match_kill_events_latest
		WHERE match_id = ? AND read_path IN (?, ?) ORDER BY time_ms, id`
	collerPlan(t, db, "defaut — vue generale filtree par match", requete,
		"match-00001", FilmReadPaths[0], FilmReadPaths[1])

	moyenne := chronometrer(t, func(matchID string) (int, error) {
		rows, err := db.QueryContext(ctx, requete, matchID, FilmReadPaths[0], FilmReadPaths[1])
		if err != nil {
			return 0, err
		}
		defer func() { _ = rows.Close() }()
		n := 0
		for rows.Next() {
			var ms int
			if err := rows.Scan(&ms); err != nil {
				return 0, err
			}
			n++
		}
		return n, rows.Err()
	})
	t.Logf("vue generale filtree par match : %s par appel sur %d lignes (%d appels)",
		moyenne.Round(time.Microsecond), nbMatchsBanc*nbPassesBanc*nbMortsBanc, nbLecturesBanc)
}
