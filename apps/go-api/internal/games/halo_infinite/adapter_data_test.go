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
	row *domain.CareerRankData
	err error
}

func (s *stubCareer) GetLatestRank(_ context.Context) (*domain.CareerRankData, error) {
	return s.row, s.err
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
