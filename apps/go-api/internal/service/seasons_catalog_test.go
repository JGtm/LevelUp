// seasons_catalog_test.go — tests du résolveur unifié TOML + DB + lazy fetch.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
)

// ─── Fakes pour MetadataRepository et SeasonProvider ──────────────────────

type fakeMetadataRepo struct {
	listCalls   int
	upsertCalls int
	listResults [][]domain.SeasonCalendar // résultats successifs (1er appel = listResults[0], etc.)
	listErr     error
	upserted    []domain.SeasonCalendar
	upsertErr   error
}

func (f *fakeMetadataRepo) GetCurrentSeason(_ context.Context, _ string) (*domain.SeasonCalendar, error) {
	return nil, nil
}
func (f *fakeMetadataRepo) ListSeasons(_ context.Context, _ string) ([]domain.SeasonCalendar, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	idx := f.listCalls
	if idx >= len(f.listResults) {
		idx = len(f.listResults) - 1
	}
	f.listCalls++
	if idx < 0 {
		return nil, nil
	}
	return f.listResults[idx], nil
}
func (f *fakeMetadataRepo) GetCSRSeasons(_ context.Context, _ string) ([]domain.CSRSeasonCalendar, error) {
	return nil, nil
}
func (f *fakeMetadataRepo) GetSeasonByDate(_ context.Context, _, _ string) (*domain.SeasonCalendar, error) {
	return nil, nil
}
func (f *fakeMetadataRepo) UpsertSeason(_ context.Context, s domain.SeasonCalendar) error {
	f.upsertCalls++
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserted = append(f.upserted, s)
	return nil
}
func (f *fakeMetadataRepo) UpsertCSRSeason(_ context.Context, _ domain.CSRSeasonCalendar) error {
	return nil
}
func (f *fakeMetadataRepo) UpsertSnapshot(_ context.Context, _ domain.WaypointResourceSnapshot) error {
	return nil
}
func (f *fakeMetadataRepo) GetSnapshot(_ context.Context, _, _ string) (*domain.WaypointResourceSnapshot, error) {
	return nil, nil
}

type fakeSeasonProvider struct {
	fetchCalls   int
	fetchResults []domain.SeasonCalendar
	fetchErr     error
}

func (f *fakeSeasonProvider) FetchSeasonCalendar(_ context.Context, _ string) ([]domain.SeasonCalendar, []byte, error) {
	f.fetchCalls++
	if f.fetchErr != nil {
		return nil, nil, f.fetchErr
	}
	return f.fetchResults, nil, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func makeAssetSetWithSeasons(t *testing.T) *mappings.AssetMappingSet {
	t.Helper()
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.season.season1]
labels = { en = "Heroes of Reach", fr = "Heroes of Reach" }
display_order = 10
start_date = "2021-12-08T00:00:00Z"
end_date   = "2022-05-03T00:00:00Z"
extra = { csr_season_id = "CsrSeason1", short_label = "S1" }

[assets.season.season2]
labels = { en = "Lone Wolves", fr = "Loups solitaires" }
display_order = 20
start_date = "2022-05-03T00:00:00Z"
end_date   = "2022-11-08T00:00:00Z"
extra = { csr_season_id = "CsrSeason2", short_label = "S2" }
`)
	set, err := mappings.LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	return set
}

func dbSeason(id, name string, start time.Time, end *time.Time) domain.SeasonCalendar {
	return domain.SeasonCalendar{
		TitleID:   "halo_infinite",
		SeasonID:  id,
		Name:      name,
		StartDate: start,
		EndDate:   end,
	}
}

// ─── Tests ─────────────────────────────────────────────────────────────────

func TestSeasonsCatalog_TOMLOnly(t *testing.T) {
	// Pas de repo, pas de provider → catalog = TOML pur.
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), nil, nil, nil)
	got := cat.Load(context.Background(), "halo_infinite")
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 entries from TOML", len(got))
	}
	for _, e := range got {
		if e.Source != SeasonSourceTOMLOnly {
			t.Errorf("season %q: source = %v, want toml_only", e.ID, e.Source)
		}
	}
	if got[0].Label != "Heroes of Reach" {
		t.Errorf("FR label fallback = %q, want Heroes of Reach", got[0].Label)
	}
	if got[1].Label != "Loups solitaires" {
		t.Errorf("FR label = %q, want Loups solitaires", got[1].Label)
	}
}

func TestSeasonsCatalog_DBPopulated_Merged(t *testing.T) {
	// DB a S1+S2 avec dates fraîches (rétroactive : end de S1 décalée d'un jour),
	// TOML a S1+S2 avec dates initiales. Merge : DB wins pour dates, TOML
	// wins pour Label/Extra.
	dbStart1 := time.Date(2021, 12, 8, 0, 0, 0, 0, time.UTC)
	dbEnd1 := time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC) // décalée vs TOML
	dbStart2 := time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC)
	dbEnd2 := time.Date(2022, 11, 8, 0, 0, 0, 0, time.UTC)
	repo := &fakeMetadataRepo{
		listResults: [][]domain.SeasonCalendar{{
			dbSeason("season1", "Heroes of Reach", dbStart1, &dbEnd1),
			dbSeason("season2", "Lone Wolves", dbStart2, &dbEnd2),
		}},
	}
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, nil, nil)
	got := cat.Load(context.Background(), "halo_infinite")

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for _, e := range got {
		if e.Source != SeasonSourceMerged {
			t.Errorf("season %q: source = %v, want merged", e.ID, e.Source)
		}
	}
	// DB end date (4 mai) doit l'emporter sur TOML (3 mai).
	if got[0].End == nil || got[0].End.Day() != 4 {
		t.Errorf("S1 end date = %v, want 2022-05-04 (DB wins)", got[0].End)
	}
	// TOML labels (FR) préservés malgré la fusion.
	if got[1].Label != "Loups solitaires" {
		t.Errorf("S2 label = %q, want Loups solitaires (TOML wins)", got[1].Label)
	}
	// Extra TOML préservés.
	if got[0].Extra["csr_season_id"] != "CsrSeason1" {
		t.Errorf("S1 extra.csr_season_id absent : %v", got[0].Extra)
	}
	// Pas de fetch tenté car DB non vide.
	if repo.upsertCalls != 0 {
		t.Errorf("upsertCalls = %d, want 0 (DB déjà peuplée)", repo.upsertCalls)
	}
}

func TestSeasonsCatalog_DBEmpty_LazyFetchPersist(t *testing.T) {
	// DB vide → trigger fetch → upsert → relit DB. Vérifie que le 2e
	// ListSeasons retourne les saisons fetchées et qu'elles sont visibles.
	provStart := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	fetched := dbSeason("season14", "Skyfall", provStart, nil)
	repo := &fakeMetadataRepo{
		listResults: [][]domain.SeasonCalendar{
			{},        // 1er appel : DB vide
			{fetched}, // 2e appel : DB peuplée par UpsertSeason
		},
	}
	provider := &fakeSeasonProvider{
		fetchResults: []domain.SeasonCalendar{fetched},
	}
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, provider, nil)

	got := cat.Load(context.Background(), "halo_infinite")

	if provider.fetchCalls != 1 {
		t.Errorf("fetchCalls = %d, want 1", provider.fetchCalls)
	}
	if repo.upsertCalls != 1 {
		t.Errorf("upsertCalls = %d, want 1", repo.upsertCalls)
	}
	if repo.listCalls != 2 {
		t.Errorf("listCalls = %d, want 2 (1 avant fetch + 1 après upsert)", repo.listCalls)
	}

	// Catalog : TOML S1+S2 + DB-only S14
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (S1+S2 TOML + S14 DB-only)", len(got))
	}
	var s14 *SeasonCatalogEntry
	for i := range got {
		if got[i].ID == "season14" {
			s14 = &got[i]
		}
	}
	if s14 == nil {
		t.Fatal("season14 absente du catalog")
	}
	if s14.Source != SeasonSourceDBOnly {
		t.Errorf("S14 source = %v, want db_only", s14.Source)
	}
	if s14.Label != "Skyfall" {
		t.Errorf("S14 label = %q, want Skyfall (Name Waypoint fallback)", s14.Label)
	}
	if s14.End != nil {
		t.Errorf("S14 EndDate = %v, want nil (saison ouverte)", s14.End)
	}
}

func TestSeasonsCatalog_FetchErrorFallsBackToTOML(t *testing.T) {
	// DB vide + provider échoue (token absent ou Waypoint indispo) →
	// fallback gracieux sur le TOML.
	repo := &fakeMetadataRepo{
		listResults: [][]domain.SeasonCalendar{{}},
	}
	provider := &fakeSeasonProvider{
		fetchErr: errors.New("token absent du contexte"),
	}
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, provider, nil)

	got := cat.Load(context.Background(), "halo_infinite")

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (TOML seul)", len(got))
	}
	if repo.upsertCalls != 0 {
		t.Errorf("upsertCalls = %d, want 0 (fetch a échoué)", repo.upsertCalls)
	}
	for _, e := range got {
		if e.Source != SeasonSourceTOMLOnly {
			t.Errorf("season %q: source = %v, want toml_only", e.ID, e.Source)
		}
	}
}

func TestSeasonsCatalog_ListSeasonsErrorFallsBackToTOML(t *testing.T) {
	// Erreur DB au ListSeasons → fallback TOML, pas de crash.
	repo := &fakeMetadataRepo{
		listErr: errors.New("connection refused"),
	}
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, nil, nil)
	got := cat.Load(context.Background(), "halo_infinite")
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (fallback TOML)", len(got))
	}
}

func TestSeasonsCatalog_NoSources(t *testing.T) {
	// Aucun TOML, aucune DB, aucun provider → slice vide (pas nil obligatoire,
	// mais pas de crash).
	cat := NewSeasonsCatalog(nil, nil, nil, nil)
	got := cat.Load(context.Background(), "halo_infinite")
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestSeasonsCatalog_DBOnly_NoTOML(t *testing.T) {
	// Pas de TOML, DB peuplée → DB-only avec libellés Name Waypoint.
	dbStart := time.Date(2021, 12, 8, 0, 0, 0, 0, time.UTC)
	dbEnd := time.Date(2022, 5, 3, 0, 0, 0, 0, time.UTC)
	repo := &fakeMetadataRepo{
		listResults: [][]domain.SeasonCalendar{{
			dbSeason("season1", "Heroes of Reach", dbStart, &dbEnd),
		}},
	}
	cat := NewSeasonsCatalog(nil, repo, nil, nil)
	got := cat.Load(context.Background(), "halo_infinite")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Source != SeasonSourceDBOnly {
		t.Errorf("source = %v, want db_only", got[0].Source)
	}
	if got[0].Label != "Heroes of Reach" {
		t.Errorf("label = %q, want Heroes of Reach (Name Waypoint)", got[0].Label)
	}
}

func TestMergeSeasonSources_DBOnlyOrderedAfterTOML(t *testing.T) {
	tomlEntries := []SeasonCatalogEntry{
		{ID: "season1", DisplayOrder: 10, Label: "S1", Source: SeasonSourceTOMLOnly,
			Start: time.Date(2021, 12, 8, 0, 0, 0, 0, time.UTC)},
	}
	dbRows := []domain.SeasonCalendar{
		dbSeason("season_zz", "Future", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC), nil),
	}
	got := mergeSeasonSources(tomlEntries, dbRows)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	// TOML d'abord (display_order 10), DB-only ensuite (display_order > 10).
	if got[0].ID != "season1" {
		t.Errorf("position 0 = %q, want season1", got[0].ID)
	}
	if got[1].ID != "season_zz" {
		t.Errorf("position 1 = %q, want season_zz", got[1].ID)
	}
	if got[1].DisplayOrder <= got[0].DisplayOrder {
		t.Errorf("DB-only DisplayOrder=%d should be > TOML DisplayOrder=%d", got[1].DisplayOrder, got[0].DisplayOrder)
	}
}
