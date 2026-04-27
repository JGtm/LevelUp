package service

import (
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func awardRow(xuid, name string, total int) port.PersonalScoreAwardRow {
	return port.PersonalScoreAwardRow{
		XUID:      xuid,
		MatchID:   "m1",
		AwardName: name,
		Total:     total,
	}
}

func TestBuildMatchRadar_AggregatesByAxis(t *testing.T) {
	t.Parallel()
	awards := []port.PersonalScoreAwardRow{
		awardRow("x_p1", "kills", 10),
		awardRow("x_p1", "double_kill", 2),
		awardRow("x_p1", "killing_spree", 1), // impact
		awardRow("x_p1", "assist", 5),        // support
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "x_p1", Gamertag: "PlayerOne"},
	}

	series := BuildMatchRadar(awards, scoreboard, "slayer")
	if len(series) != 1 {
		t.Fatalf("want 1 series, got %d", len(series))
	}
	s := series[0]
	if s.XUID != "x_p1" {
		t.Errorf("XUID want x_p1, got %s", s.XUID)
	}
	if s.Gamertag != "PlayerOne" {
		t.Errorf("Gamertag want PlayerOne, got %s", s.Gamertag)
	}
	if len(s.Axes) != 6 {
		t.Fatalf("want 6 axes, got %d", len(s.Axes))
	}

	// Vérifier que Combat aggregé = kills (10) + double_kill (2) = 12
	for _, ax := range s.Axes {
		if ax.Axis == narrative.AxisCombat && ax.Raw != 12 {
			t.Errorf("Combat raw want 12, got %f", ax.Raw)
		}
		if ax.Axis == narrative.AxisSupport && ax.Raw != 5 {
			t.Errorf("Support raw want 5, got %f", ax.Raw)
		}
		if ax.Axis == narrative.AxisImpact && ax.Raw != 1 {
			t.Errorf("Impact raw want 1, got %f", ax.Raw)
		}
	}
}

func TestBuildMatchRadar_MultiplePlayersStableOrder(t *testing.T) {
	t.Parallel()
	awards := []port.PersonalScoreAwardRow{
		awardRow("x_p1", "kills", 5),
		awardRow("x_p2", "kills", 8),
		awardRow("x_p3", "kills", 3),
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "x_p2", Gamertag: "Two"},
		{XUID: "x_p1", Gamertag: "One"},
		{XUID: "x_p3", Gamertag: "Three"},
	}
	series := BuildMatchRadar(awards, scoreboard, "")
	if len(series) != 3 {
		t.Fatalf("want 3 series, got %d", len(series))
	}
	// L'ordre doit suivre le scoreboard (Two, One, Three)
	if series[0].XUID != "x_p2" || series[1].XUID != "x_p1" || series[2].XUID != "x_p3" {
		t.Errorf("ordre series want [x_p2, x_p1, x_p3], got [%s, %s, %s]",
			series[0].XUID, series[1].XUID, series[2].XUID)
	}
}

func TestBuildMatchRadar_UnknownAwardsIgnored(t *testing.T) {
	t.Parallel()
	awards := []port.PersonalScoreAwardRow{
		awardRow("x_p1", "kills", 10),
		awardRow("x_p1", "totally_unknown_award", 999), // doit être ignoré
	}
	scoreboard := []domain.ScoreboardRaw{{XUID: "x_p1"}}
	series := BuildMatchRadar(awards, scoreboard, "")
	if len(series) != 1 {
		t.Fatalf("want 1 series, got %d", len(series))
	}
	// Combat = 10 (kills), pas 1009 — l'unknown doit être skipped
	for _, ax := range series[0].Axes {
		if ax.Axis == narrative.AxisCombat && ax.Raw != 10 {
			t.Errorf("Combat raw want 10 (unknown ignoré), got %f", ax.Raw)
		}
	}
}

func TestBuildMatchRadar_PlayerWithoutMappedAwardSkipped(t *testing.T) {
	t.Parallel()
	awards := []port.PersonalScoreAwardRow{
		awardRow("x_p1", "kills", 5),
		awardRow("x_p2", "totally_unknown", 999),
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "x_p1"},
		{XUID: "x_p2"}, // ne devrait pas apparaître (que des awards inconnus)
	}
	series := BuildMatchRadar(awards, scoreboard, "")
	if len(series) != 1 {
		t.Fatalf("want 1 series (x_p2 sans award mappé), got %d", len(series))
	}
	if series[0].XUID != "x_p1" {
		t.Errorf("XUID want x_p1, got %s", series[0].XUID)
	}
}

func TestBuildMatchRadar_EmptyInputs(t *testing.T) {
	t.Parallel()
	if got := BuildMatchRadar(nil, nil, ""); got != nil {
		t.Errorf("nil awards: want nil, got %v", got)
	}
}

func TestMatchModeFamilyFromMeta(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pairName string
		want     string
	}{
		{"Team Slayer", "slayer"},
		{"CTF Big Team", "ctf"},
		{"Strongholds Ranked", "strongholds"},
		{"Oddball Pro", "oddball"},
		{"Neutral Bomb", "oddball"},
		{"Custom Game", ""},
	}
	for _, tc := range cases {
		name := tc.pairName
		meta := &domain.MatchMetaRaw{PairName: &name}
		got := matchModeFamilyFromMeta(meta)
		if got != tc.want {
			t.Errorf("matchModeFamilyFromMeta(%q) want %q, got %q", tc.pairName, tc.want, got)
		}
	}
}

func TestMatchModeFamilyFromMeta_NilHandling(t *testing.T) {
	t.Parallel()
	if got := matchModeFamilyFromMeta(nil); got != "" {
		t.Errorf("nil meta: want empty, got %q", got)
	}
	if got := matchModeFamilyFromMeta(&domain.MatchMetaRaw{}); got != "" {
		t.Errorf("nil pair: want empty, got %q", got)
	}
}
