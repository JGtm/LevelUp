package games

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// stubData implémente TitleDataAdapter pour les tests resolver.
type stubData struct{ slug string }

func (s *stubData) TitleSlug() string           { return s.slug }
func (s *stubData) Capabilities() CapabilityMap { return CapabilityMap{} }
func (s *stubData) LoadMatchSummaries(_ context.Context, _ []string) ([]canonical.MatchSummary, error) {
	return nil, nil
}
func (s *stubData) LoadMatchDetail(_ context.Context, _ string) (*canonical.MatchDetail, error) {
	return nil, nil
}
func (s *stubData) LoadPlayerStats(_ context.Context, _ string, _ canonical.StatsScope) (*canonical.PlayerStats, error) {
	return nil, nil
}
func (s *stubData) LoadCareerSnapshot(_ context.Context, _ string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	return nil, nil
}
func (s *stubData) LoadEncounters(_ context.Context, _ string) ([]canonical.EncounterRow, error) {
	return nil, nil
}
func (s *stubData) LoadTimeseries(_ context.Context, _ string, _ canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, nil
}

// stubSemantic implémente TitleSemanticAdapter pour les tests.
type stubSemantic struct{ slug string }

func (s *stubSemantic) TitleSlug() string                 { return s.slug }
func (s *stubSemantic) SchemaVersion() int                { return 1 }
func (s *stubSemantic) Fields() *mappings.FieldMappingSet { return nil }
func (s *stubSemantic) Ranks() *mappings.RankCatalog      { return mappings.NewRankCatalog(s.slug, nil) }

func TestStaticResolver_DefaultSlug(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver("halo_infinite")
	if r.DefaultSlug() != "halo_infinite" {
		t.Errorf("DefaultSlug = %q", r.DefaultSlug())
	}
	rEmpty := NewStaticResolver("")
	if rEmpty.DefaultSlug() != "halo_infinite" {
		t.Errorf("default DefaultSlug = %q", rEmpty.DefaultSlug())
	}
}

func TestStaticResolver_RegisterAndResolve(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver("halo_infinite")
	r.RegisterData(&stubData{slug: "halo_infinite"})
	r.RegisterSemantic(&stubSemantic{slug: "halo_infinite"})

	d, err := r.Data("halo_infinite")
	if err != nil {
		t.Fatalf("Data err: %v", err)
	}
	if d.TitleSlug() != "halo_infinite" {
		t.Errorf("Data slug = %q", d.TitleSlug())
	}

	s, err := r.Semantic("halo_infinite")
	if err != nil {
		t.Fatalf("Semantic err: %v", err)
	}
	if s.TitleSlug() != "halo_infinite" {
		t.Errorf("Semantic slug = %q", s.TitleSlug())
	}
}

func TestStaticResolver_Unknown(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver("halo_infinite")
	if _, err := r.Data("unknown"); !errors.Is(err, ErrTitleNotResolved) {
		t.Errorf("Data err = %v, want ErrTitleNotResolved", err)
	}
	if _, err := r.Semantic("unknown"); !errors.Is(err, ErrTitleNotResolved) {
		t.Errorf("Semantic err = %v, want ErrTitleNotResolved", err)
	}
}

func TestStaticResolver_Slugs_Union(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver("halo_infinite")
	r.RegisterData(&stubData{slug: "halo_infinite"})
	r.RegisterSemantic(&stubSemantic{slug: "test_title"})
	slugs := r.Slugs()
	if len(slugs) != 2 {
		t.Errorf("Slugs len = %d, want 2 (union)", len(slugs))
	}
}

func TestCapabilityMap_Has(t *testing.T) {
	t.Parallel()
	cm := CapabilityMap{
		CapMatchHistory:       CapSupported,
		CapMatchSkillSnapshot: CapDegraded,
		CapTimeseries:         CapNotExposed,
	}
	if !cm.Has(CapMatchHistory) {
		t.Errorf("Has(supported) = false")
	}
	if !cm.Has(CapMatchSkillSnapshot) {
		t.Errorf("Has(degraded) = false")
	}
	if cm.Has(CapTimeseries) {
		t.Errorf("Has(not_exposed) = true")
	}
	if cm.Has(CapPveFirefight) {
		t.Errorf("Has(absent) = true")
	}
}
