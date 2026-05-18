package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/openspartan/mapper"
)

func TestToSyncRegistry_DefaultsModeCategoryWhenNil(t *testing.T) {
	m := mapper.MatchRegistryRow{
		MatchID:                 "m1",
		StartTime:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		DurationSeconds:         60,
		PlayableDurationSeconds: 59,
		FirstSyncBy:             "test",
	}
	got := toSyncRegistry(m)
	if got.ModeCategory != "Other" {
		t.Errorf("ModeCategory: want 'Other' default, got %q", got.ModeCategory)
	}
	if got.DurationSeconds == nil || *got.DurationSeconds != 60 {
		t.Errorf("DurationSeconds: want *60, got %v", got.DurationSeconds)
	}
	if got.PlayableDurationSeconds == nil || *got.PlayableDurationSeconds != 59 {
		t.Errorf("PlayableDurationSeconds: want *59, got %v", got.PlayableDurationSeconds)
	}
}

func TestToSyncRegistry_PromotesTeamScoresFromInt16Ptr(t *testing.T) {
	team0 := int16(50)
	team1 := int16(47)
	m := mapper.MatchRegistryRow{
		MatchID:     "m2",
		StartTime:   time.Now(),
		FirstSyncBy: "test",
		Team0Score:  &team0,
		Team1Score:  &team1,
	}
	got := toSyncRegistry(m)
	if got.Team0Score == nil || *got.Team0Score != 50 {
		t.Errorf("Team0Score: want *50, got %v", got.Team0Score)
	}
	if got.Team1Score == nil || *got.Team1Score != 47 {
		t.Errorf("Team1Score: want *47, got %v", got.Team1Score)
	}
}

func TestToSyncRegistry_HonoursExplicitModeCategory(t *testing.T) {
	mc := "BTB"
	m := mapper.MatchRegistryRow{
		MatchID: "m3", StartTime: time.Now(), FirstSyncBy: "test",
		ModeCategory: &mc,
	}
	if got := toSyncRegistry(m); got.ModeCategory != "BTB" {
		t.Errorf("ModeCategory: want 'BTB', got %q", got.ModeCategory)
	}
}

func TestToSyncParticipant_MapsAllScalarFields(t *testing.T) {
	gt := "TestPlayer"
	avg := 46.5
	mmr := 1234.5
	mks := int16(7)
	p := mapper.MatchParticipantRow{
		MatchID: "m", XUID: "x", Gamertag: &gt,
		TeamID: 1, Outcome: 2, Rank: 3,
		Score: 1000, Kills: 19, Deaths: 11, Assists: 4,
		KDA: 9.0, Accuracy: 60.32, ShotsFired: 436, ShotsHit: 263,
		DamageDealt: 4889, DamageTaken: 4159, PersonalScore: 2050,
		TimePlayedSeconds: 600, AvgLifeSeconds: &avg,
		TeamMMR: &mmr, MaxKillingSpree: &mks,
		HeadshotKills: 11, GrenadeKills: 3, MeleeKills: 4, PowerWeaponKills: 1,
	}
	got := toSyncParticipant(p)
	if got.Gamertag == nil || *got.Gamertag != "TestPlayer" {
		t.Errorf("Gamertag: want *TestPlayer, got %v", got.Gamertag)
	}
	if got.TeamID == nil || *got.TeamID != 1 {
		t.Errorf("TeamID: want *1, got %v", got.TeamID)
	}
	if got.Kills == nil || *got.Kills != 19 {
		t.Errorf("Kills: want *19, got %v", got.Kills)
	}
	if got.KDA == nil || *got.KDA != 9.0 {
		t.Errorf("KDA: want *9.0, got %v", got.KDA)
	}
	if got.TeamMMR == nil || *got.TeamMMR != 1234.5 {
		t.Errorf("TeamMMR: want *1234.5, got %v", got.TeamMMR)
	}
	if got.AvgLifeSeconds == nil || *got.AvgLifeSeconds != 46.5 {
		t.Errorf("AvgLifeSeconds: want *46.5, got %v", got.AvgLifeSeconds)
	}
	if got.MaxKillingSpree == nil || *got.MaxKillingSpree != 7 {
		t.Errorf("MaxKillingSpree: want *7, got %v", got.MaxKillingSpree)
	}
	if got.EnemyMMR != nil {
		t.Errorf("EnemyMMR: want nil (not in mapper row), got %v", got.EnemyMMR)
	}
}

func TestToSyncMedals_PreservesIDsAndCounts(t *testing.T) {
	medals := []mapper.MedalEarnedRow{
		{MatchID: "m", XUID: "x1", MedalNameID: 3546244406, Count: 2},
		{MatchID: "m", XUID: "x2", MedalNameID: 622331684, Count: 1},
	}
	got := toSyncMedals(medals)
	if len(got) != 2 {
		t.Fatalf("len: want 2, got %d", len(got))
	}
	if got[0].MedalNameID != 3546244406 {
		t.Errorf("[0] MedalNameID: want 3546244406, got %d", got[0].MedalNameID)
	}
	if got[0].Count != 2 {
		t.Errorf("[0] Count: want 2, got %d", got[0].Count)
	}
}

func TestToAnalysisEvent_ParsesNumericXUID(t *testing.T) {
	xuid := "2533274945467756"
	timeMs := 46832
	typeHint := 50
	row := mapper.HighlightEventRow{
		MatchID: "m", EventType: "kill",
		TimeMs: &timeMs, XUID: &xuid, TypeHint: &typeHint, RawJSON: "{}",
	}
	got := toAnalysisEvent(row)
	if got.EventType != "kill" {
		t.Errorf("EventType: want 'kill', got %q", got.EventType)
	}
	if got.XUID != 2533274945467756 {
		t.Errorf("XUID: want 2533274945467756, got %d", got.XUID)
	}
	if got.TimeMS != 46832 {
		t.Errorf("TimeMS: want 46832, got %d", got.TimeMS)
	}
	if got.TypeHint != 50 {
		t.Errorf("TypeHint: want 50, got %d", got.TypeHint)
	}
}

func TestToAnalysisEvent_NilFieldsBecomeZero(t *testing.T) {
	row := mapper.HighlightEventRow{MatchID: "m", EventType: "kill", RawJSON: "{}"}
	got := toAnalysisEvent(row)
	if got.XUID != 0 || got.TimeMS != 0 || got.TypeHint != 0 {
		t.Errorf("expected zero values for nil pointer fields, got xuid=%d timeMs=%d typeHint=%d",
			got.XUID, got.TimeMS, got.TypeHint)
	}
}
