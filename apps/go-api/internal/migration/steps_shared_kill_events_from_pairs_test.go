//go:build cgo

package migration

// steps_shared_kill_events_from_pairs_test.go — LA MIGRATION DE REPRISE, JOUEE SUR UNE BASE
// PEUPLEE.
//
// POURQUOI CE FICHIER EXISTE (constat J4R-1). La reprise n avait aucun test COMPORTEMENTAL : les
// tests du lot la jouaient sur une base VIDE, ou verifiaient qu elle ne touche pas
// `killer_victim_pairs`. Ses trois proprietes — deduplication, une passe par match, preseance du
// film — reposaient donc sur la seule lecture du SQL. Chacune peut se perdre en retirant
// exactement un fragment de la requete, et AUCUNE ne se voit a l execution : la migration
// continue de passer, la table se remplit, aucun compteur ne bouge. Ce qui change est le nombre
// de lignes SERVIES, ou quelle generation est servie.
//
// Les trois tests ci-dessous sont ecrits pour ROUGIR sur ces mutations precises, et cela a ete
// verifie mutation par mutation (cf. journal du plan, ronde de correction J4R).

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
)

// ─── Harnais : une base partagee reduite a ce que la reprise touche ────────────────────────

// basePourReprise : `match_kill_events` + sa vue, `killer_victim_pairs` et les deux tables que
// `v_gamertag_lookup` joint. La migration recree cette vue en fin de course — sans elles, on
// testerait une erreur de binding au lieu de la reprise.
func basePourReprise(t *testing.T) *sql.DB {
	t.Helper()
	db := openTmpDB(t)
	if err := applyMatchKillEvents(db); err != nil {
		t.Fatalf("applyMatchKillEvents: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE killer_victim_pairs (
			match_id        VARCHAR NOT NULL,
			killer_xuid     VARCHAR NOT NULL,
			killer_gamertag VARCHAR,
			victim_xuid     VARCHAR NOT NULL,
			victim_gamertag VARCHAR,
			kill_count      INTEGER DEFAULT 1,
			time_ms         INTEGER,
			is_validated    BOOLEAN DEFAULT FALSE,
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("ddl harnais: %v", err)
		}
	}
	return db
}

// couple ecrit UNE ligne de `killer_victim_pairs`, forme par-kill (celle de la completion).
func couple(t *testing.T, db *sql.DB, matchID, victime string, timeMS int) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO killer_victim_pairs
			(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms)
		VALUES (?, '1001', 'Tueur', '1002', ?, 1, ?)`, matchID, victime, timeMS); err != nil {
		t.Fatalf("insert couple (%s/%s): %v", matchID, victime, err)
	}
}

// passeDeFilm ecrit une passe de FILM sur un match : c est la generation que la reprise ne doit
// jamais supplanter. Aucun film n est necessaire — ce qui decide la preseance est la VOIE de
// lecture, et elle s ecrit.
//
// `source_tag` est renseigne (identifiant jpt! 32 bits, UINTEGER) parce que c est PRECISEMENT ce
// que la reprise ne sait pas mesurer : sa presence dans la generation servie est le temoin que le
// film est bien celui qu on lit.
func passeDeFilm(t *testing.T, db *sql.DB, matchID string, morts int) {
	t.Helper()
	for i := 0; i < morts; i++ {
		if _, err := db.Exec(`
			INSERT INTO match_kill_events (
				match_id, decode_pass, decoder_rev, publishable, time_ms,
				victim_gamertag, feed_present, assist_known, source_tag, read_path, read_origin
			) VALUES (?, ?, 'film-rev', TRUE, ?, 'VictimeFilm', TRUE, TRUE, 3735928559, 'marche',
				'credit-concordant')`,
			matchID, "film-"+matchID, 100*(i+1)); err != nil {
			t.Fatalf("insert passe de film (%s): %v", matchID, err)
		}
	}
}

func compterVue(t *testing.T, db *sql.DB, matchID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?`, matchID).Scan(&n); err != nil {
		t.Fatalf("count vue (%s): %v", matchID, err)
	}
	return n
}

// ─── PROPRIETE 1 : LA DEDUPLICATION ────────────────────────────────────────────────────────

// TestRepriseDedupliqueLesCouples — `SELECT DISTINCT`, et ce qu il coute de le retirer.
//
// Le defaut d origine est structurel : le flux primaire INSERT sans supprimer pendant que la
// completion fait DELETE-then-INSERT, et rien dans le schema n interdit au meme couple d exister
// deux fois. 250 139 lignes pour 133 886 cles en production, soit 46,5 % — et des agregats
// carriere gonfles d un facteur 2,006.
//
// Une reprise sans DISTINCT recopierait ce defaut TEL QUEL dans la table censee le supprimer.
func TestRepriseDedupliqueLesCouples(t *testing.T) {
	db := basePourReprise(t)

	// Deux morts REELLES, chacune ecrite DEUX FOIS : la forme exacte du doublon de production.
	couple(t, db, "m-dup", "Victime-A", 1000)
	couple(t, db, "m-dup", "Victime-A", 1000)
	couple(t, db, "m-dup", "Victime-B", 5000)
	couple(t, db, "m-dup", "Victime-B", 5000)

	if err := applyKillEventsFromPairs(db); err != nil {
		t.Fatalf("applyKillEventsFromPairs: %v", err)
	}

	if n := compterVue(t, db, "m-dup"); n != 2 {
		t.Errorf("la vue sert %d morts pour 2 morts reelles (4 lignes sources dont 2 doublons) — "+
			"la reprise a recopie le doublon de killer_victim_pairs (46,5 %% en production, "+
			"facteur 2,006 sur les agregats carriere) dans la table qui doit le rendre impossible", n)
	}
}

// ─── PROPRIETE 2 : UNE PASSE PAR MATCH ─────────────────────────────────────────────────────

// TestRepriseEcritUnePasseParMatch — l unite de generation de la vue `_latest` est LE MATCH.
//
// `decode_pass` derive du `match_id` (`'pairs-' || kvp.match_id`). Une passe unique pour toute la
// reprise ferait de l ensemble des matchs repris UNE SEULE generation : le decoupage par match
// disparaitrait, et avec lui la possibilite de remplacer un match sans toucher aux autres.
func TestRepriseEcritUnePasseParMatch(t *testing.T) {
	db := basePourReprise(t)
	couple(t, db, "m-un", "Victime-A", 1000)
	couple(t, db, "m-un", "Victime-B", 2000)
	couple(t, db, "m-deux", "Victime-C", 3000)

	if err := applyKillEventsFromPairs(db); err != nil {
		t.Fatalf("applyKillEventsFromPairs: %v", err)
	}

	// (a) UNE seule passe par match — pas deux, ce qui melangerait deux generations.
	rows, err := db.Query(`
		SELECT match_id, COUNT(DISTINCT decode_pass)
		FROM match_kill_events GROUP BY match_id ORDER BY match_id`)
	if err != nil {
		t.Fatalf("select passes: %v", err)
	}
	defer func() { _ = rows.Close() }()
	passes := map[string]int{}
	for rows.Next() {
		var m string
		var n int
		if err := rows.Scan(&m, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		passes[m] = n
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	for _, m := range []string{"m-un", "m-deux"} {
		if passes[m] != 1 {
			t.Errorf("match %s : %d passes de reprise, attendu 1", m, passes[m])
		}
	}

	// (b) et elles sont DISTINCTES d un match a l autre. Une passe constante pour toute la
	//     reprise ferait d elle une generation unique couvrant N matchs — le contraire de ce que
	//     la vue `_latest` decoupe.
	var passesDistinctes int
	if err := db.QueryRow(
		`SELECT COUNT(DISTINCT decode_pass) FROM match_kill_events`).Scan(&passesDistinctes); err != nil {
		t.Fatalf("count passes distinctes: %v", err)
	}
	if passesDistinctes != 2 {
		t.Errorf("%d passe(s) distincte(s) pour 2 matchs repris — la passe ne derive plus du "+
			"match_id, donc la reprise n est plus remplacable match par match", passesDistinctes)
	}

	// (c) et chaque match sert bien SES morts, pas celles du voisin.
	if n := compterVue(t, db, "m-un"); n != 2 {
		t.Errorf("m-un sert %d morts, attendu 2", n)
	}
	if n := compterVue(t, db, "m-deux"); n != 1 {
		t.Errorf("m-deux sert %d morts, attendu 1", n)
	}
}

// ─── PROPRIETE 3 : LA PRESEANCE ────────────────────────────────────────────────────────────

// TestRepriseNeSupplantePasUnePasseExistante — LE test le plus cher a perdre.
//
// Le filtre `match_id NOT IN (SELECT match_id FROM match_kill_events_latest)` est la seule chose
// qui empeche la reprise de devenir la generation servie sur un match deja couvert. Le retirer
// n aurait AUCUN symptome : meme nombre de morts servies, memes noms, memes instants. Seule la
// source du degat fatal — ce que le film seul mesure — disparaitrait de la lecture.
func TestRepriseNeSupplantePasUnePasseExistante(t *testing.T) {
	db := basePourReprise(t)

	// Le match porte les DEUX : une passe de film DEJA ecrite, et ses couples historiques.
	passeDeFilm(t, db, "m-film", 2)
	couple(t, db, "m-film", "Victime-A", 1000)
	couple(t, db, "m-film", "Victime-B", 2000)
	couple(t, db, "m-film", "Victime-C", 3000)
	// Un match temoin SANS film : lui doit bien etre repris, sinon le test passerait au vert
	// pour une reprise qui n ecrit tout simplement rien.
	couple(t, db, "m-sans-film", "Victime-D", 4000)

	if err := applyKillEventsFromPairs(db); err != nil {
		t.Fatalf("applyKillEventsFromPairs: %v", err)
	}

	// (a) le temoin PROUVE que la reprise a bien tourne.
	if n := compterVue(t, db, "m-sans-film"); n != 1 {
		t.Fatalf("le match sans film sert %d morts, attendu 1 — la reprise n a rien ecrit, "+
			"le reste du test ne prouverait rien", n)
	}

	// (b) le match couvert n a recu AUCUNE ligne de reprise, meme non servie.
	var lignesDeReprise int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events
		WHERE match_id = ? AND decoder_rev = ?`,
		"m-film", PairsBackfillDecoderRev).Scan(&lignesDeReprise); err != nil {
		t.Fatalf("count lignes de reprise: %v", err)
	}
	if lignesDeReprise != 0 {
		t.Errorf("%d ligne(s) de reprise ecrites sur un match deja couvert par un film — "+
			"le filtre NOT IN ne joue plus", lignesDeReprise)
	}

	// (c) et la generation SERVIE reste celle du film, avec sa source du degat.
	var voie string
	var sourceMesuree int
	if err := db.QueryRow(`SELECT
		ANY_VALUE(read_path),
		COUNT(*) FILTER (WHERE source_tag IS NOT NULL)
		FROM match_kill_events_latest WHERE match_id = ?`, "m-film").Scan(&voie, &sourceMesuree); err != nil {
		t.Fatalf("select generation servie: %v", err)
	}
	if voie == killscope.ReadPathLiveFeed {
		t.Errorf("la vue sert la voie %q sur un match a film — la reprise a supplante le film, "+
			"et la source du degat fatal a disparu de la lecture sans qu un seul nom change", voie)
	}
	if sourceMesuree != 2 {
		t.Errorf("%d mort(s) portent encore la source du degat, attendu 2 — la generation servie "+
			"n est plus celle du film", sourceMesuree)
	}
}

// ─── J4R-5 : CE QUI EST ECARTE SE COMPTE ───────────────────────────────────────────────────

// TestRepriseCompteLesCouplesSansVictime — l ecart ne doit pas etre muet.
//
// Le filtre `victim_gamertag IS NOT NULL AND <> ”` ecarte des lignes SANS RIEN DIRE (constat
// J4R-5). Le cas n existe pas sur les donnees connues, et c est exactement ce qui le rend
// dangereux : le jour ou un titre produit des couples degrades, des morts disparaitraient du
// journal sans qu aucun compteur ne bouge, et le nombre de morts d un match n a pas de valeur
// attendue a laquelle le comparer.
//
// Le test verifie les DEUX moities : ce qui est ecarte est compte, et ce qui est valide passe.
func TestRepriseCompteLesCouplesSansVictime(t *testing.T) {
	db := basePourReprise(t)
	couple(t, db, "m-mixte", "Victime-A", 1000)
	// Deux couples degrades : l un a nom vide, l autre a nom NULL.
	if _, err := db.Exec(`
		INSERT INTO killer_victim_pairs
			(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms)
		VALUES ('m-mixte', '1001', 'Tueur', '1002', '', 1, 2000),
		       ('m-mixte', '1001', 'Tueur', '1003', NULL, 1, 3000)`); err != nil {
		t.Fatalf("insert couples degrades: %v", err)
	}

	if n := compterCouplesSansVictime(db); n != 2 {
		t.Errorf("compterCouplesSansVictime = %d, attendu 2 (un nom vide, un nom NULL) — "+
			"l ecart ne serait pas compte, donc pas visible", n)
	}

	if err := applyKillEventsFromPairs(db); err != nil {
		t.Fatalf("applyKillEventsFromPairs: %v", err)
	}
	if n := compterVue(t, db, "m-mixte"); n != 1 {
		t.Errorf("la vue sert %d mort(s), attendu 1 — seul le couple nomme doit entrer, et il "+
			"doit entrer", n)
	}

	// Le comptage porte le MEME perimetre que l INSERT : un match deja couvert n est pas repris,
	// ses couples degrades ne sont donc pas une perte a annoncer.
	passeDeFilm(t, db, "m-couvert", 1)
	if _, err := db.Exec(`
		INSERT INTO killer_victim_pairs
			(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms)
		VALUES ('m-couvert', '1001', 'Tueur', '1002', '', 1, 4000)`); err != nil {
		t.Fatalf("insert couple degrade sur match couvert: %v", err)
	}
	if n := compterCouplesSansVictime(db); n != 0 {
		t.Errorf("compterCouplesSansVictime = %d apres reprise, attendu 0 — le comptage annonce "+
			"comme perdus des couples de matchs que la reprise ne prend de toute facon pas", n)
	}
}

// TestPasseLaPlusRecenteGagne — l ORDRE de la vue `_latest`, dans le sens qui compte.
//
// La reprise ecrite, un decodage de film qui arrive APRES doit devenir la generation servie :
// c est ce qui permet a un film d enrichir un match deja repris. L ordre
// (`written_at DESC, id DESC`) est ce qui le decide — inverse, la reprise resterait servie pour
// toujours et aucun decodage ulterieur ne remonterait jamais a la lecture.
func TestPasseLaPlusRecenteGagne(t *testing.T) {
	db := basePourReprise(t)
	couple(t, db, "m-puis-film", "Victime-A", 1000)
	couple(t, db, "m-puis-film", "Victime-B", 2000)
	couple(t, db, "m-puis-film", "Victime-C", 3000)

	if err := applyKillEventsFromPairs(db); err != nil {
		t.Fatalf("applyKillEventsFromPairs: %v", err)
	}
	if n := compterVue(t, db, "m-puis-film"); n != 3 {
		t.Fatalf("la reprise sert %d morts, attendu 3", n)
	}

	// Le film arrive ensuite, avec MOINS de lignes : c est le cas reel (la passe de film est plus
	// riche par ligne et plus pauvre en couverture). Elle doit malgre tout primer.
	passeDeFilm(t, db, "m-puis-film", 2)

	if n := compterVue(t, db, "m-puis-film"); n != 2 {
		t.Errorf("la vue sert %d morts apres le decodage du film, attendu 2 — soit les deux "+
			"generations se melangent, soit la reprise reste servie et le film n atteint jamais "+
			"la lecture", n)
	}
	var voie string
	if err := db.QueryRow(
		`SELECT ANY_VALUE(read_path) FROM match_kill_events_latest WHERE match_id = ?`,
		"m-puis-film").Scan(&voie); err != nil {
		t.Fatalf("select voie: %v", err)
	}
	if voie != "marche" {
		t.Errorf("voie servie = %q, attendu celle du film — la passe la plus recente ne gagne "+
			"plus, donc aucun decodage ulterieur ne remonte a la lecture", voie)
	}
}
