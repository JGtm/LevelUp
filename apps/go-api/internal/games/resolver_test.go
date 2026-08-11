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
func (s *stubData) LoadPlayerStats(_ context.Context, _ string, _ canonical.StatsScope) (*canonical.PlayerStats, error) {
	return nil, nil
}
func (s *stubData) LoadCareerSnapshot(_ context.Context, _ string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	return nil, nil
}
func (s *stubData) LoadEncounters(_ context.Context, _ string) ([]canonical.EncounterRow, error) {
	return nil, nil
}
func (s *stubData) LoadLUSRHistory(_ context.Context, _ string) ([]canonical.LUSRCheckpoint, error) {
	return nil, nil
}
func (s *stubData) LoadTopMatches(_ context.Context, _ string) ([]canonical.CareerTopMatch, error) {
	return nil, nil
}
func (s *stubData) LoadTargetRecentMatches(_ context.Context, _ string, _ int) ([]canonical.RecentMatchRow, error) {
	return nil, nil
}
func (s *stubData) LoadParticipantStats(_ context.Context, _ string, _ []string) (*canonical.PlayerMatchSetStats, error) {
	return nil, nil
}
func (s *stubData) LoadPlayerIntersection(_ context.Context, _, _ string) (*canonical.PlayerIntersection, error) {
	return nil, nil
}
func (s *stubData) LoadTimeseries(_ context.Context, _ string, _ canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, nil
}
func (s *stubData) LoadMatchScoreboard(_ context.Context, _ string) ([]canonical.MatchParticipant, error) {
	return nil, ErrCapabilityNotSupported
}
func (s *stubData) LoadHighlightEvents(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return nil, ErrCapabilityNotSupported
}
func (s *stubData) LoadMatchEvents(_ context.Context, _ string, _ canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	return nil, ErrCapabilityNotSupported
}
func (s *stubData) LoadFriendsXUIDs(_ context.Context, _ string) ([]string, error) {
	return nil, ErrCapabilityNotSupported
}

// stubSemantic implémente TitleSemanticAdapter pour les tests.
type stubSemantic struct{ slug string }

func (s *stubSemantic) TitleSlug() string                     { return s.slug }
func (s *stubSemantic) SchemaVersion() int                    { return 1 }
func (s *stubSemantic) Fields() *mappings.FieldMappingSet     { return nil }
func (s *stubSemantic) Ranks() *mappings.RankCatalog          { return mappings.NewRankCatalog(s.slug, nil) }
func (s *stubSemantic) Assets() *mappings.AssetMappingSet     { return nil }
func (s *stubSemantic) Outcomes() *mappings.OutcomeMappingSet { return nil }

// stubAssetURL implémente TitleAssetURLAdapter pour les tests resolver.
type stubAssetURL struct{ slug string }

func (s *stubAssetURL) TitleSlug() string                      { return s.slug }
func (s *stubAssetURL) MapImageURL(_ string) string            { return "" }
func (s *stubAssetURL) MedalImageURL(_ uint64) string          { return "" }
func (s *stubAssetURL) CSRRankImageURL(_ string, _ int) string { return "" }
func (s *stubAssetURL) CSRRankImageURLOnyx() string            { return "" }
func (s *stubAssetURL) WeaponImageURL(_ int64) string          { return "" }
func (s *stubAssetURL) WeaponImageIsTinted(_ int64) bool       { return false }
func (s *stubAssetURL) MatchWebURL(_ string) string            { return "" }
func (s *stubAssetURL) PlayerMatchWebURL(_, _ string) string   { return "" }

// stubCatalog implémente TitleCatalogAdapter pour les tests resolver.
type stubCatalog struct{ slug string }

func (s *stubCatalog) TitleSlug() string { return s.slug }
func (s *stubCatalog) FetchPlaylist(_ context.Context, _, _ string) (canonical.CanonicalPlaylist, error) {
	return canonical.CanonicalPlaylist{}, nil
}
func (s *stubCatalog) FetchPair(_ context.Context, _, _ string) (canonical.CanonicalPair, error) {
	return canonical.CanonicalPair{}, nil
}
func (s *stubCatalog) FetchMap(_ context.Context, _, _ string) (canonical.CanonicalMap, error) {
	return canonical.CanonicalMap{}, nil
}
func (s *stubCatalog) FetchGameVariant(_ context.Context, _, _ string) (canonical.CanonicalGameVariant, error) {
	return canonical.CanonicalGameVariant{}, nil
}
func (s *stubCatalog) ClassifyExperience(_ canonical.CanonicalPlaylist) canonical.Experience {
	return canonical.ExperienceUnknown
}

// hiSlug constante locale aux tests pour réutiliser le littéral du titre
// par défaut sans déclencher goconst sur les multiples occurrences.
const hiSlug = "halo_infinite"

func TestStaticResolver_DefaultSlug(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver(hiSlug)
	if r.DefaultSlug() != hiSlug {
		t.Errorf("DefaultSlug = %q", r.DefaultSlug())
	}
	rEmpty := NewStaticResolver("")
	if rEmpty.DefaultSlug() != hiSlug {
		t.Errorf("default DefaultSlug = %q", rEmpty.DefaultSlug())
	}
}

func TestStaticResolver_RegisterAndResolve(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver(hiSlug)
	r.RegisterData(&stubData{slug: hiSlug})
	r.RegisterSemantic(&stubSemantic{slug: hiSlug})
	r.RegisterAssetURL(&stubAssetURL{slug: hiSlug})

	d, err := r.Data(hiSlug)
	if err != nil {
		t.Fatalf("Data err: %v", err)
	}
	if d.TitleSlug() != hiSlug {
		t.Errorf("Data slug = %q", d.TitleSlug())
	}

	s, err := r.Semantic(hiSlug)
	if err != nil {
		t.Fatalf("Semantic err: %v", err)
	}
	if s.TitleSlug() != hiSlug {
		t.Errorf("Semantic slug = %q", s.TitleSlug())
	}

	a, err := r.AssetURL(hiSlug)
	if err != nil {
		t.Fatalf("AssetURL err: %v", err)
	}
	if a.TitleSlug() != hiSlug {
		t.Errorf("AssetURL slug = %q", a.TitleSlug())
	}

	// Phase C plan catalogue : Catalog adapter résolu correctement.
	r.RegisterCatalog(&stubCatalog{slug: hiSlug})
	c, err := r.Catalog(hiSlug)
	if err != nil {
		t.Fatalf("Catalog err: %v", err)
	}
	if c.TitleSlug() != hiSlug {
		t.Errorf("Catalog slug = %q", c.TitleSlug())
	}
}

// Phase C plan catalogue : isolation cross-titres pour TitleCatalogAdapter.
// Un appel Catalog("synthetic_title_b") ne doit jamais router vers Halo.
func TestStaticResolver_Catalog_Isolation(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver(hiSlug)
	r.RegisterCatalog(&stubCatalog{slug: hiSlug})
	r.RegisterCatalog(&stubCatalog{slug: "synthetic_title_b"})

	cHalo, err := r.Catalog(hiSlug)
	if err != nil {
		t.Fatalf("Catalog(halo): %v", err)
	}
	if cHalo.TitleSlug() != hiSlug {
		t.Errorf("Catalog(halo).TitleSlug = %q", cHalo.TitleSlug())
	}

	cSynth, err := r.Catalog("synthetic_title_b")
	if err != nil {
		t.Fatalf("Catalog(synth): %v", err)
	}
	if cSynth.TitleSlug() != "synthetic_title_b" {
		t.Errorf("Catalog(synth).TitleSlug = %q (cross-title leak)", cSynth.TitleSlug())
	}

	// Catalog inconnu → ErrTitleNotResolved (pas de fallback silencieux).
	if _, err := r.Catalog("nonexistent"); !errors.Is(err, ErrTitleNotResolved) {
		t.Errorf("Catalog(unknown) err = %v, want ErrTitleNotResolved", err)
	}
}

func TestStaticResolver_Unknown(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver(hiSlug)
	if _, err := r.Data("unknown"); !errors.Is(err, ErrTitleNotResolved) {
		t.Errorf("Data err = %v, want ErrTitleNotResolved", err)
	}
	if _, err := r.Semantic("unknown"); !errors.Is(err, ErrTitleNotResolved) {
		t.Errorf("Semantic err = %v, want ErrTitleNotResolved", err)
	}
	if _, err := r.AssetURL("unknown"); !errors.Is(err, ErrTitleNotResolved) {
		t.Errorf("AssetURL err = %v, want ErrTitleNotResolved", err)
	}
}

func TestStaticResolver_Slugs_Union(t *testing.T) {
	t.Parallel()
	r := NewStaticResolver(hiSlug)
	r.RegisterData(&stubData{slug: hiSlug})
	r.RegisterSemantic(&stubSemantic{slug: "test_title"})
	r.RegisterAssetURL(&stubAssetURL{slug: "third_title"})
	slugs := r.Slugs()
	if len(slugs) != 3 {
		t.Errorf("Slugs len = %d, want 3 (union)", len(slugs))
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
