// Package port définit les interfaces (ports) que les services utilisent.
// Les implémentations concrètes vivent dans internal/platform/.
// Ce package n'importe que des types domaine — jamais de platform/.
package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// BootstrapRepository fournit les données nécessaires à l'endpoint /bootstrap.
// Implémenté par platform/duckdb.BootstrapRepo.
type BootstrapRepository interface {
	// GetMatchCount retourne le nombre total de matchs dans shared_matches_v2.
	GetMatchCount(ctx context.Context) (int, error)

	// GetDBVersion retourne la version DuckDB embarquée.
	GetDBVersion(ctx context.Context) (string, error)
}

// PlayerRepository fournit les données d'un joueur spécifique.
// Implémenté par platform/duckdb.PlayerRepo.
type PlayerRepository interface {
	// XUID retourne l'identifiant Xbox du joueur.
	XUID() string

	// DBPath retourne le chemin absolu vers stats.duckdb du joueur.
	DBPath() string

	// GetInitialSyncDone indique si la sync initiale a été effectuée.
	GetInitialSyncDone(ctx context.Context) (bool, error)
}

// Ensure compile-time interface checks (aucune implémentation "fantôme").
// Les blanks identifiers évitent d'importer platform/ depuis port/.
var (
	_ BootstrapRepository = (*noopBootstrapRepo)(nil)
	_ PlayerRepository    = (*noopPlayerRepo)(nil)
	_ MatchViewRepository = (*noopMatchViewRepo)(nil)
	_ ExplorerRepository  = (*noopExplorerRepo)(nil)
)

// noopBootstrapRepo — impl nulle pour le check de compilation uniquement.
type noopBootstrapRepo struct{}

func (n *noopBootstrapRepo) GetMatchCount(_ context.Context) (int, error)    { return 0, nil }
func (n *noopBootstrapRepo) GetDBVersion(_ context.Context) (string, error)  { return "", nil }

// noopPlayerRepo — impl nulle pour le check de compilation uniquement.
type noopPlayerRepo struct{ xuid, dbPath string }

func (n *noopPlayerRepo) XUID() string                                           { return n.xuid }
func (n *noopPlayerRepo) DBPath() string                                         { return n.dbPath }
func (n *noopPlayerRepo) GetInitialSyncDone(_ context.Context) (bool, error)     { return false, nil }

// FiltersRepository fournit les données pour la résolution des filtres.
// Implémenté par platform/duckdb.FiltersRepo.
type FiltersRepository interface {
	LoadMatchesForFilters(ctx context.Context) ([]domain.FilterMatchRow, error)
	GetMatchCount(ctx context.Context) (int, error)
	GetPlayerMatchCount(ctx context.Context) (int, error)
	GetAvailablePlaylists(ctx context.Context) ([]domain.LabelValue, error)
	GetAvailableMaps(ctx context.Context) ([]domain.LabelValue, error)
}

// MatchHistoryRepository fournit les données pour l'historique des parties.
// Implémenté par platform/duckdb.MatchHistoryRepo.
type MatchHistoryRepository interface {
	LoadAll(ctx context.Context) ([]domain.MatchHistoryRawRow, error)
	LoadMapWinRates(ctx context.Context) (map[string][2]int, error)
}

// CareerRepository fournit les données de progression de carrière.
// Implémenté par platform/duckdb.CareerRepo.
type CareerRepository interface {
	GetLatestRank(ctx context.Context) (*domain.CareerRankData, error)
	GetXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error)
	GetLUSRHistory(ctx context.Context) ([]domain.LUSRCheckpointDTO, error)
	GetTopMatches(ctx context.Context) ([]domain.TopMatchRawRow, error)
	GetEncounters(ctx context.Context) ([]domain.EncounterRawRow, error)
}

// MatchViewRepository fournit toutes les données d'un match pour la vue détail.
// Implémenté par platform/duckdb.MatchViewRepo.
type MatchViewRepository interface {
	// GetMatchMeta retourne les métadonnées du match (Q13).
	GetMatchMeta(ctx context.Context, matchID string) (*domain.MatchMetaRaw, error)

	// GetPlayerMatchStats retourne les stats du joueur pour ce match (Q17).
	GetPlayerMatchStats(ctx context.Context, xuid, matchID string) (*domain.PlayerMatchStatsRaw, error)

	// GetMatchEnrichment retourne l'enrichissement joueur pour ce match (Q18).
	GetMatchEnrichment(ctx context.Context, matchID string) (*domain.MatchEnrichmentRaw, error)

	// GetMatchScoreboard retourne les stats de tous les joueurs (Q12).
	GetMatchScoreboard(ctx context.Context, matchID string) ([]domain.ScoreboardRaw, error)

	// GetMatchMedals retourne les médailles du joueur dans ce match (Q14).
	GetMatchMedals(ctx context.Context, xuid, matchID string) ([]domain.MedalRaw, error)

	// GetMatchEvents retourne les events highlight du match (Q21).
	GetMatchEvents(ctx context.Context, matchID string) ([]domain.EventRaw, error)

	// GetMatchWeaponKills retourne les kills par arme du joueur (Q16).
	GetMatchWeaponKills(ctx context.Context, xuid, matchID string) ([]domain.WeaponKillRaw, error)

	// GetMatchKVPairs retourne les paires killer→victim du match (Q20).
	GetMatchKVPairs(ctx context.Context, matchID string) ([]domain.KVPairRaw, error)
}

// ExplorerRepository fournit les données pour l'explorer.
// Implémenté par platform/duckdb.ExplorerRepo.
type ExplorerRepository interface {
	// GetCommonMatches retourne les matchs joués par 2 joueurs (Q19).
	GetCommonMatches(ctx context.Context, xuid1, xuid2 string) ([]domain.CommonMatchRaw, error)

	// ResolveXUIDByGamertag retourne le XUID pour un gamertag donné.
	ResolveXUIDByGamertag(ctx context.Context, gamertag string) (string, error)
}

// GamertagRepository fournit la recherche de gamertags.
// Implémenté par platform/duckdb.GamertagRepo.
type GamertagRepository interface {
	Search(ctx context.Context, query string) ([]domain.GamertagSearchResult, error)
}

// Ensure compile-time interface checks pour les nouveaux repos.
var (
	_ FiltersRepository      = (*noopFiltersRepo)(nil)
	_ MatchHistoryRepository = (*noopMatchHistoryRepo)(nil)
	_ CareerRepository       = (*noopCareerRepo)(nil)
	_ GamertagRepository     = (*noopGamertagRepo)(nil)
)

// noopFiltersRepo — impl nulle pour le check de compilation uniquement.
type noopFiltersRepo struct{}

func (n *noopFiltersRepo) LoadMatchesForFilters(_ context.Context) ([]domain.FilterMatchRow, error) {
	return nil, nil
}
func (n *noopFiltersRepo) GetMatchCount(_ context.Context) (int, error)          { return 0, nil }
func (n *noopFiltersRepo) GetPlayerMatchCount(_ context.Context) (int, error)    { return 0, nil }
func (n *noopFiltersRepo) GetAvailablePlaylists(_ context.Context) ([]domain.LabelValue, error) {
	return nil, nil
}
func (n *noopFiltersRepo) GetAvailableMaps(_ context.Context) ([]domain.LabelValue, error) {
	return nil, nil
}

// noopMatchHistoryRepo — impl nulle pour le check de compilation uniquement.
type noopMatchHistoryRepo struct{}

func (n *noopMatchHistoryRepo) LoadAll(_ context.Context) ([]domain.MatchHistoryRawRow, error) {
	return nil, nil
}
func (n *noopMatchHistoryRepo) LoadMapWinRates(_ context.Context) (map[string][2]int, error) {
	return nil, nil
}

// noopCareerRepo — impl nulle pour le check de compilation uniquement.
type noopCareerRepo struct{}

func (n *noopCareerRepo) GetLatestRank(_ context.Context) (*domain.CareerRankData, error) {
	return nil, nil
}
func (n *noopCareerRepo) GetXPHistory(_ context.Context) ([]domain.XPHistoryPoint, error) {
	return nil, nil
}
func (n *noopCareerRepo) GetLUSRHistory(_ context.Context) ([]domain.LUSRCheckpointDTO, error) {
	return nil, nil
}
func (n *noopCareerRepo) GetTopMatches(_ context.Context) ([]domain.TopMatchRawRow, error) {
	return nil, nil
}
func (n *noopCareerRepo) GetEncounters(_ context.Context) ([]domain.EncounterRawRow, error) {
	return nil, nil
}

// noopGamertagRepo — impl nulle pour le check de compilation uniquement.
type noopGamertagRepo struct{}

func (n *noopGamertagRepo) Search(_ context.Context, _ string) ([]domain.GamertagSearchResult, error) {
	return nil, nil
}

// noopMatchViewRepo — impl nulle pour le check de compilation uniquement.
type noopMatchViewRepo struct{}

func (n *noopMatchViewRepo) GetMatchMeta(_ context.Context, _ string) (*domain.MatchMetaRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetPlayerMatchStats(_ context.Context, _, _ string) (*domain.PlayerMatchStatsRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchEnrichment(_ context.Context, _ string) (*domain.MatchEnrichmentRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchScoreboard(_ context.Context, _ string) ([]domain.ScoreboardRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchMedals(_ context.Context, _, _ string) ([]domain.MedalRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchEvents(_ context.Context, _ string) ([]domain.EventRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchWeaponKills(_ context.Context, _, _ string) ([]domain.WeaponKillRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchKVPairs(_ context.Context, _ string) ([]domain.KVPairRaw, error) {
	return nil, nil
}

// noopExplorerRepo — impl nulle pour le check de compilation uniquement.
type noopExplorerRepo struct{}

func (n *noopExplorerRepo) GetCommonMatches(_ context.Context, _, _ string) ([]domain.CommonMatchRaw, error) {
	return nil, nil
}
func (n *noopExplorerRepo) ResolveXUIDByGamertag(_ context.Context, _ string) (string, error) {
	return "", nil
}

// SessionsRepository fournit les données brutes pour le calcul des sessions.
// Implémenté par platform/duckdb.SessionsRepo.
type SessionsRepository interface {
	// LoadSessionMatches retourne les matchs d'un joueur pour le calcul des sessions (Q22).
	LoadSessionMatches(ctx context.Context) ([]domain.SessionMatchRow, error)
}

// StatsRepository fournit les données pour les séries temporelles et le perf score.
// Implémenté par platform/duckdb.StatsRepo.
type StatsRepository interface {
	// LoadStatsMatches retourne tous les matchs du joueur avec les métriques analytiques (Q23).
	LoadStatsMatches(ctx context.Context) ([]domain.StatsMatchRow, error)

	// LoadLUSRHistory retourne le rating LUSR par match depuis match_skill_rank (Q24).
	LoadLUSRHistory(ctx context.Context) ([]domain.LUSRMatchRating, error)

	// LoadMatchParticipants retourne les participants de tous les matchs du joueur (Q25).
	LoadMatchParticipants(ctx context.Context) ([]domain.ParticipantRow, error)
}

// Ensure compile-time checks pour Sessions et Stats.
var (
	_ SessionsRepository = (*noopSessionsRepo)(nil)
	_ StatsRepository    = (*noopStatsRepo)(nil)
)

// noopSessionsRepo — impl nulle pour le check de compilation uniquement.
type noopSessionsRepo struct{}

func (n *noopSessionsRepo) LoadSessionMatches(_ context.Context) ([]domain.SessionMatchRow, error) {
	return nil, nil
}

// noopStatsRepo — impl nulle pour le check de compilation uniquement.
type noopStatsRepo struct{}

func (n *noopStatsRepo) LoadStatsMatches(_ context.Context) ([]domain.StatsMatchRow, error) {
	return nil, nil
}
func (n *noopStatsRepo) LoadLUSRHistory(_ context.Context) ([]domain.LUSRMatchRating, error) {
	return nil, nil
}
func (n *noopStatsRepo) LoadMatchParticipants(_ context.Context) ([]domain.ParticipantRow, error) {
	return nil, nil
}

// ConfigProvider fournit la configuration de l'application.
type ConfigProvider interface {
	// GetPlayers retourne la liste des joueurs configurés dans db_profiles.json.
	GetPlayers() ([]domain.PlayerSummary, error)

	// GetAppSettings retourne les paramètres de l'application depuis app_settings.json.
	GetAppSettings() (map[string]interface{}, error)

	// IsDemoMode retourne true si le mode démo est activé.
	IsDemoMode() bool
}
