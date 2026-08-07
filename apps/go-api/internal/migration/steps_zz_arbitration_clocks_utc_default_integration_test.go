//go:build integration

// Package migration — steps_zz_arbitration_clocks_utc_default_integration_test.go : la
// réparation du DEFAULT de TOUTE colonne d'horodatage sur une base LEGACY (lot S6).
//
// Même garde que le test jumeau de S2 (steps_written_at_utc_default_integration_test.go) :
// le fuseau de session est forcé à `Pacific/Kiritimati` (UTC+14, sans heure d'été), donc
// l'écart mesuré vaut exactement 14 h avant réparation et 0 après. Sans cela le test
// serait vert sur ce poste (UTC+2) et rouge à l'ouest de Greenwich.
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ecartColonneAuTempsUTC insère une ligne SANS la colonne visée (donc via son DEFAULT) et
// retourne l'écart, en minutes, entre la valeur posée et l'instant UTC courant.
//
// La ligne à mesurer est repérée par `k` CROISSANT, jamais par la colonne d'horodatage :
// une ligne écrite AVANT la réparation est datée dans le futur et gagnerait ce tri — c'est
// le défaut même que ce lot corrige.
func ecartColonneAuTempsUTC(t *testing.T, db *sql.DB, table, colonne string) int {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO ` + table + ` (k) SELECT COALESCE(MAX(k), 0) + 1 FROM ` + table); err != nil {
		t.Fatalf("insert %s: %v", table, err)
	}
	var ecart int
	err := db.QueryRow(`
		SELECT CAST(round(date_diff('second',
			CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), ` + colonne + `) / 60.0) AS INTEGER)
		FROM ` + table + ` ORDER BY k DESC LIMIT 1`).Scan(&ecart)
	if err != nil {
		t.Fatalf("mesure écart %s.%s: %v", table, colonne, err)
	}
	return ecart
}

// TestArbitrationClocksUTC_ReparToutesLesColonnes : le step répare les colonnes
// d'horodatage QUEL QUE SOIT LEUR NOM — c'est toute la différence avec son jumeau de S2,
// qui ne voyait que `written_at`. `computed_at` est le cas réel qui a motivé le lot :
// il arbitre `lusr_component_history_latest`.
func TestArbitrationClocksUTC_ReparToutesLesColonnes(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_composantes (k INTEGER, computed_at TIMESTAMP NOT NULL DEFAULT now());
		CREATE TABLE t_snapshots  (k INTEGER, snapshot_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE t_creation   (k INTEGER, created_at  TIMESTAMP NOT NULL DEFAULT now());
	`)

	// AVANT : chaque DEFAULT vaut exactement l'offset du fuseau.
	for _, c := range []struct{ table, colonne string }{
		{"t_composantes", "computed_at"},
		{"t_snapshots", "snapshot_at"},
		{"t_creation", "created_at"},
	} {
		if got := ecartColonneAuTempsUTC(t, db, c.table, c.colonne); got != 14*60 {
			t.Fatalf("%s.%s avant réparation: écart attendu +840 min (UTC+14), obtenu %+d",
				c.table, c.colonne, got)
		}
	}

	n, err := EnsureTimestampDefaultsUTC(db)
	if err != nil {
		t.Fatalf("EnsureTimestampDefaultsUTC: %v", err)
	}
	if n != 3 {
		t.Errorf("colonnes réparées: want 3, got %d", n)
	}

	for _, c := range []struct{ table, colonne string }{
		{"t_composantes", "computed_at"},
		{"t_snapshots", "snapshot_at"},
		{"t_creation", "created_at"},
	} {
		if got := ecartColonneAuTempsUTC(t, db, c.table, c.colonne); got != 0 {
			t.Errorf("%s.%s après réparation: écart attendu 0, obtenu %+d min — le DEFAULT fuit encore le fuseau",
				c.table, c.colonne, got)
		}
	}
}

// TestArbitrationClocksUTC_NeReecritAucuneLigne : contrainte NON NÉGOCIABLE du lot — la
// réparation ne touche QUE le DEFAULT. Un UPDATE de masse sur ces tables serait le vecteur
// ART que la campagne append-only a éteint (ADR 0019/0026) ; l'historique déjà biaisé est
// assumé tel quel.
func TestArbitrationClocksUTC_NeReecritAucuneLigne(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_hist (k INTEGER, computed_at TIMESTAMP NOT NULL DEFAULT now());
	`)
	// Une ligne legacy, datée au fuseau de session (donc dans le futur).
	if _, err := db.Exec(`INSERT INTO t_hist (k) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	var avant string
	if err := db.QueryRow(`SELECT CAST(computed_at AS VARCHAR) FROM t_hist WHERE k = 1`).Scan(&avant); err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureTimestampDefaultsUTC(db); err != nil {
		t.Fatalf("EnsureTimestampDefaultsUTC: %v", err)
	}

	var apres string
	if err := db.QueryRow(`SELECT CAST(computed_at AS VARCHAR) FROM t_hist WHERE k = 1`).Scan(&apres); err != nil {
		t.Fatal(err)
	}
	if avant != apres {
		t.Errorf("la ligne existante a été RÉÉCRITE (%q → %q) — le step doit se borner au DEFAULT", avant, apres)
	}
}

// TestArbitrationClocksUTC_Idempotent : un second passage ne répare plus rien (le DEFAULT
// normalisé par DuckDB ne matche plus le prédicat) — le step peut rejouer sans effet.
func TestArbitrationClocksUTC_Idempotent(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_idem (k INTEGER, computed_at TIMESTAMP NOT NULL DEFAULT now());
	`)
	if _, err := EnsureTimestampDefaultsUTC(db); err != nil {
		t.Fatalf("1re passe: %v", err)
	}
	n, err := EnsureTimestampDefaultsUTC(db)
	if err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	if n != 0 {
		t.Errorf("2e passe: want 0 colonne réparée, got %d — le step n'est pas idempotent", n)
	}
}

// TestArbitrationClocksUTC_LaisseLesColonnesTZ : une colonne TIMESTAMPTZ garde l'instant
// absolu, elle n'a pas le défaut. La réparer changerait sa sémantique pour rien
// (media_files.indexed_at est dans ce cas).
func TestArbitrationClocksUTC_LaisseLesColonnesTZ(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_tz (k INTEGER, indexed_at TIMESTAMPTZ NOT NULL DEFAULT now());
	`)
	n, err := EnsureTimestampDefaultsUTC(db)
	if err != nil {
		t.Fatalf("EnsureTimestampDefaultsUTC: %v", err)
	}
	if n != 0 {
		t.Errorf("colonne TIMESTAMPTZ: want 0 réparation, got %d", n)
	}
}

// TestArbitrationClocksUTC_IgnoreLesBasesAttachees : metadata est ATTACHée sur shared.
// Sans le filtre `database_name = current_database()`, le step tenterait un ALTER sur une
// base qu'il n'a pas à toucher (et qui peut être ouverte en lecture seule).
func TestArbitrationClocksUTC_IgnoreLesBasesAttachees(t *testing.T) {
	autre := t.TempDir() + "/autre.duckdb"
	voisine, err := sql.Open("duckdb", autre)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := voisine.Exec(
		`CREATE TABLE t_voisine (k INTEGER, computed_at TIMESTAMP NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	if err := voisine.Close(); err != nil {
		t.Fatal(err)
	}

	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_locale (k INTEGER, computed_at TIMESTAMP NOT NULL DEFAULT now());
	`)
	if _, err := db.Exec(`ATTACH '` + autre + `' AS voisine (READ_ONLY)`); err != nil {
		t.Fatalf("ATTACH: %v", err)
	}

	n, err := EnsureTimestampDefaultsUTC(db)
	if err != nil {
		t.Fatalf("EnsureTimestampDefaultsUTC avec base attachée: %v", err)
	}
	if n != 1 {
		t.Errorf("colonnes réparées: want 1 (la locale seule), got %d", n)
	}
	var defaut string
	if err := db.QueryRow(`
		SELECT column_default FROM duckdb_columns()
		WHERE database_name = 'voisine' AND table_name = 't_voisine' AND column_name = 'computed_at'`).
		Scan(&defaut); err != nil {
		t.Fatalf("lecture du DEFAULT voisin: %v", err)
	}
	if defaut != "now()" {
		t.Errorf("base attachée touchée: DEFAULT voisin = %q, attendu \"now()\"", defaut)
	}
}

// TestArbitrationClocksUTC_StepEnregistreSurLesCinqTargets : le step doit tourner sur
// TOUTE base — chacune porte un sous-ensemble différent des tables d'horodatage.
func TestArbitrationClocksUTC_StepEnregistreSurLesCinqTargets(t *testing.T) {
	attendus := map[TargetDB]bool{
		TargetShared: false, TargetPlayer: false, TargetSharedPvE: false,
		TargetSharedSocial: false, TargetMetadata: false,
	}
	for _, m := range All() {
		if _, ok := attendus[m.TargetDB]; ok && m.Name == arbitrationClocksStepName(m.TargetDB) {
			attendus[m.TargetDB] = true
		}
	}
	for target, trouve := range attendus {
		if !trouve {
			t.Errorf("step arbitration clocks absent du registre pour la target %q", target)
		}
	}
}

// TestArbitrationClocksUTC_NomDistinctDuStepS2 : le step JUMEAU doit porter un nom NEUF.
// C'est toute la raison de son existence : le runner saute tout step déjà inscrit au
// ledger (`runSteps`), donc élargir le prédicat de `written_at_default_utc_*` — déjà
// appliqué en prod — n'aurait rejoué sur AUCUNE des bases porteuses du défaut.
func TestArbitrationClocksUTC_NomDistinctDuStepS2(t *testing.T) {
	for _, target := range []TargetDB{
		TargetShared, TargetPlayer, TargetSharedPvE, TargetSharedSocial, TargetMetadata,
	} {
		if arbitrationClocksStepName(target) == writtenAtUTCStepName(target) {
			t.Fatalf("target %q : le step S6 porte le nom du step S2 — il ne rejouerait jamais", target)
		}
	}
}

// TestArbitrationClocksUTC_ReparApresRunForDB : bout en bout par le runner — une base
// legacy passée au runner ressort avec des DEFAULT en UTC (c'est le chemin du boot).
func TestArbitrationClocksUTC_ReparApresRunForDB(t *testing.T) {
	db := ouvrirBaseLegacyWrittenAt(t, `
		CREATE TABLE t_legacy (k INTEGER, computed_at TIMESTAMP NOT NULL DEFAULT now());
	`)
	if err := RunSteps(db, TargetPlayer, []Migration{{
		Name:        arbitrationClocksStepName(TargetPlayer),
		TargetDB:    TargetPlayer,
		Description: "horodatages UTC",
		ApplySchema: func(db *sql.DB) error { _, err := EnsureTimestampDefaultsUTC(db); return err },
	}}); err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if got := ecartColonneAuTempsUTC(t, db, "t_legacy", "computed_at"); got != 0 {
		t.Errorf("après RunSteps: écart attendu 0, obtenu %+d min", got)
	}
	// Et le step est tracé : un second boot ne le rejoue pas.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`,
		arbitrationClocksStepName(TargetPlayer)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("schema_migrations: want 1 ligne pour le step, got %d", n)
	}
}
