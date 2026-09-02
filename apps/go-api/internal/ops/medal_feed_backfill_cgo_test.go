// Package ops — medal_feed_backfill_cgo_test.go : la passe de rattrapage des
// medailles, sur DuckDB :memory: et sur de VRAIS octets de chunk highlight.
//
// LE FILM EST SYNTHETIQUE, LE PARSEUR EST LE VRAI. Le flux est reconstruit ici au
// format documente en tete d analysis/highlight_event_parser.go (xuid 64 bits LE +
// 0x2d + 0xc0, puis 60 octets d event et le marqueur de fin) : la passe traverse
// donc le decodeur reel. Seule la SOURCE du chunk est une doublure — c est la
// couture prevue pour ne pas dependre d un film sur disque.
package ops

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
)

// ─── Doublures et fabriques ───────────────────────────────────────────────────

// filmSynthetique rend un chunk par match ; un match absent de la carte n a pas
// de film.
type filmSynthetique struct {
	parMatch map[string][]byte
	erreur   error
}

func (f filmSynthetique) ChunkHighlight(_ context.Context, matchID string) ([]byte, bool, error) {
	if f.erreur != nil {
		return nil, false, f.erreur
	}
	data, ok := f.parMatch[matchID]
	if !ok {
		return nil, false, nil
	}
	return data, true, nil
}

// evenementFilm decrit un event a fabriquer dans le chunk synthetique.
type evenementFilm struct {
	xuid      uint64
	typeHint  int
	timeMS    int
	isMedal   bool
	medalType int
}

// octetsEvent fabrique les 60 octets d un bloc event (layout version >= 41 :
// gamertag[0:32], type_hint[47], time_ms[48:52] big-endian, is_medal[55],
// medal_type[59]).
func octetsEvent(e evenementFilm) []byte {
	b := make([]byte, 60)
	binary.LittleEndian.PutUint16(b[0:], uint16('J'))
	b[47] = byte(e.typeHint)
	binary.BigEndian.PutUint32(b[48:52], uint32(e.timeMS))
	if e.isMedal {
		b[55] = 1
	}
	b[59] = byte(e.medalType)
	return b
}

// chunkSynthetique concatene un bloc par event. Le scanner traverse tout le flux :
// concatener des blocs autonomes suffit a produire un chunk multi-events.
func chunkSynthetique(events ...evenementFilm) []byte {
	var buf bytes.Buffer
	for _, e := range events {
		buf.Write(make([]byte, 20)) // rembourrage avant le marqueur de xuid
		xuidBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(xuidBytes, e.xuid)
		buf.Write(xuidBytes)
		buf.WriteByte(0x2d)
		buf.WriteByte(0xc0)
		buf.Write(make([]byte, 30))
		buf.Write(octetsEvent(e))
		buf.Write([]byte{0x00, 0x00, 0x2e, 0xe0}) // marqueur de fin
		buf.Write(make([]byte, 10))
	}
	return buf.Bytes()
}

// tableMedailles est la table mesuree, reduite a ce que les tests utilisent.
func tableMedailles(typeHint, medalType int) (string, bool) {
	switch {
	case typeHint == 50 && medalType == 26:
		return "Killjoy", true
	case typeHint == 100 && medalType == 109:
		return "Perfect", true
	case typeHint == 150 && medalType == 1:
		return "Triple Kill", true
	}
	return "", false
}

// ouvrirSharedMedailles monte une base avec le seul schema dont la passe depend
// (DDL calque sur migration.ApplyHighlightEventsAutoincrement).
func ouvrirSharedMedailles(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb :memory:: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, q := range []string{
		`CREATE SEQUENCE highlight_events_id_seq`,
		`CREATE TABLE highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR
		)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("DDL (%s): %v", q, err)
		}
	}
	return db
}

// insererEvent pose une ligne sans identite (l etat des 415 matchs mesures).
func insererEvent(t *testing.T, db *sql.DB, matchID, eventType, xuid string, timeMS int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, ?, ?, ?)`,
		matchID, eventType, timeMS, xuid); err != nil {
		t.Fatalf("insert %s/%s: %v", matchID, eventType, err)
	}
}

// lireIdentites relit (type_hint, raw_json) des events d un match, par id croissant.
func lireIdentites(t *testing.T, db *sql.DB, matchID string) []struct {
	TypeHint sql.NullInt64
	RawJSON  sql.NullString
} {
	t.Helper()
	rows, err := db.Query(
		`SELECT type_hint, raw_json FROM highlight_events WHERE match_id = ? ORDER BY id`, matchID)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []struct {
		TypeHint sql.NullInt64
		RawJSON  sql.NullString
	}
	for rows.Next() {
		var l struct {
			TypeHint sql.NullInt64
			RawJSON  sql.NullString
		}
		if err := rows.Scan(&l.TypeHint, &l.RawJSON); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, l)
	}
	return out
}

const xuidA = uint64(2533274792574872)
const xuidB = uint64(2535467890123456)

// ─── La passe de bout en bout ─────────────────────────────────────────────────

// TestBackfillMedailles_RendLIdentite — le cas nominal : deux medailles en base
// sans identite, le film les nomme.
func TestBackfillMedailles_RendLIdentite(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	const matchID = "m-nominal"
	insererEvent(t, db, matchID, analysis.EventTypeMedal, "2533274792574872", 1000)
	insererEvent(t, db, matchID, analysis.EventTypeMedal, "2535467890123456", 2000)
	insererEvent(t, db, matchID, analysis.EventTypeKill, "2533274792574872", 1000)

	films := filmSynthetique{parMatch: map[string][]byte{matchID: chunkSynthetique(
		evenementFilm{xuid: xuidA, typeHint: 50, timeMS: 1000, isMedal: true, medalType: 26},
		evenementFilm{xuid: xuidB, typeHint: 100, timeMS: 2000, isMedal: true, medalType: 109},
	)}}

	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v", err)
	}
	if bilan.MatchsCandidats != 1 || bilan.MatchsTraites != 1 {
		t.Errorf("matchs: candidats=%d traites=%d, attendus 1/1", bilan.MatchsCandidats, bilan.MatchsTraites)
	}
	if bilan.EventsCandidats != 2 || bilan.EventsIdentifies != 2 {
		t.Errorf("events: candidats=%d identifies=%d, attendus 2/2",
			bilan.EventsCandidats, bilan.EventsIdentifies)
	}
	lignes := lireIdentites(t, db, matchID)
	if len(lignes) != 3 {
		t.Fatalf("%d lignes en base, 3 attendues", len(lignes))
	}
	if !lignes[0].RawJSON.Valid || lignes[0].RawJSON.String != `{"medal_name":"Killjoy"}` {
		t.Errorf("medaille 1: raw_json = %v, attendu Killjoy", lignes[0].RawJSON)
	}
	if !lignes[0].TypeHint.Valid || lignes[0].TypeHint.Int64 != 50 {
		t.Errorf("medaille 1: type_hint = %v, attendu 50", lignes[0].TypeHint)
	}
	if !lignes[1].RawJSON.Valid || lignes[1].RawJSON.String != `{"medal_name":"Perfect"}` {
		t.Errorf("medaille 2: raw_json = %v, attendu Perfect", lignes[1].RawJSON)
	}
	if lignes[2].RawJSON.Valid || lignes[2].TypeHint.Valid {
		t.Errorf("le kill a ete touche (type_hint=%v raw_json=%v) — la passe ne vise QUE les medal",
			lignes[2].TypeHint, lignes[2].RawJSON)
	}
}

// TestBackfillMedailles_FilmAbsentEstConsigneEtSaute — un match sans film en cache
// n est ni une erreur ni un silence : il est compte.
func TestBackfillMedailles_FilmAbsentEstConsigneEtSaute(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	const matchID = "m-sans-film"
	insererEvent(t, db, matchID, analysis.EventTypeMedal, "2533274792574872", 1000)

	bilan, err := BackfillIdentiteMedailles(context.Background(), db,
		filmSynthetique{parMatch: map[string][]byte{}}, tableMedailles, OptionsBackfillMedailles{})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v", err)
	}
	if bilan.MatchsSansFilm != 1 || bilan.MatchsTraites != 0 {
		t.Errorf("bilan: sans_film=%d traites=%d, attendus 1/0", bilan.MatchsSansFilm, bilan.MatchsTraites)
	}
	if lignes := lireIdentites(t, db, matchID); lignes[0].RawJSON.Valid || lignes[0].TypeHint.Valid {
		t.Errorf("la base a ete touchee alors qu aucun film n existait: %+v", lignes[0])
	}
}

// TestBackfillMedailles_DryRunNEcritRien — le bilan est calcule, la base intacte.
func TestBackfillMedailles_DryRunNEcritRien(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	const matchID = "m-dry"
	insererEvent(t, db, matchID, analysis.EventTypeMedal, "2533274792574872", 1000)
	films := filmSynthetique{parMatch: map[string][]byte{matchID: chunkSynthetique(
		evenementFilm{xuid: xuidA, typeHint: 50, timeMS: 1000, isMedal: true, medalType: 26},
	)}}

	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{DryRun: true})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v", err)
	}
	if bilan.EventsIdentifies != 1 {
		t.Errorf("events identifies = %d, attendu 1 (le dry-run compte)", bilan.EventsIdentifies)
	}
	if lignes := lireIdentites(t, db, matchID); lignes[0].RawJSON.Valid {
		t.Errorf("dry-run: raw_json ecrit (%q)", lignes[0].RawJSON.String)
	}
}

// TestBackfillMedailles_CoupleInconnuGardeLeTypeHintSansNom — DEGRADATION : pas de
// nom voisin. Le type_hint, lui, est une quantite mesuree et part quand meme.
func TestBackfillMedailles_CoupleInconnuGardeLeTypeHintSansNom(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	const matchID = "m-inconnu"
	insererEvent(t, db, matchID, analysis.EventTypeMedal, "2533274792574872", 1000)
	films := filmSynthetique{parMatch: map[string][]byte{matchID: chunkSynthetique(
		evenementFilm{xuid: xuidA, typeHint: 200, timeMS: 1000, isMedal: true, medalType: 44},
	)}}

	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v", err)
	}
	if bilan.EventsSansNom != 1 || bilan.EventsIdentifies != 0 {
		t.Errorf("bilan: sans_nom=%d identifies=%d, attendus 1/0",
			bilan.EventsSansNom, bilan.EventsIdentifies)
	}
	lignes := lireIdentites(t, db, matchID)
	if lignes[0].RawJSON.Valid {
		t.Errorf("raw_json = %q — un couple inconnu ne doit recevoir AUCUN nom", lignes[0].RawJSON.String)
	}
	if !lignes[0].TypeHint.Valid || lignes[0].TypeHint.Int64 != 200 {
		t.Errorf("type_hint = %v, attendu 200", lignes[0].TypeHint)
	}
}

// TestBackfillMedailles_PlafondBorneLesMatchsTraites — le plafond borne les matchs
// TRAITES, pas les events : un match commence est un match fini.
func TestBackfillMedailles_PlafondBorneLesMatchsTraites(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	insererEvent(t, db, "m-a", analysis.EventTypeMedal, "2533274792574872", 1000)
	insererEvent(t, db, "m-b", analysis.EventTypeMedal, "2533274792574872", 1000)
	chunk := chunkSynthetique(
		evenementFilm{xuid: xuidA, typeHint: 50, timeMS: 1000, isMedal: true, medalType: 26})
	films := filmSynthetique{parMatch: map[string][]byte{"m-a": chunk, "m-b": chunk}}

	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{Plafond: 1})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v", err)
	}
	// Le lot COMPLET est designe (2 matchs) ; seul le premier est traite.
	if bilan.MatchsCandidats != 2 || bilan.MatchsTraites != 1 {
		t.Errorf("bilan: candidats=%d traites=%d, attendus 2/1 (plafond)",
			bilan.MatchsCandidats, bilan.MatchsTraites)
	}
	if lignes := lireIdentites(t, db, "m-b"); lignes[0].RawJSON.Valid {
		t.Errorf("m-b a ete traite malgre le plafond")
	}
}

// TestBackfillMedailles_PlafondNEstPasEpuiseParLesSansFilm — LE BUG CORRIGE : le
// lot etait borne EN SQL, donc les N premiers identifiants de l ordre alphabetique
// etaient pris meme sans film — et une passe a `--limit 1` sur une tete de lot sans
// film ne faisait RIEN, indefiniment. Le plafond ne compte que ce qui est traite.
func TestBackfillMedailles_PlafondNEstPasEpuiseParLesSansFilm(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	// m-a et m-b n ont pas de film ; seul m-c en a un. En ordre alphabetique, les
	// deux sans-film sont EN TETE.
	insererEvent(t, db, "m-a", analysis.EventTypeMedal, "2533274792574872", 1000)
	insererEvent(t, db, "m-b", analysis.EventTypeMedal, "2533274792574872", 1000)
	insererEvent(t, db, "m-c", analysis.EventTypeMedal, "2533274792574872", 1000)
	films := filmSynthetique{parMatch: map[string][]byte{"m-c": chunkSynthetique(
		evenementFilm{xuid: xuidA, typeHint: 50, timeMS: 1000, isMedal: true, medalType: 26},
	)}}

	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{Plafond: 1})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v", err)
	}
	if bilan.MatchsSansFilm != 2 || bilan.MatchsTraites != 1 {
		t.Errorf("bilan: sans_film=%d traites=%d, attendus 2/1", bilan.MatchsSansFilm, bilan.MatchsTraites)
	}
	lignes := lireIdentites(t, db, "m-c")
	if !lignes[0].RawJSON.Valid || lignes[0].RawJSON.String != `{"medal_name":"Killjoy"}` {
		t.Errorf("m-c non rattrape (raw_json = %v) — les sans-film ont epuise le plafond", lignes[0].RawJSON)
	}
}

// TestBackfillMedailles_DeuxiemePasseNeRefaitPasLeTravail — reprenabilite : un
// match entierement resolu sort du lot.
func TestBackfillMedailles_DeuxiemePasseNeRefaitPasLeTravail(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	const matchID = "m-reprise"
	insererEvent(t, db, matchID, analysis.EventTypeMedal, "2533274792574872", 1000)
	films := filmSynthetique{parMatch: map[string][]byte{matchID: chunkSynthetique(
		evenementFilm{xuid: xuidA, typeHint: 50, timeMS: 1000, isMedal: true, medalType: 26},
	)}}
	ctx := context.Background()
	if _, err := BackfillIdentiteMedailles(ctx, db, films, tableMedailles,
		OptionsBackfillMedailles{}); err != nil {
		t.Fatalf("1re passe: %v", err)
	}
	bilan, err := BackfillIdentiteMedailles(ctx, db, films, tableMedailles, OptionsBackfillMedailles{})
	if err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	if bilan.MatchsCandidats != 0 {
		t.Errorf("2e passe: %d matchs candidats, attendu 0", bilan.MatchsCandidats)
	}
}

// TestBackfillMedailles_MatchPartiellementIdentifie — un match dont UNE medaille
// porte deja son nom : l appariement compte les deux cotes en entier (sinon le
// cardinal ne tomberait jamais juste) et ne rattrape que la ligne qui manque.
func TestBackfillMedailles_MatchPartiellementIdentifie(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	const matchID = "m-partiel"
	// Deux medailles au MEME instant pour le MEME joueur : elles tombent dans le
	// meme groupe d appariement. La premiere est deja nommee.
	if _, err := db.Exec(
		`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, raw_json)
		 VALUES (?, ?, ?, ?, ?)`,
		matchID, analysis.EventTypeMedal, 5000, "2533274792574872", `{"medal_name":"Killjoy"}`); err != nil {
		t.Fatalf("insert identifiee: %v", err)
	}
	insererEvent(t, db, matchID, analysis.EventTypeMedal, "2533274792574872", 5000)

	films := filmSynthetique{parMatch: map[string][]byte{matchID: chunkSynthetique(
		evenementFilm{xuid: xuidA, typeHint: 50, timeMS: 5000, isMedal: true, medalType: 26},
		evenementFilm{xuid: xuidA, typeHint: 100, timeMS: 5000, isMedal: true, medalType: 109},
	)}}

	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v", err)
	}
	if bilan.EventsCandidats != 1 || bilan.EventsIdentifies != 1 || bilan.EventsSansPaire != 0 {
		t.Errorf("bilan: candidats=%d identifies=%d sans_paire=%d, attendus 1/1/0",
			bilan.EventsCandidats, bilan.EventsIdentifies, bilan.EventsSansPaire)
	}
	lignes := lireIdentites(t, db, matchID)
	if lignes[0].RawJSON.String != `{"medal_name":"Killjoy"}` {
		t.Errorf("ligne deja nommee alteree: %v", lignes[0].RawJSON)
	}
	if !lignes[1].RawJSON.Valid || lignes[1].RawJSON.String != `{"medal_name":"Perfect"}` {
		t.Errorf("ligne rattrapee: raw_json = %v, attendu Perfect", lignes[1].RawJSON)
	}
}

// TestBackfillMedailles_ChunkIndecodableEstConsigneEtSaute — le film est LA mais
// ses octets ne se decodent pas : le match est compte « illisible », la base reste
// intacte, et la passe CONTINUE sur les suivants.
func TestBackfillMedailles_ChunkIndecodableEstConsigneEtSaute(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	insererEvent(t, db, "m-casse", analysis.EventTypeMedal, "2533274792574872", 1000)
	insererEvent(t, db, "m-sain", analysis.EventTypeMedal, "2533274792574872", 1000)

	// Un flux zlib annonce (0x78 0x9c) mais tronque : le decodeur echoue au lieu de
	// rendre zero event. C est le cas « chunk corrompu », distinct du « film absent ».
	casse := []byte{0x78, 0x9c, 0x01, 0x02, 0x03, 0x04, 0x05}
	films := filmSynthetique{parMatch: map[string][]byte{
		"m-casse": casse,
		"m-sain": chunkSynthetique(
			evenementFilm{xuid: xuidA, typeHint: 50, timeMS: 1000, isMedal: true, medalType: 26}),
	}}

	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{})
	if err != nil {
		t.Fatalf("BackfillIdentiteMedailles: %v — un chunk illisible ne doit PAS arreter la passe", err)
	}
	if bilan.MatchsIllisibles != 1 {
		t.Errorf("matchs illisibles = %d, attendu 1", bilan.MatchsIllisibles)
	}
	if lignes := lireIdentites(t, db, "m-casse"); lignes[0].RawJSON.Valid || lignes[0].TypeHint.Valid {
		t.Errorf("la base a ete touchee sur un chunk illisible: %+v", lignes[0])
	}
	// LA PASSE A CONTINUE : le match sain qui suit est bien rattrape.
	if bilan.MatchsTraites != 1 {
		t.Errorf("matchs traites = %d, attendu 1 (la passe doit continuer)", bilan.MatchsTraites)
	}
	if lignes := lireIdentites(t, db, "m-sain"); !lignes[0].RawJSON.Valid {
		t.Errorf("m-sain non rattrape — la passe s est arretee sur le chunk illisible")
	}
}

// TestBackfillMedailles_ErreurDeSourceInterromptLaPasse — une ERREUR de lecture du
// cache n est PAS un « film absent » : elle signale un environnement casse (disque,
// permissions). La passe s arrete et rend son bilan partiel plutot que de parcourir
// 415 matchs en echouant sur chacun.
func TestBackfillMedailles_ErreurDeSourceInterromptLaPasse(t *testing.T) {
	db := ouvrirSharedMedailles(t)
	insererEvent(t, db, "m-a", analysis.EventTypeMedal, "2533274792574872", 1000)
	insererEvent(t, db, "m-b", analysis.EventTypeMedal, "2533274792574872", 1000)

	films := filmSynthetique{erreur: errors.New("disque illisible")}
	bilan, err := BackfillIdentiteMedailles(context.Background(), db, films, tableMedailles,
		OptionsBackfillMedailles{})
	if err == nil {
		t.Fatal("erreur attendue — une source en echec doit interrompre la passe")
	}
	if !strings.Contains(err.Error(), "disque illisible") {
		t.Errorf("erreur = %v, la cause racine doit remonter", err)
	}
	// Bilan PARTIEL : le lot a bien ete designe, rien n a ete traite.
	if bilan.MatchsCandidats != 2 || bilan.MatchsTraites != 0 {
		t.Errorf("bilan partiel: candidats=%d traites=%d, attendus 2/0",
			bilan.MatchsCandidats, bilan.MatchsTraites)
	}
	if lignes := lireIdentites(t, db, "m-a"); lignes[0].RawJSON.Valid {
		t.Errorf("la base a ete touchee malgre l erreur de source")
	}
}

// ─── L appariement, isole ─────────────────────────────────────────────────────

// TestApparier_OrdreDansLeGroupe — deux medailles au meme instant pour le meme
// joueur : le n-ieme event du film va sur la n-ieme ligne du groupe.
func TestApparier_OrdreDansLeGroupe(t *testing.T) {
	enBase := []evenementBase{
		{id: 10, xuid: "2533274792574872", timeMS: 5000},
		{id: 11, xuid: "2533274792574872", timeMS: 5000},
	}
	events := []analysis.HighlightEvent{
		{XUID: xuidA, EventType: analysis.EventTypeMedal, TypeHint: 50, TimeMS: 5000, MedalType: 26},
		{XUID: xuidA, EventType: analysis.EventTypeMedal, TypeHint: 100, TimeMS: 5000, MedalType: 109},
	}
	var bilan BilanBackfillMedailles
	corrections := apparier(context.Background(), enBase, events, tableMedailles, &bilan)
	if len(corrections) != 2 {
		t.Fatalf("%d corrections, 2 attendues", len(corrections))
	}
	if corrections[0].id != 10 || corrections[0].typeHint != 50 {
		t.Errorf("correction 0 = %+v, attendu id=10 type_hint=50", corrections[0])
	}
	if corrections[1].id != 11 || corrections[1].typeHint != 100 {
		t.Errorf("correction 1 = %+v, attendu id=11 type_hint=100", corrections[1])
	}
	if bilan.EventsIdentifies != 2 || bilan.EventsSansPaire != 0 {
		t.Errorf("bilan = %+v, attendu 2 identifies / 0 sans paire", bilan)
	}
}

// TestApparier_GroupeDesaccordeNEstPasDevine — cardinaux differents des deux
// cotes : on ne devine PAS, on compte.
func TestApparier_GroupeDesaccordeNEstPasDevine(t *testing.T) {
	enBase := []evenementBase{
		{id: 10, xuid: "2533274792574872", timeMS: 5000},
		{id: 11, xuid: "2533274792574872", timeMS: 5000},
	}
	events := []analysis.HighlightEvent{
		{XUID: xuidA, EventType: analysis.EventTypeMedal, TypeHint: 50, TimeMS: 5000, MedalType: 26},
	}
	var bilan BilanBackfillMedailles
	if corrections := apparier(context.Background(), enBase, events, tableMedailles, &bilan); len(corrections) != 0 {
		t.Fatalf("%d corrections, 0 attendue sur un groupe desaccorde", len(corrections))
	}
	if bilan.EventsSansPaire != 2 {
		t.Errorf("events sans paire = %d, attendu 2", bilan.EventsSansPaire)
	}
}

// TestApparier_IgnoreLesEventsNonMedal — le film porte kills et morts ; ils
// n entrent jamais dans l appariement.
func TestApparier_IgnoreLesEventsNonMedal(t *testing.T) {
	enBase := []evenementBase{{id: 10, xuid: "2533274792574872", timeMS: 5000}}
	events := []analysis.HighlightEvent{
		{XUID: xuidA, EventType: analysis.EventTypeKill, TypeHint: 50, TimeMS: 5000},
		{XUID: xuidA, EventType: analysis.EventTypeMedal, TypeHint: 50, TimeMS: 5000, MedalType: 26},
	}
	var bilan BilanBackfillMedailles
	corrections := apparier(context.Background(), enBase, events, tableMedailles, &bilan)
	if len(corrections) != 1 || corrections[0].rawJSON == nil ||
		*corrections[0].rawJSON != `{"medal_name":"Killjoy"}` {
		t.Fatalf("corrections = %+v, attendu 1 correction Killjoy", corrections)
	}
}
