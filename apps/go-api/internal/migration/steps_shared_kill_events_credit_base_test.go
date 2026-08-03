//go:build cgo

package migration

// steps_shared_kill_events_credit_base_test.go — L INVERSION DE PRESEANCE, JOUEE SUR UNE BASE
// PEUPLEE.
//
// Meme discipline que la reprise precedente (constat J4R-1) : une migration jouee sur base VIDE
// ne prouve rien. Chacune des proprietes ci-dessous peut se perdre en retirant exactement un
// fragment de la requete, et AUCUNE ne se voit a l execution — la migration passe, la table se
// remplit, aucun compteur ne bouge. Ce qui change est le nombre de lignes SERVIES.
//
// Les proprietes, et la mutation qui doit faire rougir chacune :
//
//	la base credit est la liste des morts     retirer la branche « base » de l UNION
//	le film ENRICHIT, il ne remplace pas      retirer le LEFT JOIN `appariable`
//	les orphelins sont conserves              retirer la seconde branche de l UNION
//	la deduplication porte sur l IDENTITE     retirer le GROUP BY (ou le porter sur la clef seule)
//	la revision du decodeur survit            remplacer `COALESCE(p.rev, ?)` par `?`
//	l idempotence                             retirer la CTE `aprendre`

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
)

// coupleA : un couple avec une victime et un tueur EXPLICITES — l identite est ce qui deduplique.
func coupleA(t *testing.T, db *sql.DB, matchID, victimeXUID, victimeNom string, timeMS int) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO killer_victim_pairs
			(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms)
		VALUES (?, '1001', 'Tueur', ?, ?, 1, ?)`, matchID, victimeXUID, victimeNom, timeMS); err != nil {
		t.Fatalf("insert couple (%s@%d): %v", matchID, timeMS, err)
	}
}

// ligneDeFilm ecrit UNE ligne de passe de film a un instant donne. `source_tag` est renseigne :
// c est precisement ce que la base credit ne sait pas mesurer, donc le temoin de l enrichissement.
func ligneDeFilm(t *testing.T, db *sql.DB, matchID string, timeMS int, victime string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, written_at, publishable, time_ms,
			victim_gamertag, feed_present, assist_known, assist_gamertag,
			source_tag, source_category, read_path, read_origin
		) VALUES (?, ?, 'film-rev', TIMESTAMP '2026-01-01 00:00:00', TRUE, ?,
			?, TRUE, TRUE, 'Assistant', 3735928559, 'Headshot', ?, 'credit-concordant')`,
		matchID, "film-"+matchID, timeMS, victime, killscope.ReadPathFilmWalk); err != nil {
		t.Fatalf("insert ligne de film (%s@%d): %v", matchID, timeMS, err)
	}
}

// TestRepriseCreditBase_LeFilmEnrichitSansRemplacer — LE SCENARIO DE LA SESSION 3, INVERSE.
//
// Le match porte TROIS morts au credit (1000, 5000, 9000) et une passe de film qui n en couvre
// qu UNE (1000) plus une ligne que le credit ne porte pas (2000 — une mort de BOT en production,
// que le kill-feed humain-seul de l API ne peut pas representer).
//
// AVANT l inversion, la vue servait 2 lignes : celles du film. Les morts de 5000 et 9000
// disparaissaient de la lecture, sans erreur ni compteur. C est ce defaut, multiplie par
// 25 697 morts sur 949 matchs, que cette migration referme.
func TestRepriseCreditBase_LeFilmEnrichitSansRemplacer(t *testing.T) {
	db := basePourReprise(t)
	coupleA(t, db, "m-film", "1002", "Victime", 1000)
	coupleA(t, db, "m-film", "1002", "Victime", 5000)
	coupleA(t, db, "m-film", "1002", "Victime", 9000)
	ligneDeFilm(t, db, "m-film", 1000, "Victime")
	ligneDeFilm(t, db, "m-film", 2000, "VictimeBot")

	if err := applyKillEventsCreditBase(db); err != nil {
		t.Fatalf("applyKillEventsCreditBase: %v", err)
	}

	type ligne struct {
		voie      string
		avecTag   bool
		assistant sql.NullString
	}
	servies := map[int]ligne{}
	rows, err := db.Query(`SELECT time_ms, read_path, source_tag IS NOT NULL, assist_gamertag
		FROM match_kill_events_latest WHERE match_id = 'm-film'`)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var instant int
		var l ligne
		if err := rows.Scan(&instant, &l.voie, &l.avecTag, &l.assistant); err != nil {
			t.Fatalf("scan: %v", err)
		}
		servies[instant] = l
	}

	if len(servies) != 4 {
		t.Fatalf("%d ligne(s) servies, attendu 4 = 3 morts de credit + 1 orpheline de film. "+
			"Moins de 4 veut dire qu une mort a disparu de la lecture : instants servis %v",
			len(servies), servies)
	}
	if l := servies[1000]; !l.avecTag || l.voie != killscope.ReadPathFilmWalk ||
		l.assistant.String != "Assistant" {
		t.Errorf("mort a 1000 ms = %+v — elle doit porter l ENRICHISSEMENT du film "+
			"(source du degat, voie du film, assistant nomme)", l)
	}
	for _, instant := range []int{5000, 9000} {
		if l := servies[instant]; l.avecTag || l.voie != killscope.ReadPathLiveFeed ||
			l.assistant.Valid {
			t.Errorf("mort a %d ms = %+v — le film ne la porte pas : sa source doit rester NON "+
				"MESUREE (NULL, jamais zero) et sa voie celle du credit", instant, l)
		}
	}
	if l := servies[2000]; !l.avecTag || l.voie != killscope.ReadPathFilmWalk {
		t.Errorf("orpheline a 2000 ms = %+v — une mort de bot mesuree par le film ne doit ni "+
			"disparaitre ni changer de portee", l)
	}
}

// TestRepriseCreditBase_DedupliqueSurLIdentite — 133 886 ET 268 337, PAS 145 974 NI 268 330.
//
// Deux pieges opposes, mesures le 2026-08-03 :
//
//	dedup sur la CLEF `(match_id, time_ms)`   PERD 7 morts REELLES sur Halo 5 (deux victimes
//	                                          distinctes a la meme milliseconde)
//	dedup sur TOUTES LES COLONNES             FABRIQUE 12 088 doublons sur Halo Infinite (une
//	                                          meme mort sous deux orthographes de gamertag)
//
// Le test reproduit les deux en miniature : une mort ecrite deux fois sous deux orthographes
// (une seule ligne attendue) et deux morts DIFFERENTES au meme instant (deux lignes attendues).
func TestRepriseCreditBase_DedupliqueSurLIdentite(t *testing.T) {
	db := basePourReprise(t)
	// Meme mort, deux orthographes du nom de la victime : UNE ligne.
	coupleA(t, db, "m-dedup", "1002", "Victime", 1000)
	coupleA(t, db, "m-dedup", "1002", "Victime_renommee", 1000)
	// Deux morts DIFFERENTES au meme instant : DEUX lignes.
	coupleA(t, db, "m-dedup", "1003", "AutreVictime", 4000)
	coupleA(t, db, "m-dedup", "1004", "TroisiemeVictime", 4000)

	if err := applyKillEventsCreditBase(db); err != nil {
		t.Fatalf("applyKillEventsCreditBase: %v", err)
	}

	var a1000, a4000 int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id='m-dedup' AND time_ms=1000),
		(SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id='m-dedup' AND time_ms=4000)`).
		Scan(&a1000, &a4000); err != nil {
		t.Fatalf("comptes: %v", err)
	}
	if a1000 != 1 {
		t.Errorf("%d ligne(s) a 1000 ms, attendu 1 — la meme mort sous deux orthographes de "+
			"gamertag a produit deux morts (12 088 doublons a l echelle du corpus Infinite)", a1000)
	}
	if a4000 != 2 {
		t.Errorf("%d ligne(s) a 4000 ms, attendu 2 — deux morts REELLES au meme instant ont ete "+
			"fondues en une (7 morts perdues a l echelle du corpus Halo 5)", a4000)
	}
}

// TestRepriseCreditBase_GardeLaRevisionDuDecodeur — ne pas faire redecoder 3 a 11 heures pour rien.
//
// Un match a film garde la revision du DECODEUR : c est sur elle que
// `levelup backfill-killsource` decide qu un match est a jour. L ecraser ferait redecoder tout le
// corpus. Un match sans film porte celle de la reprise.
func TestRepriseCreditBase_GardeLaRevisionDuDecodeur(t *testing.T) {
	db := basePourReprise(t)
	coupleA(t, db, "m-film", "1002", "Victime", 1000)
	ligneDeFilm(t, db, "m-film", 1000, "Victime")
	coupleA(t, db, "m-sans-film", "1002", "Victime", 1000)

	if err := applyKillEventsCreditBase(db); err != nil {
		t.Fatalf("applyKillEventsCreditBase: %v", err)
	}

	var avecFilm, sansFilm string
	if err := db.QueryRow(`SELECT
		(SELECT ANY_VALUE(decoder_rev) FROM match_kill_events_latest WHERE match_id='m-film'),
		(SELECT ANY_VALUE(decoder_rev) FROM match_kill_events_latest WHERE match_id='m-sans-film')`).
		Scan(&avecFilm, &sansFilm); err != nil {
		t.Fatalf("revisions: %v", err)
	}
	if avecFilm != "film-rev" {
		t.Errorf("decoder_rev du match a film = %q, attendu film-rev — le backfill ne "+
			"reconnaitrait plus les matchs deja decodes et redecoderait tout le corpus", avecFilm)
	}
	if sansFilm != CreditBaseRebuildDecoderRev {
		t.Errorf("decoder_rev du match sans film = %q, attendu %q", sansFilm, CreditBaseRebuildDecoderRev)
	}
}

// TestRepriseCreditBase_EstIdempotente — rejouee, elle ne double rien.
//
// La marque est le prefixe du `decode_pass`. Sans elle, une seconde execution ecrirait une
// seconde passe portant le MEME `decode_pass` — et la vue `_latest`, qui selectionne par
// `decode_pass`, rendrait alors LES DEUX. Le doublon reviendrait par la porte de la reprise.
func TestRepriseCreditBase_EstIdempotente(t *testing.T) {
	db := basePourReprise(t)
	coupleA(t, db, "m-idem", "1002", "Victime", 1000)
	coupleA(t, db, "m-idem", "1002", "Victime", 2000)

	for i := 0; i < 2; i++ {
		if err := applyKillEventsCreditBase(db); err != nil {
			t.Fatalf("applyKillEventsCreditBase (passage %d): %v", i+1, err)
		}
	}

	var vue, table int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id='m-idem'),
		(SELECT COUNT(*) FROM match_kill_events        WHERE match_id='m-idem')`).
		Scan(&vue, &table); err != nil {
		t.Fatalf("comptes: %v", err)
	}
	if vue != 2 || table != 2 {
		t.Errorf("vue=%d table=%d, attendu 2/2 — la reprise s est rejouee et a double les morts",
			vue, table)
	}
}

// TestRepriseCreditBase_RefuseUnePasseAppauvrissante — L INVARIANT, RELU APRES COUP.
//
// La verification post-ecriture doit ECHOUER si un match publie moins de morts que sa base
// credit. Le scenario est fabrique : un match dont la passe courante (une passe de film ecrite a
// la main, avec un `decode_pass` deja prefixe pour que la reprise le saute) porte moins de lignes
// que ses couples. C est exactement l etat dans lequel la session 3 a laisse 949 matchs.
func TestRepriseCreditBase_RefuseUnePasseAppauvrissante(t *testing.T) {
	db := basePourReprise(t)
	coupleA(t, db, "m-maigre", "1002", "Victime", 1000)
	coupleA(t, db, "m-maigre", "1002", "Victime", 5000)
	// `decode_pass` deja prefixe : la CTE `aprendre` saute ce match, la passe maigre reste servie.
	if _, err := db.Exec(`
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, written_at, publishable, time_ms,
			victim_gamertag, feed_present, assist_known, read_path, read_origin
		) VALUES ('m-maigre', 'creditbase-m-maigre', 'film-rev', TIMESTAMP '2026-01-01 00:00:00',
			TRUE, 1000, 'Victime', TRUE, TRUE, ?, 'credit-concordant')`,
		killscope.ReadPathFilmWalk); err != nil {
		t.Fatalf("seed passe maigre: %v", err)
	}

	err := applyKillEventsCreditBase(db)
	if err == nil {
		t.Fatal("la reprise a rendu OK alors qu un match publie 1 mort pour une base credit de 2 " +
			"— l invariant n est plus relu, et l ecart disparaitrait de la lecture sans erreur " +
			"ni compteur")
	}
}
