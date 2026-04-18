//go:build integration

package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/metadata"
)

// ── MetadataRepo ─────────────────────────────────────────────────────────────

func TestMetadataRepo_EnsureSeasonTables(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)

	if err := repo.EnsureSeasonTables(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Idempotent
	if err := repo.EnsureSeasonTables(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataRepo_UpsertAndGetSeason(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	if err := repo.EnsureSeasonTables(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	s := domain.SeasonCalendar{
		TitleID:     "hi",
		SeasonID:    "s1",
		Version:     "1.0",
		Name:        "Season 1",
		StartDate:   now.Add(-30 * 24 * time.Hour),
		FetchedAt:   now,
		ContentHash: "abc123",
	}
	if err := repo.UpsertSeason(ctx, s); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetCurrentSeason(ctx, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if got.SeasonID != "s1" {
		t.Fatalf("expected s1, got %s", got.SeasonID)
	}
}

func TestMetadataRepo_GetCurrentSeason_NotFound(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	if err := repo.EnsureSeasonTables(ctx); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetCurrentSeason(ctx, "nonexistent")
	if err != nil && got != nil {
		t.Fatalf("expected nil or error, got %v %v", got, err)
	}
}

func TestMetadataRepo_UpsertCSRSeason(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	if err := repo.EnsureSeasonTables(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	s := domain.CSRSeasonCalendar{
		TitleID:     "hi",
		SeasonID:    "csr1",
		Version:     "1.0",
		Name:        "CSR Season 1",
		StartDate:   now.Add(-10 * 24 * time.Hour),
		FetchedAt:   now,
		ContentHash: "xyz",
	}
	if err := repo.UpsertCSRSeason(ctx, s); err != nil {
		t.Fatal(err)
	}

	seasons, err := repo.GetCSRSeasons(ctx, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if len(seasons) != 1 {
		t.Fatalf("expected 1, got %d", len(seasons))
	}
	if seasons[0].SeasonID != "csr1" {
		t.Fatalf("expected csr1, got %s", seasons[0].SeasonID)
	}
}

func TestMetadataRepo_GetSeasonByDate(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	if err := repo.EnsureSeasonTables(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	end := now.Add(30 * 24 * time.Hour)
	s := domain.SeasonCalendar{
		TitleID:   "hi",
		SeasonID:  "s1",
		Name:      "Active Season",
		StartDate: now.Add(-10 * 24 * time.Hour),
		EndDate:   &end,
		FetchedAt: now,
	}
	if err := repo.UpsertSeason(ctx, s); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetSeasonByDate(ctx, "hi", now.Format("2006-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.SeasonID != "s1" {
		t.Fatalf("expected s1, got %v", got)
	}
}

// ── MedalImageCache ──────────────────────────────────────────────────────────

func TestMetadataRepo_EnsureMedalImageCacheTable(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	if err := repo.EnsureMedalImageCacheTable(ctx); err != nil {
		t.Fatal(err)
	}
	// Idempotent
	if err := repo.EnsureMedalImageCacheTable(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataRepo_UpsertAndGetMedalImageCache(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	if err := repo.EnsureMedalImageCacheTable(ctx); err != nil {
		t.Fatal(err)
	}

	e := MedalImageEntry{
		TitleID:     "hi",
		MedalID:     42,
		ImageURL:    "https://example.com/medal.png",
		LocalPath:   "medals/42.png",
		FetchedAt:   time.Now().UTC().Truncate(time.Second),
		ContentHash: "sha256abc",
	}
	if err := repo.UpsertMedalImageCache(ctx, e); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetMedalImageCache(ctx, "hi", 42)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.ImageURL != e.ImageURL {
		t.Fatalf("expected %s, got %s", e.ImageURL, got.ImageURL)
	}
}

func TestMetadataRepo_GetMedalImageCache_NotFound(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	if err := repo.EnsureMedalImageCacheTable(ctx); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetMedalImageCache(ctx, "hi", 999)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil for missing entry")
	}
}

func TestMetadataRepo_UpsertMedalsRaw(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	// Create table
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS waypoint_medals_raw (
			title_id       VARCHAR NOT NULL,
			medal_id       BIGINT NOT NULL,
			name_id        VARCHAR NOT NULL DEFAULT '',
			description_id VARCHAR NOT NULL DEFAULT '',
			sprite_index   INTEGER NOT NULL DEFAULT 0,
			difficulty     VARCHAR NOT NULL DEFAULT '',
			medal_type     VARCHAR NOT NULL DEFAULT '',
			personal_score INTEGER NOT NULL DEFAULT 0,
			raw_json       VARCHAR NOT NULL DEFAULT '',
			fetched_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash   VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, medal_id)
		)`)
	if err != nil {
		t.Fatal(err)
	}

	entries := []metadata.MedalEntry{
		{TitleID: "hi", MedalID: 1, Label: "Kill", Description: "Get a kill", Category: "combat", Rarity: "common"},
		{TitleID: "hi", MedalID: 2, Label: "Double", Description: "Double kill", Category: "multi", Rarity: "rare"},
	}
	if err := repo.UpsertMedalsRaw(ctx, entries, "hash1"); err != nil {
		t.Fatal(err)
	}

	n, err := repo.CountMedalsRaw(ctx, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}
}

// ── AssetCacheRepo ───────────────────────────────────────────────────────────

func TestMetadataRepo_AssetCacheRoundTrip(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	// Create table
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS waypoint_assets_raw (
			title_id     VARCHAR NOT NULL,
			asset_id     VARCHAR NOT NULL,
			asset_type   VARCHAR NOT NULL DEFAULT '',
			version_id   VARCHAR NOT NULL DEFAULT '',
			name         VARCHAR NOT NULL DEFAULT '',
			description  VARCHAR NOT NULL DEFAULT '',
			raw_json     VARCHAR NOT NULL DEFAULT '',
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			content_hash VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (title_id, asset_id, version_id)
		)`)
	if err != nil {
		t.Fatal(err)
	}

	e := AssetEntry{
		TitleID:     "hi",
		AssetID:     "map_001",
		AssetType:   "map",
		VersionID:   "v1",
		Name:        "Bazaar",
		Description: "Marketplace map",
		RawJSON:     `{"id":"map_001"}`,
		FetchedAt:   time.Now().UTC().Truncate(time.Second),
		ContentHash: "abc",
	}
	if err := repo.UpsertAsset(ctx, e); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetAssetByID(ctx, "hi", "map_001")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "Bazaar" {
		t.Fatalf("expected Bazaar, got %v", got)
	}

	list, err := repo.ListAssets(ctx, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(list))
	}
}

func TestMetadataRepo_GetAssetByID_NotFound(t *testing.T) {
	db := openMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()

	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS waypoint_assets_raw (
			title_id VARCHAR NOT NULL, asset_id VARCHAR NOT NULL,
			asset_type VARCHAR DEFAULT '', version_id VARCHAR DEFAULT '',
			name VARCHAR DEFAULT '', description VARCHAR DEFAULT '',
			raw_json VARCHAR DEFAULT '', fetched_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			content_hash VARCHAR DEFAULT '',
			PRIMARY KEY (title_id, asset_id, version_id)
		)`)
	if err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetAssetByID(ctx, "hi", "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil for missing asset")
	}
}
