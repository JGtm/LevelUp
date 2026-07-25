// Package synthetic_title_b — Phase E du plan multi-titres.
//
// Ce package n'est PAS un titre réel. Il sert uniquement à :
//
//  1. démontrer que l'interface games.TitleDataAdapter et games.TitleSemantic
//     Adapter peuvent être implémentées par un autre titre que halo_infinite ;
//  2. couvrir des tests d'isolation cross-titres (un service produit ne doit
//     contenir aucun code-path conditionnel sur title_slug, à part le bootstrap) ;
//  3. valider la conversion d'unités (ce titre stocke ses durées en
//     millisecondes côté DuckDB et expose des secondes au canonique).
//
// Aucun fichier de ce package n'est référencé par le runtime de production.
package synthetic_title_b

import (
	"context"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// TitleSlug est le slug canonique de ce titre synthétique.
const TitleSlug = "synthetic_title_b"

// FakePlayerStats permet aux tests d'isolation de configurer ce que renvoie
// l'adapter sans dépendre d'une DuckDB.
type FakePlayerStats struct {
	XUID          string
	MatchesPlayed int
	Kills         int
	Deaths        int
}

// DataAdapter est l'implémentation games.TitleDataAdapter pour ce titre.
type DataAdapter struct {
	stats *FakePlayerStats
}

// NewDataAdapter construit un DataAdapter avec des stats injectées.
//
// Les méthodes Load* sont stubées : ce package n'est pas un titre réel.
// Le seul rôle ici est de prouver l'agnosticité des services aval vis-à-vis
// du titre courant.
func NewDataAdapter(stats *FakePlayerStats) *DataAdapter {
	return &DataAdapter{stats: stats}
}

// TitleSlug retourne synthetic_title_b.
func (a *DataAdapter) TitleSlug() string { return TitleSlug }

// Capabilities expose une map qui prouve que le DataAdapter peut décrire
// un autre profil de capabilities qu'Halo Infinite.
func (a *DataAdapter) Capabilities() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapMatchHistory:       games.CapSupported,
		games.CapMatchDetailCore:    games.CapSupported,
		games.CapMatchSkillSnapshot: games.CapNotExposed, // ce titre n'a pas de skill natif
		games.CapCareerProgression:  games.CapNotExposed,
		games.CapPveFirefight:       games.CapNotExposed,
		games.CapTimeseries:         games.CapNotExposed,
		games.CapEngagement:         games.CapNotExposed, // pas d'engagement sur ce titre synthetique
	}
}

func (a *DataAdapter) LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error) {
	return []canonical.MatchSummary{}, nil
}

// LoadPlayerStats projette les FakePlayerStats injectées vers le canonique.
func (a *DataAdapter) LoadPlayerStats(ctx context.Context, xuid string, scope canonical.StatsScope) (*canonical.PlayerStats, error) {
	if a.stats == nil {
		return &canonical.PlayerStats{Identity: canonical.PlayerIdentity{XUID: xuid}}, nil
	}
	return &canonical.PlayerStats{
		Identity:      canonical.PlayerIdentity{XUID: a.stats.XUID},
		MatchesPlayed: a.stats.MatchesPlayed,
		Kills:         a.stats.Kills,
		Deaths:        a.stats.Deaths,
	}, nil
}

func (a *DataAdapter) LoadCareerSnapshot(ctx context.Context, xuid string, opts canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadEncounters(ctx context.Context, xuid string) ([]canonical.EncounterRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadLUSRHistory(ctx context.Context, xuid string) ([]canonical.LUSRCheckpoint, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadTopMatches(ctx context.Context, xuid string) ([]canonical.CareerTopMatch, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadTargetRecentMatches(ctx context.Context, xuid string, limit int) ([]canonical.RecentMatchRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadParticipantStats(ctx context.Context, xuid string, matchIDs []string) (*canonical.PlayerMatchSetStats, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadPlayerIntersection(ctx context.Context, selfXUID, otherXUID string) (*canonical.PlayerIntersection, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadTimeseries(ctx context.Context, xuid string, query canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadMatchScoreboard(_ context.Context, _ string) ([]canonical.MatchParticipant, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadHighlightEvents(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadMatchEvents(_ context.Context, _ string, _ canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (a *DataAdapter) LoadFriendsXUIDs(_ context.Context, _ string) ([]string, error) {
	return nil, games.ErrCapabilityNotSupported
}

// SemanticAdapter wrappe un FieldMappingSet chargé depuis fields.toml de ce
// titre synthétique. Il prouve que l'interface games.TitleSemanticAdapter ne
// dépend d'aucun choix titre-spécifique.
type SemanticAdapter struct {
	fields   *mappings.FieldMappingSet
	assets   *mappings.AssetMappingSet
	outcomes *mappings.OutcomeMappingSet
}

// NewSemanticAdapter construit un SemanticAdapter à partir des sets chargés.
// assets et outcomes peuvent être nil — les surfaces produit dégradent.
func NewSemanticAdapter(
	fields *mappings.FieldMappingSet,
	assets *mappings.AssetMappingSet,
	outcomes *mappings.OutcomeMappingSet,
) *SemanticAdapter {
	if fields == nil {
		return nil
	}
	return &SemanticAdapter{fields: fields, assets: assets, outcomes: outcomes}
}

func (a *SemanticAdapter) TitleSlug() string                 { return TitleSlug }
func (a *SemanticAdapter) SchemaVersion() int                { return a.fields.SchemaVersion() }
func (a *SemanticAdapter) Fields() *mappings.FieldMappingSet { return a.fields }

// Ranks retourne un catalog vide : ce titre synthétique n'a pas de progression
// carrière. Les consommateurs dégradent gracieusement (ex: afficher rank_id).
func (a *SemanticAdapter) Ranks() *mappings.RankCatalog {
	return mappings.NewRankCatalog(TitleSlug, nil)
}

// Assets retourne les assets sémantiques chargés (peut être nil).
func (a *SemanticAdapter) Assets() *mappings.AssetMappingSet { return a.assets }

// Outcomes retourne les outcomes sémantiques chargés (peut être nil).
func (a *SemanticAdapter) Outcomes() *mappings.OutcomeMappingSet { return a.outcomes }

// AssetURLAdapter est un stub : ce titre synthétique n'a pas d'assets statiques
// servis sous /static/. Toutes les méthodes retournent "" — les services
// produit consommant cet adapter doivent dégrader gracieusement (cache-aside,
// fallback nil).
type AssetURLAdapter struct{}

// NewAssetURLAdapter construit un AssetURLAdapter stub pour ce titre synthétique.
func NewAssetURLAdapter() *AssetURLAdapter { return &AssetURLAdapter{} }

func (a *AssetURLAdapter) TitleSlug() string                      { return TitleSlug }
func (a *AssetURLAdapter) MapImageURL(_ string) string            { return "" }
func (a *AssetURLAdapter) MedalImageURL(_ uint64) string          { return "" }
func (a *AssetURLAdapter) CSRRankImageURL(_ string, _ int) string { return "" }
func (a *AssetURLAdapter) CSRRankImageURLOnyx() string            { return "" }
func (a *AssetURLAdapter) WeaponImageURL(_ string) string         { return "" }
func (a *AssetURLAdapter) MatchWebURL(_ string) string            { return "" }
func (a *AssetURLAdapter) PlayerMatchWebURL(_, _ string) string   { return "" }
