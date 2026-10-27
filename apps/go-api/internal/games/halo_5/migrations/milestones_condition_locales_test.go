package migrations

// milestones_condition_locales_test.go — G2 (2026-07-26), repris pour les tables par langue
// (ADR 0040 D-5, plan i18n 8.3).
//
// Historique : le CREATE h5 de milestone_catalog omettait condition_fr / condition_en alors que
// le lecteur unique les SELECTionnait (Binder Error → GET /milestones 500). Depuis que les textes
// des jalons vivent dans asset_translations (type `milestone`), le lecteur ne lit plus ces
// colonnes ; ces tests tiennent la chaîne de migration : une metadata h5 déjà provisionnée garde
// son catalogue et ses titres, une metadata vierge est seedée dans chaque langue.
//
// Piège couvert : `CREATE TABLE IF NOT EXISTS` n'ajoute PAS de colonne à une table existante, et
// un step déjà tracé dans schema_migrations n'est jamais rejoué.

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
)

// openTempMetadataDB ouvre une metadata.duckdb sur fichier temporaire (le repo
// MilestoneCatalogRepo consomme un *duckdb.DB, non constructible en :memory:
// hors du package duckdb).
func openTempMetadataDB(t *testing.T) *duckdb.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedLegacyMilestoneCatalog reproduit l'état d'une metadata.duckdb halo_5 DÉJÀ
// provisionnée en v7.2.5 : table milestone_catalog SANS condition_fr/condition_en,
// avec des lignes seedées, et les steps correspondants déjà tracés (donc jamais
// rejoués par le runner).
func seedLegacyMilestoneCatalog(t *testing.T, db *duckdb.DB) {
	t.Helper()
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS milestone_catalog (
			id          VARCHAR PRIMARY KEY,
			title_slug  VARCHAR NOT NULL,
			metric      VARCHAR NOT NULL,
			threshold   DOUBLE NOT NULL,
			title_en    VARCHAR NOT NULL,
			title_fr    VARCHAR NOT NULL,
			icon        VARCHAR,
			condition   VARCHAR,
			updated_at  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`INSERT INTO milestone_catalog (id, title_slug, metric, threshold, title_en, title_fr)
		 VALUES ('halo_5.matches.100', 'halo_5', 'matches_played', 100, 'Centurion', 'Centurion')`,
		`INSERT INTO milestone_catalog (id, title_slug, metric, threshold, title_en, title_fr)
		 VALUES ('halo_5.wins.50', 'halo_5', 'wins', 50, 'Winner', 'Vainqueur')`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			name          VARCHAR PRIMARY KEY,
			description   VARCHAR,
			applied_at    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			schema_done   BOOLEAN DEFAULT FALSE,
			backfill_done BOOLEAN DEFAULT FALSE,
			title_slug    VARCHAR DEFAULT 'halo_infinite'
		)`,
		`INSERT INTO schema_migrations (name, description, schema_done, backfill_done, title_slug)
		 VALUES ('h5_create_milestone_catalog', 'legacy', TRUE, TRUE, 'halo_5')`,
		`INSERT INTO schema_migrations (name, description, schema_done, backfill_done, title_slug)
		 VALUES ('h5_seed_milestone_catalog', 'legacy', TRUE, TRUE, 'halo_5')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(ctx, s); err != nil {
			t.Fatalf("seed legacy milestone_catalog: %v\nSQL: %s", err, s)
		}
	}
}

// configTitlesRootForH5 retourne config/titles/ depuis
// apps/go-api/internal/games/halo_5/migrations (6 niveaux jusqu'à la racine).
func configTitlesRootForH5(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller indisponible")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "..", "config", "titles")
}

// TestHalo5MilestoneCatalog_LegacyDB_ConditionLocalesAdded : une metadata.duckdb h5 EXISTANTE (titres
// en colonnes appariées, sans condition_fr/condition_en) est migrée sans perdre son catalogue :
// les titres passent dans asset_translations (type `milestone`), les colonnes appariées sont
// retirées, et le lecteur unique sert le titre dans chaque langue.
func TestHalo5MilestoneCatalog_LegacyDB_ConditionLocalesAdded(t *testing.T) {
	db := openTempMetadataDB(t)
	seedLegacyMilestoneCatalog(t, db)

	ctx := context.Background()
	repo := duckdb.NewMilestoneCatalogRepo(db)

	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()
	if err := migration.RunForTitleDB(db.SQLDb(), halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB(%s, metadata): %v", halo5.TitleSlug, err)
	}

	entries, err := repo.ListByTitle(ctx, halo5.TitleSlug)
	if err != nil {
		t.Fatalf("ListByTitle après migration: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("catalogue attendu non vide (>= 2 lignes préexistantes), obtenu %d", len(entries))
	}
	byID := map[string]milestones.CatalogEntry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	wins := byID["halo_5.wins.50"]
	if wins.Title.Pick(i18n.LocaleEN) != "Winner" || wins.Title.Pick(i18n.LocaleFR) != "Vainqueur" {
		t.Errorf("titres de halo_5.wins.50 perdus à la migration : %v", wins.Title)
	}
	for _, col := range []string{"title_en", "title_fr", "condition_fr", "condition_en"} {
		var n int
		if err := db.QueryRow(ctx,
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'milestone_catalog' AND column_name = ?`,
			col).Scan(&n); err != nil {
			t.Fatalf("information_schema: %v", err)
		}
		if n != 0 {
			t.Errorf("milestone_catalog garde la colonne %s après la migration des textes", col)
		}
	}
}

// TestHalo5MilestoneCatalog_FreshDB_ReadableAndSeeded : sur une DB VIERGE, le catalogue seedé depuis
// le TOML (titre et condition lisible écrits dans asset_translations, le seed passant APRÈS le step
// qui retire les colonnes appariées) est lisible par le repo, dans chaque langue.
func TestHalo5MilestoneCatalog_FreshDB_ReadableAndSeeded(t *testing.T) {
	db := openTempMetadataDB(t)

	SetMilestonesSeedRoot(configTitlesRootForH5(t))
	t.Cleanup(func() { SetMilestonesSeedRoot("") })

	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()
	if err := migration.RunForTitleDB(db.SQLDb(), halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB(%s, metadata): %v", halo5.TitleSlug, err)
	}

	entries, err := duckdb.NewMilestoneCatalogRepo(db).ListByTitle(context.Background(), halo5.TitleSlug)
	if err != nil {
		t.Fatalf("ListByTitle sur DB vierge migrée: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("catalogue halo_5 VIDE — le seed TOML n'a pas été appliqué ou n'est pas lisible")
	}
	for _, e := range entries {
		if e.TitleSlug != halo5.TitleSlug {
			t.Errorf("entrée %s de title_slug %q — fuite cross-titre", e.ID, e.TitleSlug)
		}
		for _, l := range []i18n.Locale{i18n.LocaleFR, i18n.LocaleEN} {
			if e.Title.Pick(l) == "" {
				t.Errorf("entrée %s : titre %s vide après le seed", e.ID, l)
			}
		}
	}
}
