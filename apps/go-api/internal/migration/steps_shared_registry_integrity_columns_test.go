package migration

// steps_shared_registry_integrity_columns_test.go — reconstruction de match_registry qui retire
// participants_loaded / player_count / idx_registry_player_count et pose le bit participants,
// sur une base synthétique à la forme des bases réelles (index sur des colonnes situées APRÈS
// les colonnes retirées, vues v_match_full et mv_player_matches d'avant le retrait).

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// fixtureRegistreIntegriteSQL : match_registry aux colonnes lues par mv_player_matches, plus les
// deux colonnes à retirer AU MILIEU de la table (comme en production), et des index dont l'un
// porte sur une colonne d'ordinal supérieur (season_id).
const fixtureRegistreIntegriteSQL = `
CREATE TABLE match_registry (
	match_id VARCHAR PRIMARY KEY,
	start_time TIMESTAMP NOT NULL,
	end_time TIMESTAMP,
	start_time_utc TIMESTAMPTZ,
	end_time_utc TIMESTAMPTZ,
	playlist_id VARCHAR, playlist_name VARCHAR, playlist_name_fr VARCHAR,
	map_id VARCHAR, map_name VARCHAR, map_name_fr VARCHAR,
	pair_name VARCHAR, pair_name_fr VARCHAR, pair_id VARCHAR,
	game_variant_id VARCHAR, game_variant_name VARCHAR, mode_category VARCHAR,
	is_ranked BOOLEAN DEFAULT FALSE, is_firefight BOOLEAN DEFAULT FALSE,
	duration_seconds INTEGER, playable_duration_seconds INTEGER,
	team_0_score INTEGER, team_1_score INTEGER, team_0_ps_score INTEGER, team_1_ps_score INTEGER,
	backfill_completed INTEGER DEFAULT 0,
	participants_loaded BOOLEAN DEFAULT FALSE,
	events_loaded BOOLEAN DEFAULT FALSE,
	player_count SMALLINT DEFAULT 0,
	created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
	season_id VARCHAR
);
CREATE INDEX idx_registry_player_count ON match_registry(player_count);
CREATE INDEX idx_registry_map ON match_registry(map_id);
CREATE INDEX idx_mr_start_time ON match_registry(start_time);
CREATE INDEX idx_match_registry_season_id ON match_registry(season_id);
CREATE TABLE match_participants (
	match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR, team_id INTEGER, outcome INTEGER,
	rank INTEGER, score INTEGER, kills INTEGER, deaths INTEGER, assists INTEGER,
	kda DOUBLE, accuracy DOUBLE, shots_fired INTEGER, shots_hit INTEGER,
	damage_dealt DOUBLE, damage_taken DOUBLE, personal_score INTEGER,
	time_played_seconds INTEGER, avg_life_seconds DOUBLE, headshot_kills INTEGER,
	max_killing_spree INTEGER, grenade_kills INTEGER, melee_kills INTEGER,
	power_weapon_kills INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
	kills_expected DOUBLE, deaths_expected DOUBLE, backfill_bits BIGINT
);
CREATE VIEW v_match_full AS SELECT mr.* FROM match_registry mr;
`

// mvPlayerMatchesAvantRetrait : mv_player_matches telle que les bases d'avant le retrait la
// portent — la définition canonique, plus mr.player_count.
func mvPlayerMatchesAvantRetrait(t *testing.T) string {
	t.Helper()
	const ancre = "mr.team_1_ps_score,"
	if !strings.Contains(mvPlayerMatchesViewSQL, ancre) {
		t.Fatalf("ancre %q absente de mvPlayerMatchesViewSQL", ancre)
	}
	return strings.Replace(mvPlayerMatchesViewSQL, ancre, ancre+"\n\t\t\tmr.player_count,", 1)
}

// registreIntegriteLignes : match_id, backfill_completed (nil = NULL), nombre de participants.
var registreIntegriteLignes = []struct {
	id      string
	bits    any
	joueurs int
}{
	{"m1", 513, 2},   // bit déjà posé (+ bit médailles) : inchangé
	{"m2", 1, 3},     // roster sans le bit : 513
	{"m3", nil, 1},   // roster, NULL : 512
	{"m4", 0, 2},     // roster, 0 : 512
	{"m5", 0, 0},     // sans roster : inchangé
	{"m6", nil, 0},   // sans roster, NULL : inchangé
	{"m7", 4096, 0},  // sans roster, autre bit : inchangé
	{"m8", 65540, 4}, // roster, bits événements + skill : 66052
}

func openRegistreIntegrite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := execScript(db, fixtureRegistreIntegriteSQL); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if _, err := db.Exec(mvPlayerMatchesAvantRetrait(t)); err != nil {
		t.Fatalf("fixture mv_player_matches: %v", err)
	}
	for i, r := range registreIntegriteLignes {
		if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, map_id, mode_category,
			backfill_completed, participants_loaded, player_count, season_id)
			VALUES (?, TIMESTAMP '2026-05-25 12:00:00', 'map1', 'Slayer', ?, ?, ?, 'S1')`,
			r.id, r.bits, i%2 == 0, i); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
		}
		for j := 0; j < r.joueurs; j++ {
			if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, kills) VALUES (?, ?, ?)`,
				r.id, "x"+string(rune('a'+j)), j); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
		}
	}
	return db
}

// bitsParMatch lit backfill_completed (« NULL » pour NULL) sans passer par le code testé.
func bitsParMatch(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	rows, err := db.Query(`SELECT match_id, COALESCE(CAST(backfill_completed AS VARCHAR), 'NULL') FROM match_registry`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() //nolint:errcheck // test
	out := map[string]string{}
	for rows.Next() {
		var id, v string
		if err := rows.Scan(&id, &v); err != nil {
			t.Fatal(err)
		}
		out[id] = v
	}
	return out
}

func colonnesPresentes(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	return scanInt(t, db, `SELECT COUNT(*) FROM duckdb_columns() WHERE table_name = 'match_registry'
		AND column_name IN ('participants_loaded', 'player_count')`)
}

func nomsIndex(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT index_name FROM duckdb_indexes() WHERE table_name = 'match_registry' ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() //nolint:errcheck // test
	var noms []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		noms = append(noms, n)
	}
	return strings.Join(noms, ",")
}

func TestRegistryIntegrity_RetireColonnesIndexEtPoseLeBit(t *testing.T) {
	db := openRegistreIntegrite(t)
	ctx := t.Context()
	colonnesAvant, err := colonnesDuRegistre(ctx, db, colonnesIntegriteRetirees)
	if err != nil {
		t.Fatal(err)
	}
	vuesAvant := map[string]EmpreinteVue{}
	for _, v := range []string{"v_match_full", "mv_player_matches"} {
		e, err := empreinteHorsColonnes(ctx, db, v, []string{"participants_loaded", "player_count", "backfill_completed"})
		if err != nil {
			t.Fatal(err)
		}
		vuesAvant[v] = e
	}
	oid := tableOID(t, db, registryTable)

	if err := ApplyRegistryParticipantsBitRebuild(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	if tableOID(t, db, registryTable) == oid {
		t.Error("la table devait être reconstruite (swap)")
	}
	if n := colonnesPresentes(t, db); n != 0 {
		t.Errorf("%d colonne(s) d'intégrité encore présentes", n)
	}
	if got, want := nomsIndex(t, db), "idx_match_registry_season_id,idx_mr_start_time,idx_registry_map"; got != want {
		t.Errorf("index = %s, attendu %s", got, want)
	}
	colonnesApres, err := colonnesDuRegistre(ctx, db, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(colonnesApres, "|") != strings.Join(colonnesAvant, "|") {
		t.Errorf("colonnes restantes changées :\navant %v\naprès %v", colonnesAvant, colonnesApres)
	}
	want := map[string]string{"m1": "513", "m2": "513", "m3": "512", "m4": "512", "m5": "0", "m6": "NULL",
		"m7": "4096", "m8": "66052"}
	got := bitsParMatch(t, db)
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s: backfill_completed %s, attendu %s", id, got[id], w)
		}
	}
	for v, e := range vuesAvant {
		apres, err := empreinteHorsColonnes(ctx, db, v, []string{"backfill_completed"})
		if err != nil {
			t.Fatalf("vue %s illisible après l'échange : %v", v, err)
		}
		if apres != e {
			t.Errorf("vue %s changée : %+v -> %+v", v, e, apres)
		}
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM duckdb_columns() WHERE table_name = 'mv_player_matches'
		AND column_name = 'player_count'`); n != 0 {
		t.Error("mv_player_matches expose encore player_count")
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM v_match_full`); n != int64(len(registreIntegriteLignes)) {
		t.Errorf("v_match_full = %d lignes", n)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM duckdb_constraints() WHERE table_name = 'match_registry'
		AND constraint_type = 'PRIMARY KEY'`); n != 1 {
		t.Errorf("clé primaire de match_registry : %d", n)
	}
}

func TestRegistryIntegrity_SansBitNeTouchePasBackfill(t *testing.T) {
	db := openRegistreIntegrite(t)
	avant := bitsParMatch(t, db)
	if err := applyRegistryIntegrity(db, false); err != nil {
		t.Fatalf("migration: %v", err)
	}
	if n := colonnesPresentes(t, db); n != 0 {
		t.Errorf("%d colonne(s) d'intégrité encore présentes", n)
	}
	for id, v := range bitsParMatch(t, db) {
		if avant[id] != v {
			t.Errorf("%s: backfill_completed %s -> %s sans bit demandé", id, avant[id], v)
		}
	}
}

func TestRegistryIntegrity_IdempotenteEtEnchainee(t *testing.T) {
	db := openRegistreIntegrite(t)
	if err := ApplyRegistryParticipantsBitRebuild(db); err != nil {
		t.Fatal(err)
	}
	oid := tableOID(t, db, registryTable)
	avant := bitsParMatch(t, db)
	// Second passage du step title-owned, puis le step global qui le suit : aucune écriture.
	if err := ApplyRegistryParticipantsBitRebuild(db); err != nil {
		t.Fatalf("second passage: %v", err)
	}
	if err := applyRegistryIntegrity(db, false); err != nil {
		t.Fatalf("step global après le title-owned: %v", err)
	}
	if got := tableOID(t, db, registryTable); got != oid {
		t.Errorf("base déjà migrée reconstruite (oid %d -> %d)", oid, got)
	}
	for id, v := range bitsParMatch(t, db) {
		if avant[id] != v {
			t.Errorf("%s: %s -> %s au second passage", id, avant[id], v)
		}
	}
}

// TestRegistryIntegrity_BitSeulSurBaseDejaSansColonnes : colonnes déjà absentes (base neuve ou
// step global déjà passé) mais bits manquants : la reconstruction a lieu pour le bit seul.
func TestRegistryIntegrity_BitSeulSurBaseDejaSansColonnes(t *testing.T) {
	db := openRegistreIntegrite(t)
	if err := applyRegistryIntegrity(db, false); err != nil {
		t.Fatal(err)
	}
	if err := ApplyRegistryParticipantsBitRebuild(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	if got := bitsParMatch(t, db); got["m2"] != "513" || got["m3"] != "512" || got["m5"] != "0" {
		t.Errorf("bits = %v", got)
	}
}

func TestRegistryIntegrity_AlterationAnnuleLEchange(t *testing.T) {
	db := openRegistreIntegrite(t)
	buildIntegriteSupplementaire = func() []string {
		return []string{`UPDATE match_registry__integrite SET map_id = 'altere' WHERE match_id = 'm5'`}
	}
	t.Cleanup(func() { buildIntegriteSupplementaire = func() []string { return nil } })
	oid := tableOID(t, db, registryTable)

	err := ApplyRegistryParticipantsBitRebuild(db)
	if err == nil || !strings.Contains(err.Error(), "projection attendue") {
		t.Fatalf("erreur %v, attendu un refus de la garde d'empreinte", err)
	}
	if got := tableOID(t, db, registryTable); got != oid {
		t.Errorf("table d'origine remplacée malgré l'échec (oid %d -> %d)", oid, got)
	}
	if n := colonnesPresentes(t, db); n != 2 {
		t.Errorf("colonnes présentes = %d, la transaction devait être annulée", n)
	}
	if !strings.Contains(nomsIndex(t, db), "idx_registry_player_count") {
		t.Error("l'index d'origine doit rester après annulation")
	}
}

// TestRegistryIntegrity_VueInconnueQuiCiteUneColonneRetireeRefusee : une vue hors du code qui
// lit une colonne retirée ne peut pas être rejouée : l'échange est annulé, la base reste intacte.
func TestRegistryIntegrity_VueInconnueQuiCiteUneColonneRetireeRefusee(t *testing.T) {
	db := openRegistreIntegrite(t)
	if _, err := db.Exec(`CREATE VIEW v_roster_annonce AS SELECT match_id, player_count FROM match_registry`); err != nil {
		t.Fatal(err)
	}
	oid := tableOID(t, db, registryTable)
	if err := ApplyRegistryParticipantsBitRebuild(db); err == nil {
		t.Fatal("une vue qui cite une colonne retirée doit faire échouer la reconstruction")
	}
	if got := tableOID(t, db, registryTable); got != oid {
		t.Errorf("table remplacée malgré l'échec (oid %d -> %d)", oid, got)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM v_roster_annonce`); n != int64(len(registreIntegriteLignes)) {
		t.Errorf("v_roster_annonce = %d lignes après annulation", n)
	}
}

// TestRegistryIntegrity_VueCanoniqueDivergenteRefusee : une base dont mv_player_matches n'est
// pas la définition canonique (plus mr.player_count) verrait la vue changer de sens en la
// rejouant : la garde d'empreinte refuse l'échange.
func TestRegistryIntegrity_VueCanoniqueDivergenteRefusee(t *testing.T) {
	db := openRegistreIntegrite(t)
	if _, err := db.Exec(`CREATE OR REPLACE VIEW mv_player_matches AS SELECT mr.match_id, mr.player_count, mp.xuid
		FROM match_registry mr JOIN match_participants mp ON mr.match_id = mp.match_id`); err != nil {
		t.Fatal(err)
	}
	oid := tableOID(t, db, registryTable)
	err := ApplyRegistryParticipantsBitRebuild(db)
	if err == nil || !strings.Contains(err.Error(), "vue mv_player_matches changée") {
		t.Fatalf("erreur %v, attendu le refus de la garde de vue", err)
	}
	if got := tableOID(t, db, registryTable); got != oid {
		t.Errorf("table remplacée malgré l'échec (oid %d -> %d)", oid, got)
	}
}

// TestRegistryIntegrity_FormeHalo5 : player_count seul (INTEGER, sans index),
// participants_loaded absente, backfill_completed NULL partout ; step global seulement.
func TestRegistryIntegrity_FormeHalo5(t *testing.T) {
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "h5.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ddl := strings.NewReplacer(
		"\tparticipants_loaded BOOLEAN DEFAULT FALSE,\n", "",
		"player_count SMALLINT DEFAULT 0", "player_count INTEGER",
		"CREATE INDEX idx_registry_player_count ON match_registry(player_count);\n", "",
	).Replace(fixtureRegistreIntegriteSQL)
	if err := execScript(db, ddl); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if _, err := db.Exec(mvPlayerMatchesAvantRetrait(t)); err != nil {
		t.Fatalf("fixture mv_player_matches: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, backfill_completed, player_count)
		VALUES ('h1', TIMESTAMP '2018-01-01 00:00:00', NULL, NULL), ('h2', TIMESTAMP '2018-01-02 00:00:00', NULL, NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid) VALUES ('h1', 'x1')`); err != nil {
		t.Fatal(err)
	}
	if err := applyRegistryIntegrity(db, false); err != nil {
		t.Fatalf("migration: %v", err)
	}
	if n := colonnesPresentes(t, db); n != 0 {
		t.Errorf("%d colonne(s) d'intégrité encore présentes", n)
	}
	if got := bitsParMatch(t, db); got["h1"] != "NULL" || got["h2"] != "NULL" {
		t.Errorf("bits = %v, un titre qui n'écrit pas le bit ne le reçoit pas", got)
	}
	if got, want := nomsIndex(t, db), "idx_match_registry_season_id,idx_mr_start_time,idx_registry_map"; got != want {
		t.Errorf("index = %s, attendu %s", got, want)
	}
}

func TestProjectionIntegrite(t *testing.T) {
	cas := []struct {
		p    planIntegrite
		want string
	}{
		{planIntegrite{retirees: []string{"player_count"}}, "SELECT * EXCLUDE (player_count) FROM match_registry"},
		{planIntegrite{retirees: []string{"participants_loaded", "player_count"}},
			"SELECT * EXCLUDE (participants_loaded, player_count) FROM match_registry"},
	}
	for _, c := range cas {
		if got := projectionIntegrite(c.p); got != c.want {
			t.Errorf("projection %q, attendu %q", got, c.want)
		}
	}
	if got := projectionIntegrite(planIntegrite{poserBit: true}); !strings.Contains(got, "| 512") ||
		!strings.Contains(got, "REPLACE") || strings.Contains(got, "EXCLUDE") {
		t.Errorf("projection avec bit : %q", got)
	}
}
