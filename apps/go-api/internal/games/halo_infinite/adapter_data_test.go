package halo_infinite

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// stubCareer implémente CareerSource pour tester sans toucher DuckDB.
type stubCareer struct {
	row            *domain.CareerRankData
	err            error
	encounters     []domain.EncounterRawRow
	encountersErr  error
	xpHistory      []domain.XPHistoryPoint
	xpHistoryErr   error
	lusrHistory    []domain.LUSRCheckpointDTO
	lusrHistoryErr error
	topMatches     []domain.TopMatchRawRow
	topMatchesErr  error
}

func (s *stubCareer) GetLatestRank(_ context.Context) (*domain.CareerRankData, error) {
	return s.row, s.err
}

func (s *stubCareer) GetEncounters(_ context.Context) ([]domain.EncounterRawRow, error) {
	return s.encounters, s.encountersErr
}

func (s *stubCareer) GetXPHistory(_ context.Context) ([]domain.XPHistoryPoint, error) {
	return s.xpHistory, s.xpHistoryErr
}

func (s *stubCareer) GetLUSRHistory(_ context.Context) ([]domain.LUSRCheckpointDTO, error) {
	return s.lusrHistory, s.lusrHistoryErr
}

func (s *stubCareer) GetTopMatches(_ context.Context) ([]domain.TopMatchRawRow, error) {
	return s.topMatches, s.topMatchesErr
}

func newSilentAdapter(c CareerSource) *DataAdapter {
	return NewDataAdapter(c, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func TestDataAdapter_TitleSlug(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(nil)
	if a.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q", a.TitleSlug())
	}
}

func TestDataAdapter_Capabilities_NoCareer(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(nil)
	caps := a.Capabilities()
	if caps[games.CapCareerProgression] != games.CapNotExposed {
		t.Errorf("CapCareerProgression sans source = %q, want not_exposed", caps[games.CapCareerProgression])
	}
	if caps.Has(games.CapCareerProgression) {
		t.Errorf("Has(CapCareerProgression) sans source devrait être false")
	}
}

func TestDataAdapter_Capabilities_WithCareer(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(&stubCareer{})
	caps := a.Capabilities()
	if !caps.Has(games.CapCareerProgression) {
		t.Errorf("Has(CapCareerProgression) avec source devrait être true")
	}
	if !caps.Has(games.CapMatchSkillSnapshot) {
		t.Errorf("Has(CapMatchSkillSnapshot=degraded) devrait être true")
	}
	if caps.Has(games.CapTimeseries) {
		t.Errorf("Has(CapTimeseries=not_exposed) devrait être false")
	}
}

func TestLoadCareerSnapshot_NoSource_ReturnsCapabilityError(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(nil)
	_, err := a.LoadCareerSnapshot(context.Background(), "0xABC", canonical.CareerOptions{})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want ErrCapabilityNotSupported", err)
	}
}

func TestLoadCareerSnapshot_NoRows_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	stub := &stubCareer{err: errSentinelNoRows}
	a := newSilentAdapter(stub)
	snap, err := a.LoadCareerSnapshot(context.Background(), "0xABC", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if snap == nil {
		t.Fatal("snapshot nil")
	}
	if snap.Player.XUID != "0xABC" {
		t.Errorf("XUID = %q", snap.Player.XUID)
	}
	if snap.CurrentXP != nil {
		t.Errorf("CurrentXP devrait être nil")
	}
}

func TestLoadCareerSnapshot_HappyPath_Projection(t *testing.T) {
	t.Parallel()
	rankLabel := "Diamond 3"
	rankName := "Diamant 3"
	xpForNext := 1234
	stub := &stubCareer{
		row: &domain.CareerRankData{
			RankNumber:    25,
			CurrentXP:     500,
			RecordedAt:    time.Now(),
			RankLabel:     &rankLabel,
			RankName:      &rankName,
			XPForNextRank: &xpForNext,
		},
	}
	a := newSilentAdapter(stub)
	snap, err := a.LoadCareerSnapshot(context.Background(), "0xABC", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if snap.CurrentXP == nil || *snap.CurrentXP != 500 {
		t.Errorf("CurrentXP = %v, want 500", snap.CurrentXP)
	}
	if snap.XPForNextRank == nil || *snap.XPForNextRank != 1234 {
		t.Errorf("XPForNextRank = %v, want 1234", snap.XPForNextRank)
	}
	if snap.CurrentRank == nil {
		t.Fatal("CurrentRank nil")
	}
	if snap.CurrentRank.Kind != "career_rank" {
		t.Errorf("CurrentRank.Kind = %q", snap.CurrentRank.Kind)
	}
	if snap.CurrentRank.DefaultLabel != "Diamant 3" {
		t.Errorf("DefaultLabel = %q, want Diamant 3", snap.CurrentRank.DefaultLabel)
	}
}

func TestLoadCareerSnapshot_PropagatesUnknownError(t *testing.T) {
	t.Parallel()
	stub := &stubCareer{err: errors.New("connection lost")}
	a := newSilentAdapter(stub)
	_, err := a.LoadCareerSnapshot(context.Background(), "0xABC", canonical.CareerOptions{})
	if err == nil || errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want propagation", err)
	}
}

func TestLoadMatchSummaries_NotImplemented(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(nil)
	_, err := a.LoadMatchSummaries(context.Background(), []string{"m1"})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v", err)
	}
}

func TestLoadTimeseries_NotImplemented(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(&stubCareer{})
	_, err := a.LoadTimeseries(context.Background(), "0xABC", canonical.TimeseriesQuery{})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v", err)
	}
}

func TestLoadPlayerStats_NotImplemented(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(&stubCareer{})
	_, err := a.LoadPlayerStats(context.Background(), "0xABC", canonical.StatsScope{})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v", err)
	}
}

func TestLoadEncounters_NoSource_ReturnsCapabilityError(t *testing.T) {
	t.Parallel()
	a := newSilentAdapter(nil)
	_, err := a.LoadEncounters(context.Background(), "0xABC")
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want ErrCapabilityNotSupported", err)
	}
}

func TestLoadEncounters_HappyPath_Projection(t *testing.T) {
	t.Parallel()
	avg := 1.42
	stub := &stubCareer{
		encounters: []domain.EncounterRawRow{
			{Gamertag: "Ally", XUID: "x1", MatchCount: 10, AsTeammate: 8, AsEnemy: 2, AvgKDA: &avg},
			{Gamertag: "Foe", XUID: "x2", MatchCount: 3, AsTeammate: 1, AsEnemy: 2, AvgKDA: nil},
		},
	}
	a := newSilentAdapter(stub)
	rows, err := a.LoadEncounters(context.Background(), "0xABC")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len = %d, want 2", len(rows))
	}
	if rows[0].Identity.Gamertag != "Ally" || rows[0].Identity.XUID != "x1" {
		t.Errorf("row[0] identity = %+v", rows[0].Identity)
	}
	if rows[0].AvgKDA == nil || *rows[0].AvgKDA != 1.42 {
		t.Errorf("row[0] AvgKDA = %v", rows[0].AvgKDA)
	}
	if rows[1].AvgKDA != nil {
		t.Errorf("row[1] AvgKDA devrait être nil")
	}
}

func TestLoadEncounters_PropagatesError(t *testing.T) {
	t.Parallel()
	stub := &stubCareer{encountersErr: errors.New("connection lost")}
	a := newSilentAdapter(stub)
	_, err := a.LoadEncounters(context.Background(), "0xABC")
	if err == nil || errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want propagation", err)
	}
}

func TestProjectEncounterRow_NilAvgKDA(t *testing.T) {
	t.Parallel()
	row := projectEncounterRow(domain.EncounterRawRow{
		XUID: "x", Gamertag: "GT", MatchCount: 1, AsTeammate: 1, AsEnemy: 0, AvgKDA: nil,
	})
	if row.AvgKDA != nil {
		t.Errorf("AvgKDA devrait être nil")
	}
	if row.Identity.XUID != "x" || row.Identity.Gamertag != "GT" {
		t.Errorf("identity = %+v", row.Identity)
	}
}

func TestProjectCareerSnapshot_RankLabelOnly(t *testing.T) {
	t.Parallel()
	rankLabel := "Iron 1"
	row := &domain.CareerRankData{
		RankNumber: 1,
		CurrentXP:  0,
		RankLabel:  &rankLabel,
	}
	snap := projectCareerSnapshot("0xABC", row)
	if snap.CurrentRank == nil {
		t.Fatal("CurrentRank nil")
	}
	if snap.CurrentRank.ID != "Iron 1" {
		t.Errorf("ID = %q, want Iron 1 (rankID fallback sur RankLabel)", snap.CurrentRank.ID)
	}
}

func TestProjectCareerSnapshot_NilRow(t *testing.T) {
	t.Parallel()
	snap := projectCareerSnapshot("0xXYZ", nil)
	if snap == nil {
		t.Fatal("snapshot nil pour row nil")
	}
	if snap.Player.XUID != "0xXYZ" {
		t.Errorf("Player.XUID = %q", snap.Player.XUID)
	}
	if snap.CurrentRank != nil || snap.CurrentXP != nil {
		t.Errorf("snapshot devrait être minimal pour row nil")
	}
}

func TestRankID_AllVariants(t *testing.T) {
	t.Parallel()
	label := "L"
	name := "N"
	cases := []struct {
		row  *domain.CareerRankData
		want string
	}{
		{&domain.CareerRankData{RankLabel: &label, RankName: &name}, "L"}, // RankLabel prioritaire
		{&domain.CareerRankData{RankLabel: nil, RankName: &name}, "N"},    // fallback RankName
		{&domain.CareerRankData{RankLabel: nil, RankName: nil}, ""},       // vide
	}
	for _, tc := range cases {
		if got := rankID(tc.row); got != tc.want {
			t.Errorf("rankID(%+v) = %q, want %q", tc.row, got, tc.want)
		}
	}
}

func TestStringDeref_NilFallback(t *testing.T) {
	t.Parallel()
	if got := stringDeref(nil, "fallback"); got != "fallback" {
		t.Errorf("nil → %q", got)
	}
	v := "value"
	if got := stringDeref(&v, "fallback"); got != "value" {
		t.Errorf("non-nil → %q", got)
	}
}

func TestIsNoRowsErr(t *testing.T) {
	t.Parallel()
	if isNoRowsErr(nil) {
		t.Errorf("nil ne devrait pas être noRows")
	}
	if !isNoRowsErr(errSentinelNoRows) {
		t.Errorf("sentinel devrait être noRows")
	}
	if isNoRowsErr(errors.New("other")) {
		t.Errorf("autre erreur ne devrait pas être noRows")
	}
}

func TestNewDataAdapter_NilLogger_UsesDefault(t *testing.T) {
	t.Parallel()
	a := NewDataAdapter(nil, nil)
	if a == nil {
		t.Fatal("adapter nil")
	}
	if a.logger == nil {
		t.Errorf("logger devrait avoir été remplacé par slog.Default()")
	}
}
