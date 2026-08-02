//go:build integration

// Package killcollector — credit_test.go : LES TROIS EXIGENCES DU PRODUCTEUR CREDIT-SEUL.
//
// Elles sont testees SUR BASE (DuckDB en memoire de fichier + migrations reelles + la vue
// `_latest`) parce que les deux premieres ne se voient QU EN BASE :
//
//	1. LA PRESEANCE  — un match porteur des DEUX sources ne doit produire QU UN jeu de lignes.
//	                   Sans ce test, on reintroduit exactement le doublon que le chantier
//	                   elimine (46,5 % de `killer_victim_pairs`).
//	2. LES TROIS ETATS DE L ASSISTANT — `assist_known = FALSE` (« on ne sait pas »), jamais un
//	                   NULL qui se lirait « pas d assistant ». Idem source et parts : ABSENTES,
//	                   pas nulles-comme-zero.
//	3. LA COUVERTURE — le producteur doit rendre des lignes la ou aucun film n existe. Le
//	                   chiffre de production (1 343 matchs et non 949) se mesure au backfill ;
//	                   ce qui se teste ici est le MECANISME : pas de film, des lignes quand meme.

package killcollector

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/persist"
)

// ─── Harnais ───────────────────────────────────────────────────────────────────────────────

// insererEvenements peuple `highlight_events` et `match_participants` d un match fabrique.
func insererEvenements(t *testing.T, db *sql.DB, matchID string, evs []evenement) {
	t.Helper()
	vus := map[string]bool{}
	for _, e := range evs {
		if !vus[e.xuid] {
			vus[e.xuid] = true
			if _, err := db.Exec(
				`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES (?, ?, ?)`,
				matchID, e.xuid, e.gamertag); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
		}
		if _, err := db.Exec(
			`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, ?, ?, ?)`,
			matchID, e.typ, e.timeMS, e.xuid); err != nil {
			t.Fatalf("insert highlight_event: %v", err)
		}
	}
}

type evenement struct {
	xuid     string
	gamertag string
	typ      string
	timeMS   int
}

// duelSimple : deux joueurs, trois morts — la forme minimale qui produit des couples.
func duelSimple() []evenement {
	return []evenement{
		{xuid: "1001", gamertag: "Alpha", typ: "kill", timeMS: 1000},
		{xuid: "1002", gamertag: "Bravo", typ: "death", timeMS: 1002},
		{xuid: "1002", gamertag: "Bravo", typ: "kill", timeMS: 5000},
		{xuid: "1001", gamertag: "Alpha", typ: "death", timeMS: 5001},
		{xuid: "1001", gamertag: "Alpha", typ: "kill", timeMS: 9000},
		{xuid: "1002", gamertag: "Bravo", typ: "death", timeMS: 9003},
	}
}

func compterVue(t *testing.T, db *sql.DB, matchID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?`, matchID).Scan(&n); err != nil {
		t.Fatalf("count vue: %v", err)
	}
	return n
}

// ─── LE ROSTER : DEUX FORMES DE NOM, ET AUCUNE NE SE DEVINE ────────────────────────────────

// TestRosterResoutLesDeuxFormesDeNom — LE test du defaut le plus couteux de la session 2.
//
// Il porte sur deux choses que rien ne signalait :
//
//  1. `match_participants.gamertag` EST VIDE en production (4 xuids nommes sur 16 996). Un
//     roster qui lit cette colonne rend une table vide, donc des morts SANS xuid — 16 908
//     ecrites, 10 avec un xuid de victime. Le nom canonique vit dans `v_gamertag_lookup`.
//  2. Le film ne donne pas toujours un gamertag : quand il n en a pas, le decodeur ecrit
//     `xuid:<decimal>`. Cette forme EST l identite ; la chercher dans une table de gamertags
//     ne rend evidemment rien.
//
// Le test fabrique les deux cas et exige un xuid dans les deux.
func TestRosterResoutLesDeuxFormesDeNom(t *testing.T) {
	const match = "match-identites"
	db := openSharedTestDB(t)

	// Deux participants SANS gamertag dans `match_participants` — le cas de production.
	for _, xuid := range []string{"1001", "1002"} {
		if _, err := db.Exec(
			`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES (?, ?, NULL)`,
			match, xuid); err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}
	// Un seul des deux est nomme par la source canonique.
	if _, err := db.Exec(
		`INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "1001", "Alpha"); err != nil {
		t.Fatalf("insert alias: %v", err)
	}

	ids, err := NewSharedRoster(db).IdentitiesForMatch(context.Background(), match)
	if err != nil {
		t.Fatalf("IdentitiesForMatch: %v", err)
	}

	// (1) le nom canonique se resout, alors que `match_participants.gamertag` est NULL.
	if xuid, nom := ids.Resoudre("Alpha"); xuid != "1001" || nom != "Alpha" {
		t.Errorf("Resoudre(\"Alpha\") = (%q, %q), attendu (\"1001\", \"Alpha\") — le roster doit "+
			"lire v_gamertag_lookup, pas match_participants.gamertag (vide en production)", xuid, nom)
	}
	// (2) la forme `xuid:` EST l identite, et elle ne depend d aucune table.
	if xuid, _ := ids.Resoudre("xuid:1002"); xuid != "1002" {
		t.Errorf("Resoudre(\"xuid:1002\") = %q, attendu \"1002\" — cette forme est le xuid "+
			"lui-meme, pas un pseudo a chercher dans le roster", xuid)
	}
	// (3) un nom inconnu reste sans xuid : on n invente pas.
	if xuid, nom := ids.Resoudre("Inconnu"); xuid != "" || nom != "Inconnu" {
		t.Errorf("Resoudre(\"Inconnu\") = (%q, %q), attendu (\"\", \"Inconnu\")", xuid, nom)
	}
	// (4) une forme `xuid:` non decimale n est pas un xuid.
	if xuid, _ := ids.Resoudre("xuid:pas-un-nombre"); xuid != "" {
		t.Errorf("Resoudre(\"xuid:pas-un-nombre\") = %q, attendu vide", xuid)
	}
}

// ─── EXIGENCE 1 : LA PRESEANCE ─────────────────────────────────────────────────────────────

// TestPreseanceFilmSurCredit — le test qui empeche de reintroduire le doublon.
//
// Le match porte LES DEUX sources. Quel que soit l ordre d arrivee, la vue doit servir UN SEUL
// jeu de lignes, et ce doit etre celui du FILM : lui seul porte la source du degat, et une
// passe credit-seul qui le supplanterait l EFFACERAIT de la lecture.
func TestPreseanceFilmSurCredit(t *testing.T) {
	const match = "match-deux-sources"
	db := openSharedTestDB(t)
	insererEvenements(t, db, match, duelSimple())

	credit := NewCreditCollector(db, sharedWriter(db))
	ctx := context.Background()

	// (a) le credit d abord, le film ensuite : le film est plus recent, il gagne.
	if _, _, err := credit.CollectMatch(ctx, match); err != nil {
		t.Fatalf("credit: %v", err)
	}
	apresCredit := compterVue(t, db, match)
	if apresCredit == 0 {
		t.Fatal("le producteur credit n a rien ecrit — le reste du test ne prouverait rien")
	}
	ecrirePasseDeFilm(t, db, match, 2)

	if n := compterVue(t, db, match); n != 2 {
		t.Fatalf("la vue sert %d lignes apres la passe de film, attendu 2 — "+
			"les deux passes se melangent, ce qui est exactement le doublon qu on elimine", n)
	}
	if voie := voieDeLaVue(t, db, match); voie == killscope.ReadPathCreditBackfill {
		t.Fatalf("la vue sert la voie %q — le film doit primer, sinon la source du degat "+
			"disparait de la lecture", voie)
	}

	// (b) le credit REPASSE alors que le film couvre deja le match : il doit REFUSER.
	outcome, n, err := credit.CollectMatch(ctx, match)
	if err != nil {
		t.Fatalf("credit (second passage): %v", err)
	}
	if outcome != CreditSkippedFilm || n != 0 {
		t.Fatalf("outcome = %q (%d lignes), attendu %q et 0 — la preseance n a pas joue",
			outcome, n, CreditSkippedFilm)
	}
	if n := compterVue(t, db, match); n != 2 {
		t.Fatalf("la vue sert %d lignes apres le second passage du credit, attendu 2", n)
	}
	if voie := voieDeLaVue(t, db, match); voie == killscope.ReadPathCreditBackfill {
		t.Fatal("le second passage du credit a supplante le film — la source du degat est perdue")
	}
}

// ecrirePasseDeFilm : une passe de FILM fabriquee (aucun film n est necessaire pour tester la
// preseance — ce qui compte est la voie de lecture, et c est elle qui la decide).
func ecrirePasseDeFilm(t *testing.T, db *sql.DB, matchID string, morts int) {
	t.Helper()
	batch := persist.KillSourceBatch{
		MatchID: matchID, DecoderRev: KillSourceDecoderRev, Publishable: true,
	}
	for i := 0; i < morts; i++ {
		batch.Deaths = append(batch.Deaths, persist.KillEventInsert{
			TimeMS: 1000 * (i + 1), VictimGamertag: "Bravo", VictimXUID: "1002",
			FeedKillerGamertag: "Alpha", FeedKillerXUID: "1001", FeedPresent: true,
			AssistKnown: true,
			SourceTag:   0xacd1cff4, SourceCategory: "None",
			ReadPath: "marche", ReadOrigin: "credit-concordant",
		})
	}
	if err := persist.NewKillSourcePersister(db).PersistPass(context.Background(), batch); err != nil {
		t.Fatalf("passe de film: %v", err)
	}
}

func voieDeLaVue(t *testing.T, db *sql.DB, matchID string) string {
	t.Helper()
	var voie string
	if err := db.QueryRow(
		`SELECT DISTINCT read_path FROM match_kill_events_latest WHERE match_id = ?`,
		matchID).Scan(&voie); err != nil {
		t.Fatalf("voie de la vue: %v", err)
	}
	return voie
}

// ─── EXIGENCE 2 : LES TROIS ETATS ──────────────────────────────────────────────────────────

// TestCreditEcritLesTroisEtatsSansMentir : ce que le credit-seul a le droit d affirmer.
//
// `assist_known = FALSE` dit ON NE SAIT PAS. La source, sa categorie, la divergence et les deux
// parts de degats sont NULL — ABSENTES, jamais 0 ni FALSE, parce qu une absence de mesure n est
// pas une mesure d absence.
func TestCreditEcritLesTroisEtatsSansMentir(t *testing.T) {
	const match = "match-credit-seul"
	db := openSharedTestDB(t)
	insererEvenements(t, db, match, duelSimple())

	outcome, morts, err := NewCreditCollector(db, sharedWriter(db)).CollectMatch(context.Background(), match)
	if err != nil {
		t.Fatalf("credit: %v", err)
	}
	if outcome != CreditWritten || morts == 0 {
		t.Fatalf("outcome = %q (%d morts) — le producteur n a rien ecrit", outcome, morts)
	}

	var assistConnus int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest
		WHERE match_id = ? AND assist_known`, match).Scan(&assistConnus); err != nil {
		t.Fatalf("select assist_known: %v", err)
	}
	if assistConnus != 0 {
		t.Errorf("%d ligne(s) avec assist_known = TRUE — le kill-feed de l API ne porte aucun "+
			"assistant, l affirmer fabriquerait un fait jamais observe", assistConnus)
	}

	// Les colonnes qui doivent etre ABSENTES. Un `0` ou un `FALSE` ici serait une mesure
	// fabriquee a partir d une absence de mesure.
	for _, col := range []string{
		"source_tag", "source_category", "diverges", "killer_damage_pct", "assist_damage_pct",
		"assist_gamertag", "assist_xuid", "assist_index",
	} {
		var renseignees int
		q := `SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ? AND ` + col + ` IS NOT NULL`
		if err := db.QueryRow(q, match).Scan(&renseignees); err != nil {
			t.Fatalf("select %s: %v", col, err)
		}
		if renseignees != 0 {
			t.Errorf("%s renseignee sur %d ligne(s) credit-seul — cette colonne se lit dans le "+
				"film, et il n y en a pas", col, renseignees)
		}
	}

	// Et la portee, elle, est OBLIGATOIRE : une mesure porte sur ce qu elle a mesure.
	var horsPortee int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest
		WHERE match_id = ? AND (read_path <> ? OR read_origin <> ?)`,
		match, killscope.ReadPathCreditBackfill, killscope.OriginCreditOnly).Scan(&horsPortee); err != nil {
		t.Fatalf("select portee: %v", err)
	}
	if horsPortee != 0 {
		t.Errorf("%d ligne(s) sans la portee credit-seul — la preseance se decide sur read_path, "+
			"une portee fausse la rendrait aveugle", horsPortee)
	}
}

// ─── EXIGENCE 3 : LA COUVERTURE ────────────────────────────────────────────────────────────

// TestCreditCouvreUnMatchSansFilm : le mecanisme de la couverture.
//
// Le chiffre de production (1 343 porteurs de couples au lieu de 949 films) se mesure au
// backfill. Ce qui se teste ici est ce dont il depend : sur un match SANS aucune passe de film,
// le producteur rend autant de morts que l appariement de reference en trouve — ni plus (il
// n invente pas), ni moins (il ne perd pas le match).
func TestCreditCouvreUnMatchSansFilm(t *testing.T) {
	const match = "match-sans-film"
	db := openSharedTestDB(t)
	insererEvenements(t, db, match, duelSimple())

	_, morts, err := NewCreditCollector(db, sharedWriter(db)).CollectMatch(context.Background(), match)
	if err != nil {
		t.Fatalf("credit: %v", err)
	}
	// duelSimple porte 3 kills apparies a 3 morts dans la fenetre de 5 ms.
	if morts != 3 {
		t.Fatalf("morts = %d, attendu 3 — l appariement doit etre celui qui produit "+
			"killer_victim_pairs aujourd hui (tolerance 5 ms, bijection 1->1)", morts)
	}
	if n := compterVue(t, db, match); n != 3 {
		t.Fatalf("la vue sert %d lignes, attendu 3", n)
	}

	// Un match sans aucun evenement ne produit RIEN, et ce n est pas une erreur.
	outcome, n, err := NewCreditCollector(db, sharedWriter(db)).CollectMatch(context.Background(), "match-vide")
	if err != nil {
		t.Fatalf("credit (match vide): %v", err)
	}
	if outcome != CreditNoEvents || n != 0 {
		t.Fatalf("outcome = %q (%d) sur un match sans evenement, attendu %q et 0",
			outcome, n, CreditNoEvents)
	}
}
