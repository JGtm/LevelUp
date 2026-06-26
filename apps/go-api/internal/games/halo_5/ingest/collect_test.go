package ingest

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
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
	// end_time dérivé de start + durée (540s → 12:09:00) : l'API h5 ne fournit pas
	// d'horodatage de fin, mais la durée oui → end_time calculé (cf. registry.go).
	wantEnd := start.Add(540 * time.Second)
	if row.EndTime == nil || !row.EndTime.Equal(wantEnd) {
		t.Errorf("end_time devrait être start+durée (%v): got %v", wantEnd, row.EndTime)
	}
	// Champs non fournis → nil (registry tolère).
	if row.SeasonID != nil || row.Team0Score != nil {
		t.Errorf("champs absents devraient être nil: %+v", row)
	}
}

func TestCollectMatchBatch_EndToEnd(t *testing.T) {
	s := canonical.MatchSummary{
		MatchID:      "m1",
		StartedAtUTC: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
		IsRanked:     boolptr(true),
	}
	timeline := []canonical.MatchEvent{
		medal("Madina97294", "100", 5000),
		medal("Madina97294", "100", 9000),
		medal("JGtm", "200", 3000),
		kill("Madina97294", "JGtm", "12345", 6000),
		kill("JGtm", "Madina97294", "", 8000),
	}
	resolve := fakeResolver(map[string]string{"Madina97294": "xA", "JGtm": "xB"})
	viewer := canonical.PlayerIdentity{Gamertag: "Madina97294", XUID: "xA"}
	participants := []domain.MatchParticipantRow{
		{MatchID: "m1", XUID: "xA"},
		{MatchID: "m1", XUID: "xB"},
	}
	commendations := []persist.CommendationInsert{
		{MatchID: "m1", XUID: "xA", CommendationID: "uuid-1", Count: 3},
		{MatchID: "m1", XUID: "xB", CommendationID: "uuid-2", Count: 1},
	}

	batch := CollectMatchBatch("halo_5", "h5_capture", viewer, s, timeline, participants, commendations, intptr(50), intptr(30), 8, resolve)

	if batch.Shared.Match == nil || batch.Shared.Match.Team0Score == nil || *batch.Shared.Match.Team0Score != 50 ||
		batch.Shared.Match.Team1Score == nil || *batch.Shared.Match.Team1Score != 30 {
		t.Fatalf("scores objectif d'équipe non posés sur le registry: %+v", batch.Shared.Match)
	}
	// player_count = roster API reporté (8), oracle d'intégrité vs les participants
	// réellement persistés (2 ici → 6 droppés, signalés par l'écart).
	if batch.Shared.Match.PlayerCount != 8 {
		t.Errorf("player_count = %d, want 8 (roster API)", batch.Shared.Match.PlayerCount)
	}

	if batch.TitleSlug != "halo_5" || batch.Player != "Madina97294" || batch.XUID != "xA" {
		t.Fatalf("métadonnées batch: slug=%q player=%q xuid=%q", batch.TitleSlug, batch.Player, batch.XUID)
	}
	// match_registry présent → SharedPersister ne no-op pas.
	if batch.Shared.Match == nil || batch.Shared.Match.MatchID != "m1" {
		t.Fatalf("match_registry (ancre) absent du batch: %+v", batch.Shared.Match)
	}
	// match_participants : roster transmis tel quel.
	if len(batch.Shared.Participants) != 2 {
		t.Errorf("participants: %d, attendu 2", len(batch.Shared.Participants))
	}
	// Médailles : 2 agrégats (Madina/100=2, JGtm/200=1) + 3 events horodatés.
	if len(batch.Shared.Medals) != 2 {
		t.Errorf("medals agrégat: %d, attendu 2 — %+v", len(batch.Shared.Medals), batch.Shared.Medals)
	}
	if len(batch.Shared.HighlightEvents) != 3 {
		t.Errorf("highlight_events: %d, attendu 3", len(batch.Shared.HighlightEvents))
	}
	// Kills : 2 paires par-kill + 1 arme (1er kill seulement).
	if len(batch.Shared.KillerVictim) != 2 {
		t.Errorf("killer_victim_pairs: %d, attendu 2 — %+v", len(batch.Shared.KillerVictim), batch.Shared.KillerVictim)
	}
	if len(batch.Shared.WeaponKills) != 1 {
		t.Errorf("weapon_kills: %d, attendu 1", len(batch.Shared.WeaponKills))
	}
	// Commendations natives : transmises telles quelles au batch (AXE B).
	if len(batch.Shared.Commendations) != 2 {
		t.Errorf("commendations: %d, attendu 2 — %+v", len(batch.Shared.Commendations), batch.Shared.Commendations)
	}
}
