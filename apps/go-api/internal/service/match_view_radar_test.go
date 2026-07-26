package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
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

// TestObjectiveIndexInputFromScoreboard_KeysMatchWeights : garde-rail anti-dérive
// entre le mapping ObjectiveRaw → colonnes (match view) et la table de poids
// narrative — toute clé produite doit exister dans les poids de sa famille, et la
// classification (dont split KOTH/Strongholds par ticks) doit être correcte.
func TestObjectiveIndexInputFromScoreboard_KeysMatchWeights(t *testing.T) {
	t.Parallel()
	ip := func(v int) *int { return &v }
	fp := func(v float64) *float64 { return &v }
	tp := fp(600)

	cases := []struct {
		name string
		obj  domain.ObjectiveRaw
		want narrative.ObjectiveFamily
	}{
		{"ctf", domain.ObjectiveRaw{
			FlagCaptures: ip(1), FlagCaptureAssists: ip(1), FlagGrabs: ip(2), FlagSecures: ip(1),
			FlagSteals: ip(1), FlagReturns: ip(2), FlagCarriersKilled: ip(1), FlagReturnersKilled: ip(1),
			KillsAsFlagCarrier: ip(3), KillsAsFlagReturner: ip(1), TimeAsFlagCarrierSeconds: fp(30),
		}, narrative.FamilyCTF},
		{"strongholds", domain.ObjectiveRaw{
			ZoneCaptures: ip(4), ZoneSecures: ip(2), ZoneOffensiveKills: ip(3), ZoneDefensiveKills: ip(2),
			ZoneScoringTicks: ip(0), TimeInZonesSeconds: fp(100),
		}, narrative.FamilyZonesStrongholds},
		{"koth", domain.ObjectiveRaw{
			ZoneCaptures: ip(4), ZoneSecures: ip(2), ZoneOffensiveKills: ip(3), ZoneDefensiveKills: ip(2),
			ZoneScoringTicks: ip(12), TimeInZonesSeconds: fp(100),
		}, narrative.FamilyZonesKOTH},
		{"oddball", domain.ObjectiveRaw{
			SkullGrabs: ip(3), SkullCarriersKilled: ip(2), SkullScoringTicks: ip(40),
			KillsAsSkullCarrier: ip(1), TimeAsSkullCarrierSeconds: fp(70),
		}, narrative.FamilyOddball},
		{"stockpile", domain.ObjectiveRaw{
			PowerSeedsDeposited: ip(4), PowerSeedsStolen: ip(2), PowerSeedCarriersKilled: ip(2),
			KillsAsPowerSeedCarrier: ip(1), TimeAsPowerSeedCarrierSeconds: fp(40), TimeAsPowerSeedDriverSeconds: fp(20),
		}, narrative.FamilyStockpile},
		{"extraction", domain.ObjectiveRaw{
			ExtractionConversionsCompleted: ip(2), ExtractionConversionsDenied: ip(1),
			ExtractionInitiationsCompleted: ip(3), ExtractionInitiationsDenied: ip(1), SuccessfulExtractions: ip(2),
		}, narrative.FamilyExtraction},
		{"vip", domain.ObjectiveRaw{
			KillsAsVip: ip(2), VipKills: ip(3), VipAssists: ip(1), TimesSelectedAsVip: ip(2),
			MaxKillingSpreeAsVip: ip(3), TimeAsVipSeconds: fp(60),
		}, narrative.FamilyVIP},
	}
	for _, tc := range cases {
		in := objectiveIndexInputFromScoreboard(domain.ScoreboardRaw{Obj: tc.obj, TimePlayed: tp})
		if len(in) != 1 {
			t.Errorf("%s: want 1 famille, got %v", tc.name, in)
			continue
		}
		st, ok := in[tc.want]
		if !ok {
			t.Errorf("%s: famille %s absente de %v", tc.name, tc.want, in)
			continue
		}
		if st.Matches != 1 || st.TimePlayedSeconds != 600 {
			t.Errorf("%s: Matches/TimePlayed = %d/%v, want 1/600", tc.name, st.Matches, st.TimePlayedSeconds)
		}
		weights := narrative.ObjectiveFamilyActionWeights[tc.want]
		for col := range st.ColumnSums {
			if _, known := weights[col]; !known {
				t.Errorf("%s: colonne %q hors table de poids de %s", tc.name, col, tc.want)
			}
		}
	}
	// Slayer (aucun bloc) → nil.
	if in := objectiveIndexInputFromScoreboard(domain.ScoreboardRaw{TimePlayed: tp}); in != nil {
		t.Errorf("slayer: want nil, got %v", in)
	}
}

// TestBuildMatchRadarFromScoreboard_ObjectiveIndex : l'axe Objectif vient de
// l'index par opportunité (P80 → 80/100), est RETIRÉ sur un match sans bloc
// objectif, et le PSA objectiveScore ne sert qu'au résiduel de l'axe Score.
func TestBuildMatchRadarFromScoreboard_ObjectiveIndex(t *testing.T) {
	t.Parallel()
	ip := func(v int) *int { return &v }
	fp := func(v float64) *float64 { return &v }
	axisOf := func(series []MatchViewRadarSeries, axis narrative.ParticipationAxis) *narrative.ParticipationScore {
		for i := range series[0].Axes {
			if series[0].Axes[i].Axis == axis {
				return &series[0].Axes[i]
			}
		}
		return nil
	}

	// Match CTF exactement au P80 (actions 11, hold 0.0434 × 600 s).
	ctfRow := domain.ScoreboardRaw{
		XUID: "x1", Gamertag: "P1", Kills: 10, Deaths: 5, Assists: 2,
		DamageDealt: fp(2500), DamageTaken: fp(2000), PersonalScore: fp(2000), TimePlayed: fp(600),
		Obj: domain.ObjectiveRaw{
			FlagCaptures: ip(1), FlagSteals: ip(1), FlagReturns: ip(2), FlagSecures: ip(1),
			TimeAsFlagCarrierSeconds: fp(0.0434 * 600),
		},
	}
	series := BuildMatchRadarFromScoreboard([]domain.ScoreboardRaw{ctfRow}, "x1", 500, "ctf", 225, 0.90)
	if len(series) != 1 {
		t.Fatalf("want 1 série, got %d", len(series))
	}
	obj := axisOf(series, narrative.AxisObjective)
	if obj == nil {
		t.Fatal("axe objective absent sur un match CTF")
	}
	if obj.Raw < 0.999 || obj.Raw > 1.001 {
		t.Errorf("objective raw: want ~1.0 (P80), got %v", obj.Raw)
	}
	if obj.Value < 79.9 || obj.Value > 80.1 {
		t.Errorf("objective value: want ~80, got %v", obj.Value)
	}
	// Résiduel Score : PSA soustrait (décision 1) — 2000 − 10×100 − 2×50 − 500 = 400.
	if sc := axisOf(series, narrative.AxisScore); sc == nil || sc.Raw != 400 {
		t.Errorf("score residuel: want 400, got %+v", sc)
	}

	// Match sans bloc objectif (Slayer) → axe objective RETIRÉ.
	slayerRow := ctfRow
	slayerRow.Obj = domain.ObjectiveRaw{}
	series = BuildMatchRadarFromScoreboard([]domain.ScoreboardRaw{slayerRow}, "x1", 0, "slayer", 225, 0.90)
	if len(series) != 1 {
		t.Fatalf("want 1 série, got %d", len(series))
	}
	if obj := axisOf(series, narrative.AxisObjective); obj != nil {
		t.Errorf("axe objective sur un match Slayer: want retiré, got %+v", obj)
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

// fakeAwardsRepo mock pour tester loadAwardsForScoreboard.
type fakeAwardsRepo struct {
	rows        []port.PersonalScoreAwardRow
	err         error
	lastFilters port.PersonalScoreAwardsFilters
	lastTitle   string
	callCount   int
}

func (f *fakeAwardsRepo) LoadPersonalScoreAwards(
	_ context.Context,
	slug string,
	filters port.PersonalScoreAwardsFilters,
) ([]port.PersonalScoreAwardRow, error) {
	f.callCount++
	f.lastTitle = slug
	f.lastFilters = filters
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func TestLoadAwardsForScoreboard_QueriesAllXUIDs(t *testing.T) {
	t.Parallel()
	repo := &fakeAwardsRepo{
		rows: []port.PersonalScoreAwardRow{
			{XUID: "x_p1", MatchID: "m1", AwardName: "kills", Total: 10},
			{XUID: "x_p2", MatchID: "m1", AwardName: "assist", Total: 5},
		},
	}
	svc := &MatchViewService{awardsRepo: repo, titleSlug: "halo_infinite"}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "x_p1"},
		{XUID: "x_p2"},
		{XUID: "x_p3"},
	}
	got := svc.loadAwardsForScoreboard(context.Background(), "m1", scoreboard)
	if len(got) != 2 {
		t.Errorf("rows want 2, got %d", len(got))
	}
	// Vérifie que les filters incluent TOUS les xuids du scoreboard
	if len(repo.lastFilters.XUIDs) != 3 {
		t.Errorf("filters.XUIDs want 3 (tous les xuids du scoreboard), got %d",
			len(repo.lastFilters.XUIDs))
	}
	if repo.lastTitle != "halo_infinite" {
		t.Errorf("titleSlug want halo_infinite, got %s", repo.lastTitle)
	}
}

func TestLoadAwardsForScoreboard_NilRepoReturnsNil(t *testing.T) {
	t.Parallel()
	svc := &MatchViewService{awardsRepo: nil}
	got := svc.loadAwardsForScoreboard(context.Background(), "m1", []domain.ScoreboardRaw{
		{XUID: "x_p1"},
	})
	if got != nil {
		t.Errorf("nil repo: want nil, got %v", got)
	}
}

func TestLoadAwardsForScoreboard_EmptyScoreboardReturnsNil(t *testing.T) {
	t.Parallel()
	repo := &fakeAwardsRepo{}
	svc := &MatchViewService{awardsRepo: repo}
	got := svc.loadAwardsForScoreboard(context.Background(), "m1", nil)
	if got != nil {
		t.Errorf("empty scoreboard: want nil, got %v", got)
	}
	if repo.callCount != 0 {
		t.Errorf("repo should not be called when scoreboard empty, got %d calls",
			repo.callCount)
	}
}

func TestLoadAwardsForScoreboard_CapabilityNotSupported_ReturnsNilSilently(t *testing.T) {
	t.Parallel()
	repo := &fakeAwardsRepo{err: games.ErrCapabilityNotSupported}
	svc := &MatchViewService{awardsRepo: repo, titleSlug: "halo_infinite"}
	got := svc.loadAwardsForScoreboard(context.Background(), "m1", []domain.ScoreboardRaw{
		{XUID: "x_p1"},
	})
	if got != nil {
		t.Errorf("capability not supported: want nil, got %v", got)
	}
}
