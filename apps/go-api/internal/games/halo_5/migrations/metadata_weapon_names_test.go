package migrations

// metadata_weapon_names_test.go — noms d'armes par langue (ADR 0040 D-5, plan i18n 8.4) sur la
// metadata Halo 5 : une base neuve naît dans la forme finale, une base à l'ancienne forme (colonnes
// name_en / name_fr) est portée sans perte, et le step est le DERNIER du jeu h5.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func columnPresent(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?", table, column,
	).Scan(&n); err != nil {
		t.Fatalf("columnPresent(%s.%s): %v", table, column, err)
	}
	return n == 1
}

func TestHalo5Metadata_WeaponNamesStepEstLeDernier(t *testing.T) {
	names := metadataStepNames()
	if got := names[len(names)-1]; got != migration.WeaponNamesStepName {
		t.Errorf("dernier step du jeu h5 = %q, want %q", got, migration.WeaponNamesStepName)
	}
	steps := MetadataSteps()
	if got := steps[len(steps)-1].Name; got != migration.WeaponNamesStepName {
		t.Errorf("dernier step de MetadataSteps = %q, want %q", got, migration.WeaponNamesStepName)
	}
}

func TestHalo5Metadata_WeaponNamesBaseNeuve(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB: %v", err)
	}
	for _, c := range []struct {
		table, column string
		want          bool
	}{
		{"weapon_labels", "name_en", false}, {"weapon_labels", "name_fr", false},
		{"weapon_labels", "icon_url", true}, {"weapon_labels", "weapon_type", true},
		{"weapon_label_names", "lang", true}, {"weapon_label_names", "name", true},
		{"weapon_name_labels", "lang", true}, {"weapon_name_labels", "name_en", false},
	} {
		if got := columnPresent(t, db, c.table, c.column); got != c.want {
			t.Errorf("%s.%s présente = %v, want %v", c.table, c.column, got, c.want)
		}
	}
}

func TestHalo5Metadata_WeaponNamesBaseALAncienneForme(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, q := range []string{
		`CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL,
			icon_url VARCHAR, weapon_type VARCHAR)`,
		`INSERT INTO weapon_labels VALUES (2511447508, 'Light Rifle', 'Fusil léger', 'https://cdn.example/lr.png', 'Rifle'),
			(4106030681, 'FRAG GRENADE', 'FRAG GRENADE', NULL, NULL)`,
		`CREATE TABLE weapon_name_labels (title_slug VARCHAR NOT NULL, weapon_key VARCHAR NOT NULL,
			name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL, PRIMARY KEY (title_slug, weapon_key))`,
		`INSERT INTO weapon_name_labels VALUES ('halo_5', 'h5_light_rifle', 'Light Rifle', 'Fusil léger')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB: %v", err)
	}
	var fr, icon string
	if err := db.QueryRow(`SELECT name FROM weapon_label_names WHERE weapon_id = 2511447508 AND lang = 'fr'`).Scan(&fr); err != nil || fr != "Fusil léger" {
		t.Errorf("nom fr migré = %q (%v)", fr, err)
	}
	if err := db.QueryRow(`SELECT icon_url FROM weapon_labels WHERE weapon_id = 2511447508`).Scan(&icon); err != nil || icon != "https://cdn.example/lr.png" {
		t.Errorf("icône conservée = %q (%v)", icon, err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM weapon_label_names`).Scan(&n); err != nil || n != 4 {
		t.Errorf("weapon_label_names = %d lignes (%v), want 4", n, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM weapon_name_labels WHERE lang IN ('fr','en')`).Scan(&n); err != nil || n != 2 {
		t.Errorf("weapon_name_labels = %d lignes (%v), want 2", n, err)
	}
	if columnPresent(t, db, "weapon_labels", "name_en") || columnPresent(t, db, "weapon_name_labels", "name_fr") {
		t.Error("colonnes de nom encore présentes après la migration")
	}
}
