package service

import (
	"testing"
	"time"
)

func tm(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func TestBuildMatchesPerSeason_Bucketing(t *testing.T) {
	endS1 := tm("2022-05-03T17:00:00Z")
	endS2 := tm("2022-11-08T17:00:00Z")
	seasons := []SeasonCatalogEntry{
		{ID: "season1", Label: "Heroes of Reach", Start: tm("2021-12-08T17:00:00Z"), End: &endS1, Extra: map[string]string{"short_label": "S1"}},
		{ID: "season2", Label: "Lone Wolves", Start: tm("2022-05-03T17:00:00Z"), End: &endS2, Extra: map[string]string{"short_label": "S2"}},
		{ID: "season3", Label: "Open", Start: tm("2022-11-08T17:00:00Z"), End: nil}, // saison ouverte, pas de short_label
	}
	starts := []time.Time{
		tm("2022-01-10T10:00:00Z"), // S1
		tm("2022-02-10T10:00:00Z"), // S1
		tm("2022-06-10T10:00:00Z"), // S2
		tm("2023-01-10T10:00:00Z"), // S3 (ouverte)
		tm("2020-01-01T10:00:00Z"), // avant toute saison → ignoré
	}

	got := buildMatchesPerSeason(starts, seasons)
	if len(got) != 3 {
		t.Fatalf("attendu 3 saisons avec matchs, got %d (%+v)", len(got), got)
	}
	// Ordre préservé (DisplayOrder via l'ordre du slice).
	if got[0].SeasonID != "season1" || got[0].Matches != 2 || got[0].SeasonName != "S1" {
		t.Errorf("S1 attendu (2 matchs, label S1), got %+v", got[0])
	}
	if got[1].Matches != 1 || got[1].SeasonName != "S2" {
		t.Errorf("S2 attendu 1 match, got %+v", got[1])
	}
	// season3 sans short_label → fallback Label.
	if got[2].SeasonID != "season3" || got[2].Matches != 1 || got[2].SeasonName != "Open" {
		t.Errorf("S3 attendu 1 match label fallback 'Open', got %+v", got[2])
	}
}

func TestBuildMatchesPerSeason_Empty(t *testing.T) {
	seasons := []SeasonCatalogEntry{{ID: "s1", Start: tm("2021-01-01T00:00:00Z")}}
	if got := buildMatchesPerSeason(nil, seasons); got != nil {
		t.Errorf("starts vide → nil, got %v", got)
	}
	if got := buildMatchesPerSeason([]time.Time{tm("2021-06-01T00:00:00Z")}, nil); got != nil {
		t.Errorf("seasons vide → nil, got %v", got)
	}
}
