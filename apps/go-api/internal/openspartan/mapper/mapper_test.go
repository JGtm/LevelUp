package mapper

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/openspartan"
)

const (
	ownerXUID = "2533274823110022"
	otherXUID = "2533274801010001"
)

func sampleMatch(t *testing.T) openspartan.MatchStats {
	t.Helper()
	teamStats := json.RawMessage(`{"CoreStats":{"Score":50,"PersonalScore":5810}}`)
	return openspartan.MatchStats{
		MatchID: "11111111-aaaa-bbbb-cccc-000000000001",
		MatchInfo: openspartan.MatchInfo{
			StartTime:           time.Date(2026, 1, 2, 20, 18, 1, 0, time.UTC),
			EndTime:             time.Date(2026, 1, 2, 20, 30, 0, 0, time.UTC),
			Duration:            "PT11M59.25S",
			PlayableDuration:    "PT11M59.25S",
			LifecycleMode:       3,
			GameVariantCategory: 6,
			LevelID:             "level-1",
			MapVariant:          openspartan.AssetRef{AssetKind: 2, AssetID: "map-1", VersionID: "map-v1"},
			UgcGameVariant:      openspartan.AssetRef{AssetKind: 6, AssetID: "variant-1", VersionID: "variant-v1"},
			Playlist:            &openspartan.AssetRef{AssetKind: 3, AssetID: "playlist-1", VersionID: "playlist-v1"},
			PlaylistMapModePair: &openspartan.AssetRef{AssetKind: 7, AssetID: "pair-1", VersionID: "pair-v1"},
			TeamsEnabled:        true,
			TeamScoringEnabled:  true,
		},
		Teams: []openspartan.Team{
			{TeamID: 0, Outcome: 2, Rank: 1, Stats: teamStats},
			{TeamID: 1, Outcome: 3, Rank: 2, Stats: json.RawMessage(`{"CoreStats":{"Score":47,"PersonalScore":4321}}`)},
		},
		Players: []openspartan.Player{
			{
				PlayerID: "xuid(" + ownerXUID + ")", PlayerType: 1,
				LastTeamID: 0, Outcome: 2, Rank: 1,
				ParticipationInfo: openspartan.ParticipationInfo{TimePlayed: "PT11M59.25S"},
				PlayerTeamStats: []openspartan.PlayerTeamStat{{
					TeamID: 0,
					Stats: openspartan.StatsBundle{CoreStats: openspartan.CoreStats{
						Score: 19, PersonalScore: 2050, Kills: 19, Deaths: 11, Assists: 3,
						KDA: 9.0, Accuracy: 60.32, ShotsFired: 436, ShotsHit: 263,
						DamageDealt: 4889, DamageTaken: 4159, HeadshotKills: 11,
						MaxKillingSpree: 6, GrenadeKills: 3, MeleeKills: 4,
						AverageLifeDuration: "PT46S",
						Medals: []openspartan.MedalAward{
							{NameID: 3546244406, Count: 1},
							{NameID: 622331684, Count: 2},
						},
					}},
				}},
			},
			{
				PlayerID: "xuid(" + otherXUID + ")", PlayerType: 1,
				LastTeamID: 1, Outcome: 3, Rank: 4,
				ParticipationInfo: openspartan.ParticipationInfo{TimePlayed: "PT11M"},
				PlayerTeamStats: []openspartan.PlayerTeamStat{{
					TeamID: 1,
					Stats: openspartan.StatsBundle{CoreStats: openspartan.CoreStats{
						Kills: 5, Deaths: 12, Medals: []openspartan.MedalAward{
							{NameID: 622331684, Count: 1},
						},
					}},
				}},
			},
			{
				PlayerID: "bid(123-1)", PlayerType: 2,
				LastTeamID: 1, Outcome: 3, Rank: 5,
				ParticipationInfo: openspartan.ParticipationInfo{TimePlayed: "PT11M"},
				PlayerTeamStats: []openspartan.PlayerTeamStat{{
					TeamID: 1,
					Stats: openspartan.StatsBundle{CoreStats: openspartan.CoreStats{
						Kills: 1, Deaths: 7, Medals: []openspartan.MedalAward{
							{NameID: 999, Count: 9},
						},
					}},
				}},
			},
		},
	}
}

func TestMapMatch_FullCycle(t *testing.T) {
	ms := sampleMatch(t)
	pm := &openspartan.ParsedMatch{
		MatchID: ms.MatchID,
		Stats:   ms,
		PlayerStats: []openspartan.PlayerMatchStatsValue{
			{ID: "xuid(" + ownerXUID + ")", Result: &openspartan.PlayerMatchStatsResult{TeamMmr: 1041.7}},
		},
	}
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	mm, err := MapMatch(pm, MapOptions{
		Now:    now,
		Source: "test",
		AliasResolver: func(xuid string) string {
			if xuid == ownerXUID {
				return "TestOwner"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("MapMatch: %v", err)
	}

	if mm.Registry.MatchID != ms.MatchID {
		t.Errorf("registry MatchID: want %s, got %s", ms.MatchID, mm.Registry.MatchID)
	}
	if mm.Registry.DurationSeconds != 719 { // PT11M59.25S
		t.Errorf("DurationSeconds: want 719, got %d", mm.Registry.DurationSeconds)
	}
	if mm.Registry.PlayerCount != 2 {
		t.Errorf("PlayerCount: want 2 humans, got %d", mm.Registry.PlayerCount)
	}
	if mm.Registry.FirstSyncBy != "test" {
		t.Errorf("FirstSyncBy: want 'test', got %q", mm.Registry.FirstSyncBy)
	}
	if !mm.Registry.FirstSyncAt.Equal(now) {
		t.Errorf("FirstSyncAt: want %s, got %s", now, mm.Registry.FirstSyncAt)
	}
	if mm.Registry.Team0Score == nil || *mm.Registry.Team0Score != 50 {
		t.Errorf("Team0Score: want 50, got %v", deref(mm.Registry.Team0Score))
	}
	if mm.Registry.Team1Score == nil || *mm.Registry.Team1Score != 47 {
		t.Errorf("Team1Score: want 47, got %v", deref(mm.Registry.Team1Score))
	}
	if mm.Registry.PlaylistID == nil || *mm.Registry.PlaylistID != "playlist-1" {
		t.Errorf("PlaylistID: want 'playlist-1', got %v", mm.Registry.PlaylistID)
	}
	if !mm.Registry.ParticipantsLoaded || !mm.Registry.EventsLoaded || !mm.Registry.MedalsLoaded {
		t.Error("ParticipantsLoaded / EventsLoaded / MedalsLoaded should all be true")
	}

	if len(mm.Participants) != 2 {
		t.Fatalf("Participants: want 2 (bot filtered), got %d", len(mm.Participants))
	}
	owner := findParticipant(t, mm.Participants, ownerXUID)
	if owner.Gamertag == nil || *owner.Gamertag != "TestOwner" {
		t.Errorf("owner gamertag: want 'TestOwner', got %v", owner.Gamertag)
	}
	if owner.Kills != 19 || owner.Deaths != 11 || owner.Assists != 3 {
		t.Errorf("owner K/D/A: want 19/11/3, got %d/%d/%d", owner.Kills, owner.Deaths, owner.Assists)
	}
	if owner.TeamMMR == nil || *owner.TeamMMR != 1041.7 {
		t.Errorf("owner TeamMMR: want 1041.7, got %v", owner.TeamMMR)
	}
	if owner.AvgLifeSeconds == nil || *owner.AvgLifeSeconds != 46.0 {
		t.Errorf("owner AvgLifeSeconds: want 46.0, got %v", owner.AvgLifeSeconds)
	}
	if owner.MaxKillingSpree == nil || *owner.MaxKillingSpree != 6 {
		t.Errorf("owner MaxKillingSpree: want 6, got %v", owner.MaxKillingSpree)
	}

	if len(mm.Medals) != 3 {
		t.Errorf("Medals: want 3 unique (owner=2, other=1, bot filtered), got %d", len(mm.Medals))
	}
	ownerMedalCount := 0
	for _, m := range mm.Medals {
		if m.XUID == ownerXUID {
			ownerMedalCount++
		}
		if m.MedalNameID == 999 {
			t.Error("bot medal_id=999 should have been filtered out")
		}
	}
	if ownerMedalCount != 2 {
		t.Errorf("owner medal rows: want 2, got %d", ownerMedalCount)
	}
}

func TestMapMatch_RejectsNil(t *testing.T) {
	if _, err := MapMatch(nil, MapOptions{}); !errors.Is(err, ErrInvalidMatch) {
		t.Errorf("MapMatch(nil): want ErrInvalidMatch, got %v", err)
	}
}

func TestMapMatch_RejectsEmptyMatchID(t *testing.T) {
	pm := &openspartan.ParsedMatch{MatchID: "  ", Stats: openspartan.MatchStats{Players: []openspartan.Player{{PlayerType: 1}}}}
	_, err := MapMatch(pm, MapOptions{})
	if !errors.Is(err, ErrInvalidMatch) {
		t.Errorf("want ErrInvalidMatch, got %v", err)
	}
}

func TestMapRegistry_RejectsFutureStartTime(t *testing.T) {
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	ms := openspartan.MatchStats{
		MatchID: "future-1",
		MatchInfo: openspartan.MatchInfo{
			StartTime: now.AddDate(0, 1, 0), // a month ahead
			Duration:  "PT10M", PlayableDuration: "PT10M",
		},
	}
	if _, err := MapRegistry(ms, MapOptions{Now: now}); !errors.Is(err, ErrFutureMatch) {
		t.Errorf("want ErrFutureMatch, got %v", err)
	}
}

func TestMapRegistry_HandlesMissingOptionalAssets(t *testing.T) {
	ms := openspartan.MatchStats{
		MatchID: "x",
		MatchInfo: openspartan.MatchInfo{
			StartTime: time.Now().Add(-1 * time.Hour),
			Duration:  "PT5M", PlayableDuration: "PT5M",
			// No Playlist, no PlaylistMapModePair, no Teams.
			MapVariant:     openspartan.AssetRef{AssetID: "map-1"},
			UgcGameVariant: openspartan.AssetRef{AssetID: "var-1"},
		},
	}
	row, err := MapRegistry(ms, MapOptions{})
	if err != nil {
		t.Fatalf("MapRegistry: %v", err)
	}
	if row.PlaylistID != nil {
		t.Errorf("PlaylistID: want nil, got %v", row.PlaylistID)
	}
	if row.PairID != nil {
		t.Errorf("PairID: want nil, got %v", row.PairID)
	}
	if row.Team0Score != nil || row.Team1Score != nil {
		t.Error("Team scores should be nil when Teams[] is empty")
	}
	if row.MapID == nil || *row.MapID != "map-1" {
		t.Errorf("MapID: want 'map-1', got %v", row.MapID)
	}
}

func TestMapParticipants_HonoursTeamSwitchByPickingLastEntry(t *testing.T) {
	ms := openspartan.MatchStats{
		MatchID: "ts",
		Players: []openspartan.Player{{
			PlayerID: "xuid(" + ownerXUID + ")", PlayerType: 1,
			Rank: 2, Outcome: 3,
			ParticipationInfo: openspartan.ParticipationInfo{TimePlayed: "PT10M"},
			PlayerTeamStats: []openspartan.PlayerTeamStat{
				{TeamID: 0, Stats: openspartan.StatsBundle{CoreStats: openspartan.CoreStats{Kills: 3, Deaths: 1}}},
				{TeamID: 1, Stats: openspartan.StatsBundle{CoreStats: openspartan.CoreStats{Kills: 8, Deaths: 4}}},
			},
		}},
	}
	parts, err := MapParticipants(ms, nil, MapOptions{})
	if err != nil {
		t.Fatalf("MapParticipants: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("want 1 participant, got %d", len(parts))
	}
	if parts[0].TeamID != 1 {
		t.Errorf("TeamID: want last (1), got %d", parts[0].TeamID)
	}
	if parts[0].Kills != 8 {
		t.Errorf("Kills should reflect the last PlayerTeamStats (8), got %d", parts[0].Kills)
	}
}

func TestMapMedals_AggregatesAcrossTeamSwitch(t *testing.T) {
	ms := openspartan.MatchStats{
		MatchID: "ag",
		Players: []openspartan.Player{{
			PlayerID: "xuid(" + ownerXUID + ")", PlayerType: 1,
			PlayerTeamStats: []openspartan.PlayerTeamStat{
				{Stats: openspartan.StatsBundle{CoreStats: openspartan.CoreStats{
					Medals: []openspartan.MedalAward{{NameID: 100, Count: 2}},
				}}},
				{Stats: openspartan.StatsBundle{CoreStats: openspartan.CoreStats{
					Medals: []openspartan.MedalAward{{NameID: 100, Count: 3}, {NameID: 200, Count: 1}},
				}}},
			},
		}},
	}
	medals, err := MapMedals(ms)
	if err != nil {
		t.Fatalf("MapMedals: %v", err)
	}
	totals := make(map[int64]int16)
	for _, m := range medals {
		totals[m.MedalNameID] += m.Count
	}
	if totals[100] != 5 {
		t.Errorf("medal 100 aggregated count: want 5 (2+3), got %d", totals[100])
	}
	if totals[200] != 1 {
		t.Errorf("medal 200 count: want 1, got %d", totals[200])
	}
}

func TestMapMedals_NoMedalsReturnsNil(t *testing.T) {
	ms := openspartan.MatchStats{
		MatchID: "nm",
		Players: []openspartan.Player{{
			PlayerID: "xuid(" + ownerXUID + ")", PlayerType: 1,
			PlayerTeamStats: []openspartan.PlayerTeamStat{{}},
		}},
	}
	medals, err := MapMedals(ms)
	if err != nil {
		t.Fatalf("MapMedals: %v", err)
	}
	if medals != nil {
		t.Errorf("want nil slice for no-medal match, got %v", medals)
	}
}

func TestMapHighlight_NumericXUID(t *testing.T) {
	body := []byte(`{"event_type":"kill","time_ms":46832,"xuid":2533274945467756,"type_hint":50}`)
	row, err := MapHighlight("m-1", body)
	if err != nil {
		t.Fatalf("MapHighlight: %v", err)
	}
	if row.EventType != "kill" {
		t.Errorf("EventType: want 'kill', got %q", row.EventType)
	}
	if row.TimeMs == nil || *row.TimeMs != 46832 {
		t.Errorf("TimeMs: want 46832, got %v", row.TimeMs)
	}
	if row.XUID == nil || *row.XUID != "2533274945467756" {
		t.Errorf("XUID: want '2533274945467756', got %v", row.XUID)
	}
	if row.TypeHint == nil || *row.TypeHint != 50 {
		t.Errorf("TypeHint: want 50, got %v", row.TypeHint)
	}
	if !strings.Contains(row.RawJSON, "event_type") {
		t.Errorf("RawJSON should preserve the original body, got %q", row.RawJSON)
	}
}

func TestMapHighlight_StringXUIDWrapped(t *testing.T) {
	body := []byte(`{"event_type":"medal","xuid":"xuid(` + ownerXUID + `)","time_ms":1234}`)
	row, err := MapHighlight("m-2", body)
	if err != nil {
		t.Fatalf("MapHighlight: %v", err)
	}
	if row.XUID == nil || *row.XUID != ownerXUID {
		t.Errorf("XUID: want %s, got %v", ownerXUID, row.XUID)
	}
}

func TestMapHighlight_RejectsEmptyEventType(t *testing.T) {
	body := []byte(`{"event_type":"","xuid":2533274945467756}`)
	if _, err := MapHighlight("m-3", body); !errors.Is(err, ErrInvalidMatch) {
		t.Errorf("want ErrInvalidMatch, got %v", err)
	}
}

func TestParseISO8601Duration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"PT10M", 10 * time.Minute},
		{"PT46S", 46 * time.Second},
		{"PT11M59.25S", 11*time.Minute + 59*time.Second + 250*time.Millisecond},
		{"PT1H30M", time.Hour + 30*time.Minute},
		{"PT1M8.2S", time.Minute + 8*time.Second + 200*time.Millisecond},
	}
	for _, tc := range cases {
		got, err := ParseISO8601Duration(tc.in)
		if err != nil {
			t.Errorf("ParseISO8601Duration(%q) returned error: %v", tc.in, err)
			continue
		}
		// Allow a 1ms slop for floating-point conversions.
		diff := got - tc.want
		if diff < -time.Millisecond || diff > time.Millisecond {
			t.Errorf("ParseISO8601Duration(%q): want %v, got %v", tc.in, tc.want, got)
		}
	}
}

func TestParseISO8601Duration_InvalidInputs(t *testing.T) {
	invalid := []string{"P1Y", "PT", "10M", "not-a-duration", "PT1.5M"}
	for _, in := range invalid {
		if _, err := ParseISO8601Duration(in); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("ParseISO8601Duration(%q): want ErrInvalidDuration, got %v", in, err)
		}
	}
}

// findParticipant locates a participant row by XUID in a slice, failing the
// test if missing.
func findParticipant(t *testing.T, parts []MatchParticipantRow, xuid string) MatchParticipantRow {
	t.Helper()
	for _, p := range parts {
		if p.XUID == xuid {
			return p
		}
	}
	t.Fatalf("participant %s not found in %d rows", xuid, len(parts))
	return MatchParticipantRow{}
}

func deref[T any](p *T) any {
	if p == nil {
		return "<nil>"
	}
	return *p
}
