//go:build integration

// Package migration — steps_written_at_utc_default_integration_test.go : la réparation
// du DEFAULT de `written_at` sur une base LEGACY (lot S2, suite de R1).
//
// La mesure ne doit pas dépendre de la machine : chaque test force le fuseau de session
// à `Pacific/Kiritimati` (UTC+14, sans heure d'été — donc stable toute l'année). Sur une
// base non réparée l'écart mesuré est exactement 14 h ; après réparation il est nul.
// C'est la même garde que TestRepriseHorodateEnUTC (R1) : sans elle, le test serait vert
// sur ce poste (UTC+2) et rouge à l'ouest de Greenwich.
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// fuseauLointain : UTC+14 sans heure d'été. Toute fuite de fuseau se voit à 14 h près.
const fuseauLointain = "Pacific/Kiritimati"

// ouvrirBaseLegacyWrittenAt ouvre une base in-memory au fuseau lointain, portant une
// table append-only telle qu'une migration HISTORIQUE l'a créée : `written_at` naive avec
// un DEFAULT sensible au fuseau.
func ouvrirBaseLegacyWrittenAt(t *testing.T, ddl string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("SET TimeZone = '" + fuseauLointain + "'"); err != nil {
		t.Fatalf("SET TimeZone: %v", err)
	}
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("DDL legacy: %v", err)
	}
	return db
}

// ecartAuTempsUTC insère une ligne SANS written_at (donc via le DEFAULT) et retourne
// l'écart, en minutes, entre la valeur posée et l'instant UTC courant.
//
// La ligne à mesurer est repérée par `k` CROISSANT, jamais par `written_at DESC` : une
// ligne écrite AVANT la réparation est datée dans le futur et gagnerait ce tri. C'est le
// défaut même que ce lot corrige — un harnais qui trie sur written_at mesurerait la
// mauvaise ligne et déclarerait la réparation inopérante.
func ecartAuTempsUTC(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO ` + table + ` (k) SELECT COALESCE(MAX(k), 0) + 1 FROM ` + table); err != nil {
		t.Fatalf("insert %s: %v", table, err)
	}
	var ecart int
	err := db.QueryRow(`
		SELECT CAST(round(date_diff('second',
			CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), written_at) / 60.0) AS INTEGER)
		FROM ` + table + ` ORDER BY k DESC LIMIT 1`).Scan(&ecart)
	if err != nil {
		t.Fatalf("mesure écart %s: %v", table, err)
	}
	return ecart
}

// TestWrittenAtDefaultUTC_ReparBaseLegacy : sur une base existante dont le DEFAULT date
// sur le fuseau de session, la ligne insérée sans written_at part 14 h dans le futur ;
// après EnsureWrittenAtDefaultsUTC elle tombe à l'heure UTC.
func TestWrittenAtDefaultUTC_ReparBaseLegacy(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_now (k INTEGER, written_at TIMESTAMP NOT NULL DEFAULT now());
		CREATE TABLE t_courant (k INTEGER, written_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
	`)

	// AVANT : le défaut est bien là, et il vaut exactement l'offset du fuseau.
	if got := ecartAuTempsUTC(t, db, "t_now"); got != 14*60 {
		t.Fatalf("t_now avant réparation: écart attendu +840 min (UTC+14), obtenu %+d", got)
	}
	if got := ecartAuTempsUTC(t, db, "t_courant"); got != 14*60 {
		t.Fatalf("t_courant avant réparation: écart attendu +840 min, obtenu %+d", got)
	}

	n, err := EnsureWrittenAtDefaultsUTC(db)
	if err != nil {
		t.Fatalf("EnsureWrittenAtDefaultsUTC: %v", err)
	}
	if n != 2 {
		t.Errorf("tables réparées: want 2, got %d", n)
	}

	if got := ecartAuTempsUTC(t, db, "t_now"); got != 0 {
		t.Errorf("t_now après réparation: écart attendu 0, obtenu %+d min — le DEFAULT fuit encore le fuseau", got)
	}
	if got := ecartAuTempsUTC(t, db, "t_courant"); got != 0 {
		t.Errorf("t_courant après réparation: écart attendu 0, obtenu %+d min", got)
	}
}

// TestWrittenAtDefaultUTC_Idempotent : un second passage ne répare plus rien (le DEFAULT
// normalisé par DuckDB ne matche plus le prédicat) — le step peut rejouer sans effet.
func TestWrittenAtDefaultUTC_Idempotent(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_now (k INTEGER, written_at TIMESTAMP NOT NULL DEFAULT now());
	`)
	if _, err := EnsureWrittenAtDefaultsUTC(db); err != nil {
		t.Fatalf("1re passe: %v", err)
	}
	n, err := EnsureWrittenAtDefaultsUTC(db)
	if err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	if n != 0 {
		t.Errorf("2e passe: want 0 table réparée, got %d — le step n'est pas idempotent", n)
	}
}

// TestWrittenAtDefaultUTC_LaisseLesColonnesTZ : une colonne TIMESTAMPTZ garde l'instant
// absolu, elle n'a pas le défaut. La réparer changerait sa sémantique pour rien.
func TestWrittenAtDefaultUTC_LaisseLesColonnesTZ(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_tz (k INTEGER, written_at TIMESTAMPTZ NOT NULL DEFAULT now());
	`)
	n, err := EnsureWrittenAtDefaultsUTC(db)
	if err != nil {
		t.Fatalf("EnsureWrittenAtDefaultsUTC: %v", err)
	}
	if n != 0 {
		t.Errorf("colonne TIMESTAMPTZ: want 0 réparation, got %d", n)
	}
}

// TestWrittenAtDefaultUTC_IgnoreLesBasesAttachees : metadata est ATTACHée sur shared.
// Sans le filtre `database_name = current_database()`, le step tenterait un ALTER sur une
// base qu'il n'a pas à toucher (et qui peut être ouverte en lecture seule).
func TestWrittenAtDefaultUTC_IgnoreLesBasesAttachees(t *testing.T) {
	autre := t.TempDir() + "/autre.duckdb"
	voisine, err := sql.Open("duckdb", autre)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := voisine.Exec(
		`CREATE TABLE t_voisine (k INTEGER, written_at TIMESTAMP NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	if err := voisine.Close(); err != nil {
		t.Fatal(err)
	}

	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_locale (k INTEGER, written_at TIMESTAMP NOT NULL DEFAULT now());
	`)
	if _, err := db.Exec(`ATTACH '` + autre + `' AS voisine (READ_ONLY)`); err != nil {
		t.Fatalf("ATTACH: %v", err)
	}

	n, err := EnsureWrittenAtDefaultsUTC(db)
	if err != nil {
		t.Fatalf("EnsureWrittenAtDefaultsUTC avec base attachée: %v", err)
	}
	if n != 1 {
		t.Errorf("tables réparées: want 1 (la locale seule), got %d", n)
	}
	// La voisine n'a pas bougé : son DEFAULT est resté celui d'origine.
	var defaut string
	if err := db.QueryRow(`
		SELECT column_default FROM duckdb_columns()
		WHERE database_name = 'voisine' AND table_name = 't_voisine' AND column_name = 'written_at'`).
		Scan(&defaut); err != nil {
		t.Fatalf("lecture du DEFAULT voisin: %v", err)
	}
	if defaut != "now()" {
		t.Errorf("base attachée touchée: DEFAULT voisin = %q, attendu \"now()\"", defaut)
	}
}

// TestWrittenAtDefaultUTC_StepEnregistreSurLesCinqTargets : le step doit tourner sur
// TOUTE base — chacune porte un sous-ensemble différent des tables append-only.
func TestWrittenAtDefaultUTC_StepEnregistreSurLesCinqTargets(t *testing.T) {
	attendus := map[TargetDB]bool{
		TargetShared: false, TargetPlayer: false, TargetSharedPvE: false,
		TargetSharedSocial: false, TargetMetadata: false,
	}
	for _, m := range All() {
		if _, ok := attendus[m.TargetDB]; ok && m.Name == writtenAtUTCStepName(m.TargetDB) {
			attendus[m.TargetDB] = true
		}
	}
	for target, trouve := range attendus {
		if !trouve {
			t.Errorf("step written_at UTC absent du registre pour la target %q", target)
		}
	}
}

// TestWrittenAtDefaultUTC_ReparApresRunForDB : bout en bout par le runner — une base
// legacy passée à RunForDB ressort avec un DEFAULT en UTC (c'est le chemin du boot).
func TestWrittenAtDefaultUTC_ReparApresRunForDB(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_legacy (k INTEGER, written_at TIMESTAMP NOT NULL DEFAULT now());
	`)
	if err := RunSteps(db, TargetPlayer, []Migration{{
		Name:        writtenAtUTCStepName(TargetPlayer),
		TargetDB:    TargetPlayer,
		Description: "written_at UTC",
		ApplySchema: func(db *sql.DB) error { _, err := EnsureWrittenAtDefaultsUTC(db); return err },
	}}); err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if got := ecartAuTempsUTC(t, db, "t_legacy"); got != 0 {
		t.Errorf("après RunSteps: écart attendu 0, obtenu %+d min", got)
	}
	// Et le step est tracé : un second boot ne le rejoue pas.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`,
		writtenAtUTCStepName(TargetPlayer)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("schema_migrations: want 1 ligne pour le step, got %d", n)
	}
}
