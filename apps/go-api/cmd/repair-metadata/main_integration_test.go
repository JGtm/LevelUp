//go:build integration

package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "metadata.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, context.Background()
}

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

func scalar[T any](t *testing.T, db *sql.DB, q string) T {
	t.Helper()
	var v T
	if err := db.QueryRow(q).Scan(&v); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
	return v
}

// metadataAvantLeLot construit les trois tables dans la forme d'AVANT le lot i18n (colonnes de texte
// label_* / title_* / description_* encore présentes), plus les colonnes ajoutées depuis (source, tagging) :
// la définition que l'outil écrivait en dur ne correspondait à aucune de ces deux formes.
func metadataAvantLeLot(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE challenge_template (
		id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL, metric VARCHAR NOT NULL,
		window_type VARCHAR NOT NULL, window_value VARCHAR, cadence VARCHAR NOT NULL,
		eval_type VARCHAR NOT NULL, mode_filter VARCHAR NOT NULL DEFAULT 'universal',
		label_en VARCHAR NOT NULL, label_fr VARCHAR NOT NULL, description_en VARCHAR, description_fr VARCHAR,
		normal_target DOUBLE NOT NULL, heroic_target DOUBLE NOT NULL, legendary_target DOUBLE NOT NULL,
		mythic_target DOUBLE NOT NULL, schema_version INTEGER NOT NULL DEFAULT 1,
		updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		source VARCHAR DEFAULT 'catalog', lusr_components VARCHAR, radar_axes VARCHAR, is_long_term BOOLEAN DEFAULT FALSE)`)
	mustExec(t, db, `CREATE INDEX idx_ctmpl_metric ON challenge_template(metric)`)
	mustExec(t, db, `INSERT INTO challenge_template (id, title_slug, metric, window_type, cadence, eval_type,
		label_en, label_fr, description_en, description_fr, normal_target, heroic_target, legendary_target,
		mythic_target, source, lusr_components, radar_axes, is_long_term) VALUES
		('t1','halo_infinite','kills','matches','daily','sum','Kills','Frags','Get kills','Fais des frags',1,2,3,4,'catalog','a,b','slaying',TRUE),
		('t2','halo_infinite','wins','matches','weekly','sum','Wins','Victoires',NULL,NULL,5,6,7,8,'coach_synthesized',NULL,NULL,FALSE)`)

	mustExec(t, db, `CREATE TABLE preset_arc (
		id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL, title_en VARCHAR NOT NULL, title_fr VARCHAR NOT NULL,
		description_en VARCHAR, description_fr VARCHAR, schema_version INTEGER NOT NULL DEFAULT 1,
		updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`)
	mustExec(t, db, `INSERT INTO preset_arc (id, title_slug, title_en, title_fr, description_en, description_fr)
		VALUES ('a1','halo_infinite','Arc one','Arc un','Desc','Description')`)

	mustExec(t, db, `CREATE TABLE preset_arc_step (
		preset_arc_id VARCHAR NOT NULL, position INTEGER NOT NULL, template_id VARCHAR NOT NULL,
		target_tier VARCHAR NOT NULL, PRIMARY KEY (preset_arc_id, position))`)
	mustExec(t, db, `INSERT INTO preset_arc_step VALUES ('a1',1,'t1','normal'),('a1',2,'t2','heroic')`)
}

// TestRepairAll_BaseDAvantLeLot : sur une base d'avant le lot (colonnes de texte encore présentes), aucune
// ligne ni aucune colonne n'est perdue, la clé primaire et l'index existants sont recréés, et aucune table
// temporaire ne subsiste.
func TestRepairAll_BaseDAvantLeLot(t *testing.T) {
	db, ctx := openTestDB(t)
	metadataAvantLeLot(t, db)

	if err := repairAll(ctx, db); err != nil {
		t.Fatalf("repairAll: %v", err)
	}

	for table, want := range map[string]int{"challenge_template": 2, "preset_arc": 1, "preset_arc_step": 2} {
		if got := scalar[int](t, db, `SELECT COUNT(*) FROM `+table); got != want {
			t.Errorf("%s : %d lignes, attendu %d", table, got, want)
		}
		if n := scalar[int](t, db, `SELECT COUNT(*) FROM duckdb_tables() WHERE table_name = '`+table+`_repair_tmp'`); n != 0 {
			t.Errorf("%s : la table temporaire subsiste", table)
		}
	}
	// Les colonnes de texte d'avant le lot et celles ajoutées depuis gardent leurs valeurs.
	if got := scalar[string](t, db, `SELECT label_fr FROM challenge_template WHERE id = 't1'`); got != "Frags" {
		t.Errorf("label_fr = %q, attendu Frags", got)
	}
	if got := scalar[string](t, db, `SELECT lusr_components FROM challenge_template WHERE id = 't1'`); got != "a,b" {
		t.Errorf("lusr_components = %q, attendu a,b", got)
	}
	if got := scalar[string](t, db, `SELECT source FROM challenge_template WHERE id = 't2'`); got != "coach_synthesized" {
		t.Errorf("source = %q, attendu coach_synthesized", got)
	}
	if got := scalar[string](t, db, `SELECT title_fr FROM preset_arc WHERE id = 'a1'`); got != "Arc un" {
		t.Errorf("title_fr = %q, attendu Arc un", got)
	}
	// Clé primaire et index recréés.
	if _, err := db.Exec(`INSERT INTO challenge_template (id, title_slug, metric, window_type, cadence, eval_type,
		label_en, label_fr, normal_target, heroic_target, legendary_target, mythic_target)
		VALUES ('t1','x','m','w','c','e','l','l',1,1,1,1)`); err == nil {
		t.Error("la clé primaire de challenge_template doit être recréée (insertion d'un doublon acceptée)")
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = 'idx_ctmpl_metric'`); n != 1 {
		t.Errorf("l'index idx_ctmpl_metric doit être recréé, %d trouvé", n)
	}
}

// TestRepairAll_BaseDApresLeLot : la même réparation sur la forme actuelle (sans colonnes de texte) fonctionne.
func TestRepairAll_BaseDApresLeLot(t *testing.T) {
	db, ctx := openTestDB(t)
	mustExec(t, db, `CREATE TABLE challenge_template (
		id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL, metric VARCHAR NOT NULL, window_type VARCHAR NOT NULL,
		window_value VARCHAR, cadence VARCHAR NOT NULL, eval_type VARCHAR NOT NULL,
		mode_filter VARCHAR NOT NULL DEFAULT 'universal', normal_target DOUBLE NOT NULL, heroic_target DOUBLE NOT NULL,
		legendary_target DOUBLE NOT NULL, mythic_target DOUBLE NOT NULL, schema_version INTEGER NOT NULL DEFAULT 1,
		updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), source VARCHAR DEFAULT 'catalog',
		lusr_components VARCHAR, radar_axes VARCHAR, is_long_term BOOLEAN DEFAULT FALSE)`)
	mustExec(t, db, `INSERT INTO challenge_template (id, title_slug, metric, window_type, cadence, eval_type,
		normal_target, heroic_target, legendary_target, mythic_target) VALUES ('t1','s','m','w','c','e',1,2,3,4)`)
	mustExec(t, db, `CREATE TABLE preset_arc (id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL,
		schema_version INTEGER NOT NULL DEFAULT 1, updated_at TIMESTAMP)`)
	mustExec(t, db, `CREATE TABLE preset_arc_step (preset_arc_id VARCHAR NOT NULL, position INTEGER NOT NULL,
		template_id VARCHAR NOT NULL, target_tier VARCHAR NOT NULL, PRIMARY KEY (preset_arc_id, position))`)

	if err := repairAll(ctx, db); err != nil {
		t.Fatalf("repairAll: %v", err)
	}
	if got := scalar[int](t, db, `SELECT COUNT(*) FROM challenge_template`); got != 1 {
		t.Errorf("challenge_template : %d lignes, attendu 1", got)
	}
}

// TestRepairTable_TableAbsente : une table absente est une erreur avant toute écriture.
func TestRepairTable_TableAbsente(t *testing.T) {
	db, ctx := openTestDB(t)
	if err := repairTable(ctx, db, "challenge_template"); err == nil {
		t.Fatal("une table absente doit être refusée")
	}
}

// TestRestoreFromTemp : après une reconstruction échouée, les données reviennent sous le nom de la table et la
// table temporaire disparaît (jamais de données seulement dans la table temporaire).
func TestRestoreFromTemp(t *testing.T) {
	db, ctx := openTestDB(t)
	mustExec(t, db, `CREATE TABLE preset_arc (id VARCHAR PRIMARY KEY)`) // table partielle de la reconstruction
	mustExec(t, db, `CREATE TABLE preset_arc_repair_tmp AS SELECT * FROM (VALUES ('a'), ('b')) v(id)`)

	cause := errors.New("reinsert data: colonnes différentes")
	err := restoreFromTemp(ctx, db, "preset_arc", "preset_arc_repair_tmp", 2, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("l'erreur d'origine doit être portée : %v", err)
	}
	if got := scalar[int](t, db, `SELECT COUNT(*) FROM preset_arc`); got != 2 {
		t.Errorf("preset_arc : %d lignes, attendu 2 (données restaurées)", got)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM duckdb_tables() WHERE table_name = 'preset_arc_repair_tmp'`); n != 0 {
		t.Error("la table temporaire doit avoir été renommée")
	}
}
