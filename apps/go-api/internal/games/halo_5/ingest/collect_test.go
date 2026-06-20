package ingest

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

func boolptr(b bool) *bool { return &b }
func intptr(i int) *int    { return &i }

func TestMatchRegistryRowFromSummary(t *testing.T) {
	start := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	s := canonical.MatchSummary{
		MatchID:         "match-xyz",
		StartedAtUTC:    start,
		DurationSeconds: intptr(540),
		Map:             &canonical.AssetReference{ID: "map-1", DefaultLabel: "Truth"},
		Playlist:        &canonical.AssetReference{ID: "pl-1", DefaultLabel: "Team Arena", VersionID: "v2"},
		PairMode:        &canonical.AssetReference{ID: "slayer", DefaultLabel: "Slayer"},
		IsRanked:        boolptr(true),
		IsPvE:           boolptr(false),
	}

	row := MatchRegistryRowFromSummary(s, "Madina97294")

	if row.MatchID != "match-xyz" || !row.StartTime.Equal(start) {
		t.Fatalf("identité/temps mal mappés: %+v", row)
	}
	if row.MapID == nil || *row.MapID != "map-1" || row.MapName == nil || *row.MapName != "Truth" {
		t.Errorf("map mal mappée: %+v", row)
	}
	if row.PlaylistVersionID == nil || *row.PlaylistVersionID != "v2" {
		t.Errorf("playlist version manquante: %+v", row)
	}
	if !row.IsRanked || row.IsFirefight {
		t.Errorf("flags ranked/firefight mal dérivés: ranked=%v ff=%v", row.IsRanked, row.IsFirefight)
	}
	if row.ModeCategory != "Slayer" || row.FirstSyncBy != "Madina97294" {
		t.Errorf("mode/first_sync_by: cat=%q by=%q", row.ModeCategory, row.FirstSyncBy)
	}
	// Champs non fournis → nil (registry tolère).
	if row.EndTime != nil || row.SeasonID != nil || row.Team0Score != nil {
		t.Errorf("champs absents devraient être nil: %+v", row)
	}
}

func TestCollectMedalsBatch_EndToEnd(t *testing.T) {
	s := canonical.MatchSummary{
		MatchID:      "m1",
		StartedAtUTC: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
		IsRanked:     boolptr(true),
	}
	timeline := []canonical.MatchEvent{
		medal("Madina97294", "100", 5000),
		medal("Madina97294", "100", 9000),
		medal("JGtm", "200", 3000),
		{Type: canonical.MatchEventKill, TimeMs: 1000}, // ignoré (tranche médailles)
	}
	resolve := fakeResolver(map[string]string{"Madina97294": "xA", "JGtm": "xB"})
	viewer := canonical.PlayerIdentity{Gamertag: "Madina97294", XUID: "xA"}

	batch := CollectMedalsBatch("halo_5", "h5_capture", viewer, s, timeline, resolve)

	if batch.TitleSlug != "halo_5" || batch.Player != "Madina97294" || batch.XUID != "xA" {
		t.Fatalf("métadonnées batch: slug=%q player=%q xuid=%q", batch.TitleSlug, batch.Player, batch.XUID)
	}
	// match_registry présent → SharedPersister ne no-op pas.
	if batch.Shared.Match == nil || batch.Shared.Match.MatchID != "m1" {
		t.Fatalf("match_registry (ancre) absent du batch: %+v", batch.Shared.Match)
	}
	// 2 lignes agrégat (Madina/100=2, JGtm/200=1) + 3 events horodatés.
	if len(batch.Shared.Medals) != 2 {
		t.Errorf("medals agrégat: %d, attendu 2 — %+v", len(batch.Shared.Medals), batch.Shared.Medals)
	}
	if len(batch.Shared.HighlightEvents) != 3 {
		t.Errorf("highlight_events: %d, attendu 3", len(batch.Shared.HighlightEvents))
	}
}
