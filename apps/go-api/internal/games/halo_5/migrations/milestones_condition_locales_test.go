package migrations

// milestones_condition_locales_test.go — G2 (2026-07-26).
//
// Régression : le CREATE h5 de milestone_catalog omettait condition_fr /
// condition_en alors que le lecteur unique (duckdb.MilestoneCatalogRepo) les
// SELECTionne toujours → Binder Error → GET /milestones 500 sur halo_5 → grille
// Réalisations vide. Halo Infinite recevait ces colonnes par SA migration ; le set
// h5 possède son propre TargetMetadata et ne les recevait jamais.
//
// Piège couvert ici : `CREATE TABLE IF NOT EXISTS` n'ajoute PAS de colonne à une
// table existante, et un step déjà tracé dans schema_migrations n'est jamais
// rejoué → corriger le CREATE ne répare PAS les DB démo/prod déjà provisionnées.
// Seul un step ALTER additif au NOUVEAU nom le fait.

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
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
			updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO milestone_catalog (id, title_slug, metric, threshold, title_en, title_fr)
		 VALUES ('halo_5.matches.100', 'halo_5', 'matches_played', 100, 'Centurion', 'Centurion')`,
		`INSERT INTO milestone_catalog (id, title_slug, metric, threshold, title_en, title_fr)
		 VALUES ('halo_5.wins.50', 'halo_5', 'wins', 50, 'Winner', 'Vainqueur')`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			name          VARCHAR PRIMARY KEY,
			description   VARCHAR,
			applied_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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

// TestHalo5MilestoneCatalog_LegacyDB_ConditionLocalesAdded : une metadata.duckdb
// h5 EXISTANTE sans condition_fr/condition_en fait échouer le repo (Binder Error)
// AVANT migration, et le sert sans erreur APRÈS — sans perdre son catalogue.
func TestHalo5MilestoneCatalog_LegacyDB_ConditionLocalesAdded(t *testing.T) {
	db := openTempMetadataDB(t)
	seedLegacyMilestoneCatalog(t, db)

	ctx := context.Background()
	repo := duckdb.NewMilestoneCatalogRepo(db)

	// Contrôle : sans les colonnes, le lecteur DOIT échouer (c'est le 500 observé).
	if _, err := repo.ListByTitle(ctx, halo5.TitleSlug); err == nil {
		t.Fatal("ListByTitle attendu en échec sur une DB legacy sans condition_fr/condition_en " +
			"— le test ne reproduit plus la régression")
	}

	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()
	if err := migration.RunForTitleDB(db.SQLDb(), halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB(%s, metadata): %v", halo5.TitleSlug, err)
	}

	entries, err := repo.ListByTitle(ctx, halo5.TitleSlug)
	if err != nil {
		t.Fatalf("ListByTitle après migration (Binder Error non réparé): %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("catalogue attendu non vide (>= 2 lignes préexistantes), obtenu %d", len(entries))
	}
}

// TestHalo5MilestoneCatalog_FreshDB_ReadableAndSeeded : sur une DB VIERGE, le
// CREATE porte les colonnes dès la création et le catalogue seedé depuis le TOML
// est lisible par le repo (chemin nominal d'un nouveau provisioning).
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
	}
}
