package service

import (
	"context"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// fakeDetailAdapter implémente games.TitleDataAdapter en dégradant toutes les
// capabilities en ErrCapabilityNotSupported. Sert de base embarquable aux stubs de
// test qui n'ont besoin de servir qu'UNE méthode (ex. h5CareerStubAdapter surcharge
// LoadCareerSnapshot). Rescapé de match_view_canonical_test.go (purgé avec le fallback
// LIVE match view) : seul l'embed dans career_service_test.go l'utilise encore.
type fakeDetailAdapter struct{}

func (f *fakeDetailAdapter) TitleSlug() string                 { return "halo_5" }
func (f *fakeDetailAdapter) Capabilities() games.CapabilityMap { return games.CapabilityMap{} }
func (f *fakeDetailAdapter) LoadPlayerStats(_ context.Context, _ string, _ canonical.StatsScope) (*canonical.PlayerStats, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadCareerSnapshot(_ context.Context, _ string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadMatchSummaries(_ context.Context, _ []string) ([]canonical.MatchSummary, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadEncounters(_ context.Context, _ string) ([]canonical.EncounterRow, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadLUSRHistory(_ context.Context, _ string) ([]canonical.LUSRCheckpoint, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadTopMatches(_ context.Context, _ string) ([]canonical.CareerTopMatch, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadTargetRecentMatches(_ context.Context, _ string, _ int) ([]canonical.RecentMatchRow, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadParticipantStats(_ context.Context, _ string, _ []string) (*canonical.PlayerMatchSetStats, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadPlayerIntersection(_ context.Context, _, _ string) (*canonical.PlayerIntersection, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadTimeseries(_ context.Context, _ string, _ canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadMatchScoreboard(_ context.Context, _ string) ([]canonical.MatchParticipant, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadHighlightEvents(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadMatchEvents(_ context.Context, _ string, _ canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadFriendsXUIDs(_ context.Context, _ string) ([]string, error) {
	return nil, games.ErrCapabilityNotSupported
}
