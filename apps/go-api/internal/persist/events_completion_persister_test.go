//go:build integration

// Package persist — events_completion_persister_test.go : tests d'intégration du
// EventsCompletionPersister sur DuckDB en mémoire.
//
// Build tag `integration` (+ CGO pour le driver duckdb). Lancer avec :
//
//	CGO_ENABLED=1 go test -tags integration ./internal/persist/...
//
// Vérifie : écriture atomique events + killer_victim (par-kill, gamertags +
// time_ms), marquage des bits + events_loaded, idempotence kv (DELETE+INSERT),
// INSERT OR IGNORE highlight_events (pas de DELETE — table ART-indexée), et le
// marquage no-film définitif.
package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/migration"
)

// Valeurs des bits passées par le caller (sync.MBitEvents / MBitKillerVictim).
// Dupliquées ici car persist n'importe pas sync ; doivent rester alignées.
const (
	testBitEvents       int64 = 1 << 16 // 65536
	testBitKillerVictim int64 = 1 << 19 // 524288
)

// openCompletionTestDB ouvre une DuckDB en mémoire avec le schéma combat réel :
// highlight_events (PK id seq), killer_victim_pairs (forme par-kill, sans PK),
// match_registry minimal.
func openCompletionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	schema := []string{
		`CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq START 1`,
		`CREATE TABLE highlight_events (
			id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id VARCHAR, event_type VARCHAR, time_ms INTEGER, xuid VARCHAR, type_hint INTEGER
		)`,
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
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			backfill_completed BIGINT DEFAULT 0,
			events_loaded BOOLEAN DEFAULT FALSE,
			events_empty BOOLEAN DEFAULT FALSE
		)`,
	}
	for _, ddl := range schema {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}
	// match_kill_events : seconde destination des couples depuis le 2026-08-02 (double
	// écriture datée). DDL issue de la migration réelle — la recopier ici la ferait diverger.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		t.Fatalf("ensure match_kill_events: %v", err)
	}
	return db
}

func seedCompletionRegistry(t *testing.T, db *sql.DB, matchID string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO match_registry (match_id) VALUES (?)`, matchID); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
}

func countCompletionRows(t *testing.T, db *sql.DB, table, matchID string) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table+" WHERE match_id = ?", matchID).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func completionRegistryState(t *testing.T, db *sql.DB, matchID string) (bits int64, loaded bool) {
	t.Helper()
	if err := db.QueryRowContext(context.Background(),
		`SELECT backfill_completed, events_loaded FROM match_registry WHERE match_id = ?`, matchID).
		Scan(&bits, &loaded); err != nil {
		t.Fatalf("registry state: %v", err)
	}
	return bits, loaded
}

func sampleCompletionInput(matchID string) EventsCompletionInput {
	return EventsCompletionInput{
		MatchID: matchID,
		Events: []HLEventCompletion{
			{XUID: "111", EventType: "kill", TimeMS: 1000, TypeHint: 50},
			{XUID: "222", EventType: "death", TimeMS: 1002, TypeHint: 1},
		},
		Pairs: []KVPairCompletion{
			{KillerXUID: "111", KillerGamertag: "Killer1", VictimXUID: "222", VictimGamertag: "Victim1", TimeMS: 1000},
		},
		MarkKV:          true,
		EventsBit:       testBitEvents,
		KillerVictimBit: testBitKillerVictim,
	}
}

func TestEventsCompletionPersister_Persist(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-complete-001"
	seedCompletionRegistry(t, db, matchID)

	n, err := NewEventsCompletionPersister(db).Persist(ctx, sampleCompletionInput(matchID))
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if n != 2 {
		t.Errorf("eventsInserted = %d, want 2", n)
	}
	if got := countCompletionRows(t, db, "highlight_events", matchID); got != 2 {
		t.Errorf("highlight_events = %d, want 2", got)
	}
	if got := countCompletionRows(t, db, "killer_victim_pairs", matchID); got != 1 {
		t.Errorf("killer_victim_pairs = %d, want 1", got)
	}

	bits, loaded := completionRegistryState(t, db, matchID)
	if !loaded {
		t.Error("events_loaded = false, want true")
	}
	if bits&testBitEvents == 0 {
		t.Errorf("MBitEvents non positionné: bits=%d", bits)
	}
	if bits&testBitKillerVictim == 0 {
		t.Errorf("MBitKillerVictim non positionné: bits=%d", bits)
	}

	// Vérifie que la forme par-kill (gamertags + time_ms) est bien écrite.
	var killerGT, victimGT string
	var timeMS int64
	if err := db.QueryRowContext(ctx,
		`SELECT killer_gamertag, victim_gamertag, time_ms FROM killer_victim_pairs WHERE match_id = ?`, matchID).
		Scan(&killerGT, &victimGT, &timeMS); err != nil {
		t.Fatalf("kv row: %v", err)
	}
	if killerGT != "Killer1" || victimGT != "Victim1" || timeMS != 1000 {
		t.Errorf("kv row = (%q, %q, %d), want (Killer1, Victim1, 1000)", killerGT, victimGT, timeMS)
	}
}

// L'idempotence killer_victim repose sur DELETE-then-INSERT (table sans index).
// Deux passages → 1 seule paire (pas de doublon).
func TestEventsCompletionPersister_KVIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-idem-001"
	seedCompletionRegistry(t, db, matchID)

	p := NewEventsCompletionPersister(db)
	if _, err := p.Persist(ctx, sampleCompletionInput(matchID)); err != nil {
		t.Fatalf("Persist #1: %v", err)
	}
	if _, err := p.Persist(ctx, sampleCompletionInput(matchID)); err != nil {
		t.Fatalf("Persist #2: %v", err)
	}
	if got := countCompletionRows(t, db, "killer_victim_pairs", matchID); got != 1 {
		t.Errorf("killer_victim_pairs après 2 runs = %d, want 1 (idempotent)", got)
	}
	// Côté table canonique, l'idempotence a une AUTRE mécanique : deux passes ne
	// s'additionnent pas, la vue `_latest` n'en retient qu'une. C'est là que meurt le
	// doublon de killer_victim_pairs — il faut le vérifier, pas seulement l'écrire.
	if got := countCompletionRows(t, db, "match_kill_events_latest", matchID); got != 1 {
		t.Errorf("match_kill_events_latest après 2 runs = %d, want 1 — les deux passes "+
			"s'additionnent, ce qui est exactement le doublon (46,5 %%) que la table doit "+
			"rendre impossible", got)
	}
	if got := countCompletionRows(t, db, "match_kill_events", matchID); got != 2 {
		t.Errorf("match_kill_events (table brute) = %d, want 2 — la table est append-only, "+
			"les deux passes doivent y rester", got)
	}
}

// TestEventsCompletionPersister_EcritLaTableCanonique — LE SECOND PRODUCTEUR, ASSERTÉ.
//
// Constat J4R-2 : depuis le 2026-08-02 la complétion combat écrit `match_kill_events` en plus de
// `killer_victim_pairs` (`insertCompletionKillerVictim` → `persistCreditKillEvents`), et AUCUN
// test ne le vérifiait. Supprimer cet appel laissait toute la suite au vert : les couples
// continuaient d'arriver dans la table historique, et la table canonique se vidait en silence
// sur tout le chemin de complétion — c'est-à-dire sur les matchs dont le film arrive un cycle
// après le sync primaire.
//
// Le test porte sur les DEUX choses qui se perdent sans bruit : la présence des lignes, et la
// portée qu'elles déclarent.
func TestEventsCompletionPersister_EcritLaTableCanonique(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-canonique-001"
	seedCompletionRegistry(t, db, matchID)

	if _, err := NewEventsCompletionPersister(db).Persist(ctx, sampleCompletionInput(matchID)); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// (1) LES LIGNES SONT LÀ. Une mort par couple, servie par la vue `_latest`.
	if got := countCompletionRows(t, db, "match_kill_events_latest", matchID); got != 1 {
		t.Fatalf("match_kill_events_latest = %d ligne(s), want 1 — la complétion combat n'écrit "+
			"plus la table canonique ; les matchs dont le film arrive après le sync primaire n'y "+
			"auraient aucune mort, et rien ne le signalerait", got)
	}

	var victime, tueur, voie, origine string
	var timeMS int
	var feedPresent, assistKnown, publishable bool
	var sourceTag, sourceCat sql.NullInt64
	var diverge sql.NullBool
	var pctTueur, pctAssist sql.NullInt64
	if err := db.QueryRowContext(ctx, `
		SELECT victim_gamertag, feed_killer_gamertag, time_ms, feed_present, assist_known,
		       publishable, read_path, read_origin,
		       source_tag, source_category, diverges, killer_damage_pct, assist_damage_pct
		FROM match_kill_events_latest WHERE match_id = ?`, matchID).
		Scan(&victime, &tueur, &timeMS, &feedPresent, &assistKnown, &publishable, &voie, &origine,
			&sourceTag, &sourceCat, &diverge, &pctTueur, &pctAssist); err != nil {
		t.Fatalf("select ligne canonique: %v", err)
	}

	// (2) LE FAIT LUI-MÊME est intact : qui, qui, quand.
	if victime != "Victim1" || tueur != "Killer1" || timeMS != 1000 {
		t.Errorf("mort = (victime=%q, tueur=%q, t=%d), attendu (Victim1, Killer1, 1000) — "+
			"la traduction couple → mort a perdu ou décalé un champ", victime, tueur, timeMS)
	}
	if !feedPresent || !publishable {
		t.Errorf("feed_present=%v publishable=%v, attendu true/true : ces lignes valent "+
			"EXACTEMENT ce que valait killer_victim_pairs, qui était déjà lue ligne par ligne",
			feedPresent, publishable)
	}

	// (3) « ON NE SAIT PAS » — l'assertion qui motive le constat. Le kill-feed ne porte AUCUN
	//     assistant. Écrire `assist_known = TRUE` fabriquerait un « pas d'assistant » observé,
	//     alors que rien n'a été observé du tout : c'est le troisième état qui s'effondre.
	if assistKnown {
		t.Error("assist_known = TRUE : le kill-feed de la complétion ne porte aucun assistant. " +
			"« Connu » ici fabrique une mesure d'absence à partir d'une absence de mesure — les " +
			"trois états de l'assistant retombent à deux")
	}

	// (4) LA PORTÉE est déclarée, et c'est elle qui décide la préséance du film.
	if voie != killscope.ReadPathLiveFeed || origine != killscope.OriginCreditOnly {
		t.Errorf("portée = %q/%q, attendu %q/%q — la préséance se décide sur read_path",
			voie, origine, killscope.ReadPathLiveFeed, killscope.OriginCreditOnly)
	}

	// (5) « NULL n'est jamais zéro » : ce que la complétion ne mesure pas reste ABSENT.
	if sourceTag.Valid || sourceCat.Valid || diverge.Valid || pctTueur.Valid || pctAssist.Valid {
		t.Errorf("source/divergence/parts renseignées (tag=%v cat=%v div=%v pctT=%v pctA=%v) — "+
			"la source du dégât se lit dans le film, et il n'y en a pas ici : ces colonnes sont "+
			"NON MESURÉES, donc NULL",
			sourceTag.Valid, sourceCat.Valid, diverge.Valid, pctTueur.Valid, pctAssist.Valid)
	}
}

// TestEventsCompletionPersister_RecomposeAvecLeFilm — la complétion ENRICHIT, elle n'efface plus.
//
// AVANT le 2026-08-03 ce test s'appelait `_PreseanceFilm` et vérifiait l'inverse : la complétion
// REFUSAIT d'écrire dès qu'un film couvrait le match, pour ne pas effacer la source du dégât. Le
// refus protégeait la source — au prix des morts que le film ne porte pas (25,4 % en production).
//
// Les deux propriétés que le test tient maintenant, et il faut les DEUX :
//
//	(a) la mort de la complétion est PUBLIÉE (avant, elle disparaissait purement) ;
//	(b) la ligne du film, à un autre instant, survit AVEC sa source du dégât.
//
// La table HISTORIQUE continue d'être écrite : la recomposition ne porte que sur la table
// canonique, et ses ~20 lecteurs n'ont pas encore migré.
func TestEventsCompletionPersister_RecomposeAvecLeFilm(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-preseance-001"
	seedCompletionRegistry(t, db, matchID)

	// Une passe de FILM précède. Aucun film n'est nécessaire : ce qui décide est la voie.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, written_at, publishable, time_ms,
			victim_gamertag, feed_present, assist_known, source_tag, source_category,
			read_path, read_origin
		) VALUES (?, 'film-pass', 'film-rev', TIMESTAMP '2026-01-01 00:00:00', TRUE, 500,
			'VictimeFilm', TRUE, TRUE, 3735928559, 'Headshot', ?, 'credit-concordant')`, matchID, killscope.ReadPathFilmWalk); err != nil {
		t.Fatalf("seed passe de film: %v", err)
	}

	if _, err := NewEventsCompletionPersister(db).Persist(ctx, sampleCompletionInput(matchID)); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// (a) la table historique est bien écrite — la préséance ne la concerne pas.
	if got := countCompletionRows(t, db, "killer_victim_pairs", matchID); got != 1 {
		t.Errorf("killer_victim_pairs = %d, want 1 : la préséance ne porte QUE sur la table "+
			"canonique, les ~20 lecteurs historiques doivent continuer d'être servis", got)
	}

	// (b) la génération servie porte LES DEUX morts : celle de la complétion (1000 ms) et
	// l'orpheline du film (500 ms), qui garde sa source du dégât.
	var lignes int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?`, matchID).
		Scan(&lignes); err != nil {
		t.Fatalf("compte de la génération servie: %v", err)
	}
	if lignes != 2 {
		t.Fatalf("%d ligne(s) servies, attendu 2 (la mort de la complétion + l'orpheline du "+
			"film) — moins de 2 veut dire qu'une mort a disparu de la lecture", lignes)
	}

	var voie string
	var source sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT read_path, source_tag
		FROM match_kill_events_latest WHERE match_id = ? AND time_ms = 500`, matchID).
		Scan(&voie, &source); err != nil {
		t.Fatalf("select ligne du film: %v", err)
	}
	if voie != killscope.ReadPathFilmWalk || !source.Valid {
		t.Errorf("ligne du film : voie=%q source_tag valide=%v — la complétion l'a supplantée, "+
			"et la source du dégât fatal a disparu de la lecture", voie, source.Valid)
	}

	if err := db.QueryRowContext(ctx, `SELECT read_path
		FROM match_kill_events_latest WHERE match_id = ? AND time_ms = 1000`, matchID).
		Scan(&voie); err != nil {
		t.Fatalf("select mort de la complétion: %v", err)
	}
	if voie != killscope.ReadPathLiveFeed {
		t.Errorf("mort de la complétion : voie=%q, attendu %q — la portée reste PAR LIGNE",
			voie, killscope.ReadPathLiveFeed)
	}
}

// MarkKV=true positionne le bit killer_victim même sans paires (parité legacy :
// le bit était marqué dès que le calcul réussissait, y compris 0 paire).
func TestEventsCompletionPersister_EventsOnlyMarksKVBit(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-nopairs-001"
	seedCompletionRegistry(t, db, matchID)

	in := EventsCompletionInput{
		MatchID:         matchID,
		Events:          []HLEventCompletion{{XUID: "111", EventType: "medal", TimeMS: 500, TypeHint: 12}},
		Pairs:           nil,
		MarkKV:          true,
		EventsBit:       testBitEvents,
		KillerVictimBit: testBitKillerVictim,
	}
	n, err := NewEventsCompletionPersister(db).Persist(ctx, in)
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if n != 1 {
		t.Errorf("eventsInserted = %d, want 1", n)
	}
	if got := countCompletionRows(t, db, "killer_victim_pairs", matchID); got != 0 {
		t.Errorf("killer_victim_pairs = %d, want 0", got)
	}
	bits, loaded := completionRegistryState(t, db, matchID)
	if !loaded {
		t.Error("events_loaded = false, want true")
	}
	if bits&testBitEvents == 0 || bits&testBitKillerVictim == 0 {
		t.Errorf("bits = %d, want MBitEvents|MBitKillerVictim", bits)
	}
}

func TestEventsCompletionPersister_MarkNoFilmDefinitive(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-nofilm-001"
	seedCompletionRegistry(t, db, matchID)

	if err := NewEventsCompletionPersister(db).MarkNoFilmDefinitive(ctx, matchID, testBitEvents); err != nil {
		t.Fatalf("MarkNoFilmDefinitive: %v", err)
	}
	if got := countCompletionRows(t, db, "highlight_events", matchID); got != 0 {
		t.Errorf("highlight_events = %d, want 0 (no film)", got)
	}
	bits, loaded := completionRegistryState(t, db, matchID)
	if !loaded {
		t.Error("events_loaded = false, want true")
	}
	if bits&testBitEvents == 0 {
		t.Errorf("MBitEvents non positionné: bits=%d", bits)
	}
}

// TestEventsCompletionPersister_MarkEventsEmptyDefinitive : un chunk récupéré mais
// 0 event légitime pose events_empty=TRUE + le bit events, SANS mentir sur
// events_loaded (qui reste FALSE — aucun event chargé). Sort le match du retry set
// tout en restant honnête et auditable.
func TestEventsCompletionPersister_MarkEventsEmptyDefinitive(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	matchID := "m-empty-001"
	seedCompletionRegistry(t, db, matchID)

	if err := NewEventsCompletionPersister(db).MarkEventsEmptyDefinitive(ctx, matchID, testBitEvents); err != nil {
		t.Fatalf("MarkEventsEmptyDefinitive: %v", err)
	}
	if got := countCompletionRows(t, db, "highlight_events", matchID); got != 0 {
		t.Errorf("highlight_events = %d, want 0 (vide)", got)
	}

	var eventsEmpty, eventsLoaded sql.NullBool
	var bits int64
	if err := db.QueryRowContext(ctx,
		`SELECT backfill_completed, events_loaded, events_empty FROM match_registry WHERE match_id = ?`, matchID).
		Scan(&bits, &eventsLoaded, &eventsEmpty); err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if !eventsEmpty.Bool {
		t.Error("events_empty = false, want true (chunk vide définitif)")
	}
	if eventsLoaded.Bool {
		t.Error("events_loaded = true, want false — on ne doit PAS mentir : aucun event chargé")
	}
	if bits&testBitEvents == 0 {
		t.Errorf("MBitEvents non positionné (sortie du retry set): bits=%d", bits)
	}
}

func TestEventsCompletionPersister_EmptyMatchID(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	if _, err := NewEventsCompletionPersister(db).Persist(ctx, EventsCompletionInput{}); err == nil {
		t.Error("Persist avec MatchID vide: attendu une erreur, got nil")
	}
}
