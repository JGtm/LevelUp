package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeMapRows() []domain.FilterMatchRow {
	// 4 rows with EN map name == MapNameFR (no FR in DB) → eligible for enrichment
	// 3 rows with FR map name already set
	rows := make([]domain.FilterMatchRow, 0, 7)
	for i := 0; i < 4; i++ {
		en := "Aquarius"
		fr := "Aquarius Bord de mer"
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "m" + string(rune('0'+i)),
			MapName:   strPtr(en),
			MapNameFR: strPtr(fr), // applyMapFRTranslations enriched this
		})
	}
	for i := 0; i < 3; i++ {
		en := "Recharge"
		fr := "Station de Recharge"
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "n" + string(rune('0'+i)),
			MapName:   strPtr(en),
			MapNameFR: strPtr(fr),
		})
	}
	return rows
}

func makePlaylistRows() []domain.FilterMatchRow {
	rows := make([]domain.FilterMatchRow, 0, 6)
	for i := 0; i < 3; i++ {
		en := "Ranked Arena"
		fr := "Arène classée"
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "p" + string(rune('0'+i)),
			PlaylistNameEN: strPtr(en),
			PlaylistName:   strPtr(fr),
		})
	}
	for i := 0; i < 3; i++ {
		en := "Quick Play"
		fr := "Jeu rapide"
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "q" + string(rune('0'+i)),
			PlaylistNameEN: strPtr(en),
			PlaylistName:   strPtr(fr),
		})
	}
	return rows
}

// ---------------------------------------------------------------------------
// buildMapTranslationMap
// ---------------------------------------------------------------------------

func TestBuildMapTranslationMap_WithTranslations(t *testing.T) {
	rows := makeMapRows()
	tr := buildMapTranslationMap(rows)
	if len(tr) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tr))
	}
	if tr["Aquarius"] != "Aquarius Bord de mer" {
		t.Errorf("expected 'Aquarius Bord de mer', got %q", tr["Aquarius"])
	}
	if tr["Recharge"] != "Station de Recharge" {
		t.Errorf("expected 'Station de Recharge', got %q", tr["Recharge"])
	}
}

func TestBuildMapTranslationMap_SameName_NotIncluded(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", MapName: strPtr("Catalyst"), MapNameFR: strPtr("Catalyst")},
	}
	tr := buildMapTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map when EN==FR, got %d entries", len(tr))
	}
}

func TestBuildMapTranslationMap_NilMapName(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", MapName: nil, MapNameFR: strPtr("Aquarius Bord de mer")},
	}
	tr := buildMapTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map for nil MapName, got %d entries", len(tr))
	}
}

// ---------------------------------------------------------------------------
// buildPlaylistTranslationMap
// ---------------------------------------------------------------------------

func TestBuildPlaylistTranslationMap_WithTranslations(t *testing.T) {
	rows := makePlaylistRows()
	tr := buildPlaylistTranslationMap(rows)
	if len(tr) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tr))
	}
	if tr["Ranked Arena"] != "Arène classée" {
		t.Errorf("expected 'Arène classée', got %q", tr["Ranked Arena"])
	}
	if tr["Quick Play"] != "Jeu rapide" {
		t.Errorf("expected 'Jeu rapide', got %q", tr["Quick Play"])
	}
}

func TestBuildPlaylistTranslationMap_SameName_NotIncluded(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", PlaylistNameEN: strPtr("Ranked Arena"), PlaylistName: strPtr("Ranked Arena")},
	}
	tr := buildPlaylistTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map when EN==FR, got %d entries", len(tr))
	}
}

func TestBuildPlaylistTranslationMap_NilEN(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", PlaylistNameEN: nil, PlaylistName: strPtr("Arène classée")},
	}
	tr := buildPlaylistTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map for nil PlaylistNameEN, got %d entries", len(tr))
	}
}

// ---------------------------------------------------------------------------
// migrateCascadeValues
// ---------------------------------------------------------------------------

func TestMigrateCascadeValues_TranslatesKnown(t *testing.T) {
	tr := map[string]string{"Aquarius": "Aquarius Bord de mer", "Recharge": "Station de Recharge"}
	in := []string{"Aquarius", "Recharge", "Unknown Map"}
	out := migrateCascadeValues(in, tr)
	if out[0] != "Aquarius Bord de mer" {
		t.Errorf("[0] want 'Aquarius Bord de mer', got %q", out[0])
	}
	if out[1] != "Station de Recharge" {
		t.Errorf("[1] want 'Station de Recharge', got %q", out[1])
	}
	if out[2] != "Unknown Map" {
		t.Errorf("[2] want 'Unknown Map' preserved, got %q", out[2])
	}
}

func TestMigrateCascadeValues_EmptyTranslation(t *testing.T) {
	in := []string{"Aquarius", "Recharge"}
	out := migrateCascadeValues(in, map[string]string{})
	for i, v := range out {
		if v != in[i] {
			t.Errorf("[%d] want %q preserved, got %q", i, in[i], v)
		}
	}
}

// ---------------------------------------------------------------------------
// ResolveFiltersFromRows — cascade migration for maps
// ---------------------------------------------------------------------------

func makeMapFilterRows() []domain.FilterMatchRow {
	rows := make([]domain.FilterMatchRow, 0, 8)
	for i := 0; i < 5; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "a" + string(rune('0'+i)),
			MapName:   strPtr("Aquarius"),
			MapNameFR: strPtr("Aquarius Bord de mer"),
		})
	}
	for i := 0; i < 3; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "r" + string(rune('0'+i)),
			MapName:   strPtr("Recharge"),
			MapNameFR: strPtr("Station de Recharge"),
		})
	}
	return rows
}

func TestResolveFilters_LegacyEnglishMapFilter_MigratedToFR(t *testing.T) {
	rows := makeMapFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Maps: []string{"Aquarius"}, // stored as EN, should be migrated to FR
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches after EN→FR map migration, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_FrenchMapFilter_WorksDirectly(t *testing.T) {
	rows := makeMapFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Maps: []string{"Aquarius Bord de mer"},
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches for direct FR map filter, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_MapOptions_AreFR(t *testing.T) {
	rows := makeMapFilterRows()
	result := ResolveFiltersFromRows(rows, domain.FilterContextInput{})
	maps := result.AvailableOptions.Maps
	for _, m := range maps {
		if m.Value == "Aquarius" || m.Value == "Recharge" {
			t.Errorf("map option should be FR, got EN value %q", m.Value)
		}
	}
	found := false
	for _, m := range maps {
		if m.Value == "Aquarius Bord de mer" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Aquarius Bord de mer' in map options")
	}
}

// ---------------------------------------------------------------------------
// ResolveFiltersFromRows — cascade migration for playlists
// ---------------------------------------------------------------------------

func makePlaylistFilterRows() []domain.FilterMatchRow {
	rows := make([]domain.FilterMatchRow, 0, 8)
	for i := 0; i < 5; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "p" + string(rune('0'+i)),
			PlaylistNameEN: strPtr("Ranked Arena"),
			PlaylistName:   strPtr("Arène classée"),
		})
	}
	for i := 0; i < 3; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "q" + string(rune('0'+i)),
			PlaylistNameEN: strPtr("Quick Play"),
			PlaylistName:   strPtr("Jeu rapide"),
		})
	}
	return rows
}

func TestResolveFilters_LegacyEnglishPlaylistFilter_MigratedToFR(t *testing.T) {
	rows := makePlaylistFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Ranked Arena"}, // stored as EN
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches after EN→FR playlist migration, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_FrenchPlaylistFilter_WorksDirectly(t *testing.T) {
	rows := makePlaylistFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Arène classée"},
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches for direct FR playlist filter, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_PlaylistOptions_AreFR(t *testing.T) {
	rows := makePlaylistFilterRows()
	result := ResolveFiltersFromRows(rows, domain.FilterContextInput{})
	playlists := result.AvailableOptions.Playlists
	for _, p := range playlists {
		if p.Value == "Ranked Arena" || p.Value == "Quick Play" {
			t.Errorf("playlist option should be FR, got EN value %q", p.Value)
		}
	}
	found := false
	for _, p := range playlists {
		if p.Value == "Arène classée" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Arène classée' in playlist options")
	}
}

func TestResolveFilters_MultiplePlaylistsLegacy_AllMigrated(t *testing.T) {
	rows := makePlaylistFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Ranked Arena", "Quick Play"},
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 8 {
		t.Errorf("expected 8 matches for both playlists migrated, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}
