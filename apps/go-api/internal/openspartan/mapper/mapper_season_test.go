// Package mapper — mapper_season_test.go : couverture Phase 9.5 OpenSpartan.
//
// Vérifie que MapRegistry populate IsRanked + SeasonID au lieu de laisser les
// valeurs par défaut (false, nil), ce qui aurait nécessité un backfill migration
// après chaque import OpenSpartan.
package mapper

import (
	"testing"
	"time"

	"levelup/go-api/internal/openspartan"
)

func baseMatchInfo() openspartan.MatchInfo {
	return openspartan.MatchInfo{
		StartTime:        time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
		EndTime:          time.Date(2026, 4, 1, 12, 12, 0, 0, time.UTC),
		Duration:         "PT12M",
		PlayableDuration: "PT12M",
	}
}

func TestMapRegistry_PopulatesSeasonIDFromPayload(t *testing.T) {
	mi := baseMatchInfo()
	mi.SeasonID = "CsrSeason13-1"
	ms := openspartan.MatchStats{
		MatchID:   "season-test-1",
		MatchInfo: mi,
	}
	row, err := MapRegistry(ms, MapOptions{Now: time.Now(), Source: "test"})
	if err != nil {
		t.Fatalf("MapRegistry: %v", err)
	}
	if row.SeasonID == nil || *row.SeasonID != "CsrSeason13-1" {
		t.Errorf("SeasonID = %v, want CsrSeason13-1", row.SeasonID)
	}
}

func TestMapRegistry_NilSeasonIDWhenAbsent(t *testing.T) {
	ms := openspartan.MatchStats{
		MatchID:   "no-season-test",
		MatchInfo: baseMatchInfo(),
	}
	row, err := MapRegistry(ms, MapOptions{Now: time.Now(), Source: "test"})
	if err != nil {
		t.Fatalf("MapRegistry: %v", err)
	}
	if row.SeasonID != nil {
		t.Errorf("SeasonID should be nil when payload SeasonId absent, got %v", row.SeasonID)
	}
}

func TestMapRegistry_NilSeasonIDWhenWhitespace(t *testing.T) {
	mi := baseMatchInfo()
	mi.SeasonID = "   "
	ms := openspartan.MatchStats{
		MatchID:   "whitespace-season",
		MatchInfo: mi,
	}
	row, err := MapRegistry(ms, MapOptions{Now: time.Now(), Source: "test"})
	if err != nil {
		t.Fatalf("MapRegistry: %v", err)
	}
	if row.SeasonID != nil {
		t.Errorf("SeasonID should be nil for whitespace-only payload, got %v", row.SeasonID)
	}
}

// NOTE : IsRanked dans le mapper OpenSpartan dépend de PlaylistName qui n'est
// populé que par le post-import recompute (commentaire rows.go ligne 17).
// L'heuristique appliquée dans MapRegistry est donc no-op au moment du mapping
// brut (PlaylistName == nil). Le post-import recompute appliquera la même
// heuristique sur les données alors disponibles. Pas de test direct ici pour
// éviter de tester un comportement qui dépend de post-import.
