//go:build integration

// Package duckdb — career_progression_fresh_migrations_test.go : GARDE-RAIL de la
// promesse « une player DB reconstruite par la seule chaîne de migrations sert le
// chemin de persistance carrière ».
//
// Trou de couverture comblé (bug prod constaté le 2026-08-05) : tous les tests de
// career_progression partaient d'un schéma seedé À LA MAIN (seedPlayerSchema) ou du
// schéma de sync.EnsurePlayerSchema — jamais de celui que produit la chaîne de
// migrations. La dérive a donc pu s'installer sans qu'aucun test ne rougisse :
// rebuild_career_progression_defeat_art_corruption (position 109 de canonicalOrder)
// reconstruisait la table depuis un cliché figé du schéma de mai 2026, effaçant
// last_fetch_status (ajouté par la baseline, position 66) et rétablissant le
// DEFAULT 0 sur xp_total que fix_career_xp_total_default_zero (position 83) venait
// de retirer. Les DB de prod portaient la sentinelle sync_meta du rebuild (donc ne
// rejouaient jamais le swap) et étaient re-soignées par l'ALTER IF NOT EXISTS de
// sync.EnsurePlayerSchema : seules les DB FRAÎCHES/reconstruites étaient cassées.
//
// Ces tests partent donc du chemin RÉEL du runner (migration.RunForDB) — aucun DDL
// écrit à la main ici, sinon le garde-rail se mentirait à lui-même.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// newFreshMigratedPlayerDB construit une player DB :memory: VIERGE provisionnée par
// la seule chaîne de migrations (registre global + steps title-owned câblés par le
// TestMain du package). Aucun seed manuel : c'est l'objet du garde-rail.
func newFreshMigratedPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb :memory:: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := migration.RunForDB(sqlDB, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
	return &PlayerDB{
		Player:    newTestDB(sqlDB, ":memory:"),
		XUID:      pTestXUID,
		Gamertag:  pTestGamertag,
		TitleSlug: titlepkg.DefaultSlug,
	}
}

// careerColumnDefaults introspecte le schéma RÉEL de career_progression.
func careerColumnDefaults(t *testing.T, pdb *PlayerDB) map[string]string {
	t.Helper()
	rows, err := pdb.Player.Query(context.Background(), `
		SELECT column_name, COALESCE(column_default, '')
		FROM duckdb_columns()
		WHERE table_name = 'career_progression'`)
	if err != nil {
		t.Fatalf("introspection career_progression: %v", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			t.Fatalf("scan colonne: %v", err)
		}
		out[name] = def
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("career_progression absente de la DB migrée")
	}
	return out
}

// fullCareerPartial : un partial dont TOUS les champs sont set — il énumère donc
// l'intégralité des colonnes que le chemin career_live sait écrire.
func fullCareerPartial() *CareerProgressionPartial {
	return &CareerProgressionPartial{
		Rank:             partialPtr(42),
		CurrentXP:        partialPtr(1234),
		XPForNextRank:    partialPtr(5678),
		XPTotal:          partialPtr(910111),
		IsMaxRank:        partialPtr(false),
		RankName:         partialPtr("Sergent"),
		RankTier:         partialPtr("Bronze"),
		SpartanID:        partialPtr("SPARTAN-117"),
		BannerImageURL:   partialPtr("https://cdn/banner.png"),
		EmblemImageURL:   partialPtr("https://cdn/emblem.png"),
		BackdropImageURL: partialPtr("https://cdn/backdrop.png"),
		AdornmentPath:    partialPtr("Progression/adornment.png"),
		LastFetchStatus:  partialPtr("ok"),
	}
}

// TestFreshMigratedPlayerDB_CareerProgressionSchema : le schéma produit par la chaîne
// de migrations doit couvrir TOUTES les colonnes que l'INSERT partiel référence.
// Rouge si un step postérieur au créateur de la table en efface une (dérive 2026-08-05)
// ou si un nouveau champ du partial arrive sans son step de migration.
func TestFreshMigratedPlayerDB_CareerProgressionSchema(t *testing.T) {
	pdb := newFreshMigratedPlayerDB(t)
	cols := careerColumnDefaults(t, pdb)

	insertCols, _, _ := buildPartialInsertColumns(pTestXUID, fullCareerPartial())
	for _, c := range insertCols {
		if _, ok := cols[c]; !ok {
			t.Errorf("colonne %q écrite par InsertCareerProgressionPartial mais ABSENTE du schéma produit par les migrations", c)
		}
	}

	// fix_career_xp_total_default_zero (position 83 de canonicalOrder) retire le
	// DEFAULT 0 de xp_total : un xp_total à 0 est une donnée corrompue, pas un zéro
	// légitime. Un step postérieur ne doit pas le rétablir.
	if def := cols["xp_total"]; def != "" {
		t.Errorf("career_progression.xp_total porte un DEFAULT %q sur DB fraîche — fix_career_xp_total_default_zero a été défait par un step postérieur", def)
	}
}

// TestFreshMigratedPlayerDB_CareerLivePersistPath : le chemin career_live complet
// (INSERT partiel -> relecture rang -> relecture statut de fetch) sur une DB
// FRAÎCHE issue des migrations. C'est le test qui rougissait en prod avec
// « Binder Error: Table "career_progression" does not have a column with name
// "last_fetch_status" ».
func TestFreshMigratedPlayerDB_CareerLivePersistPath(t *testing.T) {
	pdb := newFreshMigratedPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()

	inserted, err := repo.InsertCareerProgressionPartial(ctx, pTestXUID, fullCareerPartial())
	if err != nil {
		t.Fatalf("InsertCareerProgressionPartial sur DB fraîche migrée: %v", err)
	}
	if !inserted {
		t.Fatal("partial complet: aucune ligne insérée")
	}

	last, err := repo.LoadLastCareerRank(ctx, pTestXUID)
	if err != nil {
		t.Fatalf("LoadLastCareerRank: %v", err)
	}
	if last == nil {
		t.Fatal("LoadLastCareerRank: aucune ligne relue après INSERT")
	}
	if last.Rank != 42 || last.BannerImageURL != "https://cdn/banner.png" {
		t.Errorf("relecture incohérente: rank=%d banner=%q", last.Rank, last.BannerImageURL)
	}

	status, err := repo.LoadLastFetchStatus(ctx, pTestXUID)
	if err != nil {
		t.Fatalf("LoadLastFetchStatus: %v", err)
	}
	if status != "ok" {
		t.Errorf("LoadLastFetchStatus = %q, attendu \"ok\"", status)
	}

	// Tentative infructueuse tracée seule (Phase 6 PLAN_V2) : status sans data.
	statusOnly := &CareerProgressionPartial{LastFetchStatus: partialPtr("forbidden_403")}
	inserted, err = repo.InsertCareerProgressionPartial(ctx, pTestXUID, statusOnly)
	if err != nil {
		t.Fatalf("InsertCareerProgressionPartial (status seul): %v", err)
	}
	if !inserted {
		t.Error("partial porteur d'un seul status: ligne attendue (traçage des échecs de fetch)")
	}
}
