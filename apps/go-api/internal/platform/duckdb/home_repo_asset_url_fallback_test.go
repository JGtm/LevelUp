// Package duckdb — home_repo_asset_url_fallback_test.go : tests pour le
// fallback name-based d'image map quand `map_images_registry` est vide
// (Option B 2026-05-08, cf. .ai/PLAN_RECENT_MATCH_REGRESSION_FIX.md).
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// stubAssetURLForHome implémente l'interface homeAssetURLAdapter (duck-typing)
// avec un mapping fixe nom EN → URL. Permet de tester le fallback sans avoir
// à scanner un vrai répertoire static.
type stubAssetURLForHome struct {
	urls map[string]string
}

func (s *stubAssetURLForHome) MapImageURL(name string) string {
	if url, ok := s.urls[name]; ok {
		return url
	}
	return ""
}

// newHomeRepoTestMetaDB crée une metadata DB en-mémoire seedée pour les tests
// d'EnrichCanonicalAssetTranslations (asset_translations + map_images_registry
// + mode_name_tr + structures attendues).
func newHomeRepoTestMetaDB(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open meta: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	db := newTestDB(sqlDB, ":memory:")

	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE asset_translations (
			asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR,
			name VARCHAR, description VARCHAR DEFAULT '',
			fetched_at TIMESTAMPTZ DEFAULT now(),
			PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE map_images_registry (
			title_id VARCHAR, map_id VARCHAR, local_path VARCHAR,
			fetched_at TIMESTAMPTZ DEFAULT now(),
			PRIMARY KEY (title_id, map_id))`,
		`CREATE TABLE mode_name_tr (lang VARCHAR, mode_en VARCHAR, name VARCHAR)`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seed schema: %v\nSQL: %s", err, q)
		}
	}
	return db
}

// TestEnrichCanonicalAssetTranslations_FallbackToAssetURLAdapter reproduit
// exactement le scénario du screenshot 2026-05-08 : map_images_registry vide
// pour Shiro (cmd/migrate-static-maps pas re-runnée), mais Shiro.jpg existe
// en static dir → l'adapter doit résoudre l'URL via le **nom EN** depuis
// asset_translations en-US, sans dépendre du registry.
func TestEnrichCanonicalAssetTranslations_FallbackToAssetURLAdapter(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	ctx := context.Background()

	// Seed asset_translations : Shiro a en-US + fr-FR, mais map_images_registry
	// est volontairement vide (cas user post-2026-05-08 avant CLI).
	if _, err := meta.Exec(ctx,
		`INSERT INTO asset_translations VALUES ('shiro-uuid', 'map', 'fr-FR', 'Shiro', '', now())`,
	); err != nil {
		t.Fatalf("seed FR: %v", err)
	}
	if _, err := meta.Exec(ctx,
		`INSERT INTO asset_translations VALUES ('shiro-uuid', 'map', 'en-US', 'Shiro', '', now())`,
	); err != nil {
		t.Fatalf("seed EN: %v", err)
	}
	// Note : on n'insère RIEN dans map_images_registry — c'est le cas qu'on teste.

	pdb := &PlayerDB{Metadata: meta}
	repo := NewHomeRepo(pdb).WithAssetURL(&stubAssetURLForHome{
		urls: map[string]string{
			"Shiro": "/static/maps/halo_infinite/Shiro.jpg",
		},
	})

	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m1",
				Map: &canonical.AssetReference{
					ID:           "shiro-uuid",
					DefaultLabel: "shiro-uuid", // simulating sync pre-resolution (map_name = UUID)
				},
			},
		},
	}

	if err := repo.EnrichCanonicalAssetTranslations(ctx, rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}

	got := rows[0].Summary.Map
	if got == nil {
		t.Fatal("Summary.Map = nil après enrichment")
	}
	if got.IconURL != "/static/maps/halo_infinite/Shiro.jpg" {
		t.Errorf("IconURL = %q, want /static/maps/halo_infinite/Shiro.jpg (fallback adapter via nom EN)", got.IconURL)
	}
	// Vérifier aussi que les Labels EN/FR sont bien hydratés
	if got.Labels["en"] != "Shiro" {
		t.Errorf("Labels[en] = %q, want Shiro", got.Labels["en"])
	}
	if got.Labels["fr"] != "Shiro" {
		t.Errorf("Labels[fr] = %q, want Shiro", got.Labels["fr"])
	}
}

// TestEnrichCanonicalAssetTranslations_RegistryWinsOverAdapter vérifie que
// quand `map_images_registry` a une entrée, elle est prioritaire sur le
// fallback adapter (le registry est censé être la source canonique
// title-scoped, l'adapter est juste un filet de sécurité).
func TestEnrichCanonicalAssetTranslations_RegistryWinsOverAdapter(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	ctx := context.Background()

	if _, err := meta.Exec(ctx,
		`INSERT INTO asset_translations VALUES ('shiro-uuid', 'map', 'en-US', 'Shiro', '', now())`,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := meta.Exec(ctx,
		`INSERT INTO map_images_registry VALUES ('halo_infinite', 'shiro-uuid', '/registry/Shiro.png', now())`,
	); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	pdb := &PlayerDB{Metadata: meta}
	repo := NewHomeRepo(pdb).WithAssetURL(&stubAssetURLForHome{
		urls: map[string]string{"Shiro": "/adapter/Shiro.jpg"},
	})

	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m1",
				Map:     &canonical.AssetReference{ID: "shiro-uuid"},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(ctx, rows); err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	got := rows[0].Summary.Map.IconURL
	if got != "/registry/Shiro.png" {
		t.Errorf("IconURL = %q, want /registry/Shiro.png (registry doit gagner sur adapter)", got)
	}
}

// TestEnrichCanonicalAssetTranslations_NoAdapterNoRegistry vérifie le
// comportement par défaut (registry vide + adapter pas câblé) : IconURL reste
// vide, dégradation gracieuse — aucune erreur, juste pas d'image. Cas
// d'environnements de test ou de déploiements legacy.
func TestEnrichCanonicalAssetTranslations_NoAdapterNoRegistry(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	ctx := context.Background()

	if _, err := meta.Exec(ctx,
		`INSERT INTO asset_translations VALUES ('shiro-uuid', 'map', 'en-US', 'Shiro', '', now())`,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	pdb := &PlayerDB{Metadata: meta}
	repo := NewHomeRepo(pdb) // ← pas de WithAssetURL

	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m1",
				Map:     &canonical.AssetReference{ID: "shiro-uuid"},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(ctx, rows); err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	if rows[0].Summary.Map.IconURL != "" {
		t.Errorf("IconURL = %q, want \"\" (pas de fallback possible)", rows[0].Summary.Map.IconURL)
	}
}
