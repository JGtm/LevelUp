// Package port définit les interfaces (ports) que les services utilisent.
// Les implémentations concrètes vivent dans internal/platform/.
// Ce package n'importe que des types domaine — jamais de platform/.
package port

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BattlePassCacheRepository fournit les données Battle Pass et Challenges depuis le cache DB.
// Implémenté par platform/duckdb.HomeRepo.
type BattlePassCacheRepository interface {
	// LoadCachedBattlePass retourne les données BP si une entrée is_current existe
	// et a été vue il y a moins de ttl. Retourne (nil, false, nil) si pas en cache.
	LoadCachedBattlePass(ctx context.Context, ttl time.Duration) (*domain.BattlePassResponse, bool, error)

	// LoadCachedChallenges retourne un résumé des snapshots récents du joueur
	// si des entrées existent dans la fenêtre ttl. Retourne (nil, false, nil) si pas en cache.
	LoadCachedChallenges(ctx context.Context, ttl time.Duration) (*domain.ChallengesResponse, bool, error)
}

// SeasonPassRepository fournit les tracks Battle Pass persistées.
// Implémenté par platform/duckdb.SeasonPassRepo.
type SeasonPassRepository interface {
	// LoadSeasonPassTracks retourne toutes les tracks connues (is_current ou historique)
	// triées par last_seen_at DESC.
	LoadSeasonPassTracks(ctx context.Context, xuid, titleSlug string) ([]domain.SeasonPassTrackSummary, error)
}

// BootstrapRepository fournit les données nécessaires à l'endpoint /bootstrap.
// Implémenté par platform/duckdb.BootstrapRepo.
type BootstrapRepository interface {
	// GetMatchCount retourne le nombre total de matchs dans shared_matches_v2.
	GetMatchCount(ctx context.Context) (int, error)

	// GetDBVersion retourne la version DuckDB embarquée.
	GetDBVersion(ctx context.Context) (string, error)

	// Sprint 41 T3 : méthodes enrichissement healthcheck.
	GetPlayerCount(ctx context.Context) (int, error)
	GetLastSyncAt(ctx context.Context) (*time.Time, error)
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

func (n *noopBootstrapRepo) GetMatchCount(_ context.Context) (int, error)        { return 0, nil }
func (n *noopBootstrapRepo) GetDBVersion(_ context.Context) (string, error)      { return "", nil }
func (n *noopBootstrapRepo) GetPlayerCount(_ context.Context) (int, error)       { return 0, nil }
func (n *noopBootstrapRepo) GetLastSyncAt(_ context.Context) (*time.Time, error) { return nil, nil }

// noopPlayerRepo — impl nulle pour le check de compilation uniquement.
type noopPlayerRepo struct{ xuid, dbPath string }

func (n *noopPlayerRepo) XUID() string                                       { return n.xuid }
func (n *noopPlayerRepo) DBPath() string                                     { return n.dbPath }
func (n *noopPlayerRepo) GetInitialSyncDone(_ context.Context) (bool, error) { return false, nil }

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

	// GetMatchNeighbors retourne les matchs précédent/suivant (Q25).
	GetMatchNeighbors(ctx context.Context, xuid, matchID string) (*domain.MatchNeighbors, error)

	// GetMatchSkillRank retourne le rang compétitif pour ce match (Q22).
	GetMatchSkillRank(ctx context.Context, matchID string) (*domain.SkillRankRaw, error)

	// GetMatchEncounters retourne l'historique de rencontres avec les participants (Q23).
	GetMatchEncounters(ctx context.Context, matchID, myXUID string) ([]domain.EncounterRaw, error)

	// GetMatchEncounterStats retourne les stats riches par encounter (Q23b,
	// chunk MV4.C'). Permet narrative.ComputeEncounterBadges (ally_plus +
	// tough_enemy). Optionnel : implémentations qui ne supportent pas peuvent
	// retourner (nil, nil) — le service dégrade gracieusement (badge ordinal
	// seul attribué).
	GetMatchEncounterStats(ctx context.Context, matchID, myXUID string) ([]domain.EncounterStatsRaw, error)

	// GetMatchMedia retourne les médias associés au match (Q24).
	GetMatchMedia(ctx context.Context, matchID, playerSlug string) ([]domain.MediaAssocRaw, error)

	// GetMatchExpectedStats retourne les stats attendues pour ce match (Q26).
	GetMatchExpectedStats(ctx context.Context, matchID, xuid string) (*domain.ExpectedStatsRaw, error)

	// GetMatchBulkMedals retourne les médailles de tous les joueurs du match (Q27).
	GetMatchBulkMedals(ctx context.Context, matchID string) ([]domain.BulkMedalRaw, error)

	// GetMatchBulkWeaponKills retourne les kills par arme de tous les joueurs (Q28).
	GetMatchBulkWeaponKills(ctx context.Context, matchID string) ([]domain.BulkWeaponKillRaw, error)
}

// ExplorerRepository fournit les données pour l'explorer.
// Implémenté par platform/duckdb.ExplorerRepo.
type ExplorerRepository interface {
	// GetCommonMatches retourne les matchs joués par 2 joueurs (Q19).
	GetCommonMatches(ctx context.Context, xuid1, xuid2 string) ([]domain.CommonMatchRaw, error)

	// GetKillerVictimBetween retourne les kills croisés agrégés entre xuid1 et xuid2.
	GetKillerVictimBetween(ctx context.Context, xuid1, xuid2 string) (domain.KillerVictimAggregate, error)

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
func (n *noopFiltersRepo) GetMatchCount(_ context.Context) (int, error)       { return 0, nil }
func (n *noopFiltersRepo) GetPlayerMatchCount(_ context.Context) (int, error) { return 0, nil }
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
func (n *noopMatchViewRepo) GetMatchNeighbors(_ context.Context, _, _ string) (*domain.MatchNeighbors, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchSkillRank(_ context.Context, _ string) (*domain.SkillRankRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchEncounters(_ context.Context, _, _ string) ([]domain.EncounterRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchEncounterStats(_ context.Context, _, _ string) ([]domain.EncounterStatsRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchMedia(_ context.Context, _, _ string) ([]domain.MediaAssocRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchExpectedStats(_ context.Context, _, _ string) (*domain.ExpectedStatsRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchBulkMedals(_ context.Context, _ string) ([]domain.BulkMedalRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchBulkWeaponKills(_ context.Context, _ string) ([]domain.BulkWeaponKillRaw, error) {
	return nil, nil
}

// noopExplorerRepo — impl nulle pour le check de compilation uniquement.
type noopExplorerRepo struct{}

func (n *noopExplorerRepo) GetCommonMatches(_ context.Context, _, _ string) ([]domain.CommonMatchRaw, error) {
	return nil, nil
}
func (n *noopExplorerRepo) GetKillerVictimBetween(_ context.Context, _, _ string) (domain.KillerVictimAggregate, error) {
	return domain.KillerVictimAggregate{}, nil
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
//
// P4.3 finale : LoadStatsMatches a été retiré (les services chargent canonical
// via PlayerMatchesRepository).
type StatsRepository interface {
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

func (n *noopMediaRepo) SetMediaLike(_ context.Context, _ string, _ bool) (bool, error) {
	return false, nil
}
func (n *noopMediaRepo) ToggleSharedLike(_ context.Context, _, _, _ string, _ bool) error { return nil }
func (n *noopMediaRepo) GetMediaLikers(_ context.Context, _ []string) (map[string]domain.MediaLikersInfo, error) {
	return nil, nil
}
func (n *noopMediaRepo) CurrentPlayerSlug() string { return "" }
func (n *noopMediaRepo) LoadMatchCandidatesForMedia(_ context.Context, _ string, _ int) (domain.MediaMatchCandidatesResponse, error) {
	return domain.MediaMatchCandidatesResponse{}, nil
}
func (n *noopMediaRepo) SetMediaMatchAssociation(_ context.Context, _, _ string) (*string, *string, error) {
	return nil, nil, nil
}

// noopSessionsRepo — impl nulle pour le check de compilation uniquement.
type noopSessionsRepo struct{}

func (n *noopSessionsRepo) LoadSessionMatches(_ context.Context) ([]domain.SessionMatchRow, error) {
	return nil, nil
}

// noopStatsRepo — impl nulle pour le check de compilation uniquement.
type noopStatsRepo struct{}

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

// HomeRepository fournit les données pour la page d'accueil Mission Control.
// Implémenté par platform/duckdb.HomeRepo.
//
// P4.3 finale : LoadHomeMatches/LoadHomeSessions ont été retirés (le service
// charge canonical via PlayerMatchesRepository et dérive les sessions depuis
// les enrichments).
type HomeRepository interface {
	// LoadSpartanIdentity charge l'identité record compacte (Spartan ID + rang carrière).
	LoadSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error)

	// CountPlayerMatches retourne le nombre total de matchs du joueur (Q26b).
	CountPlayerMatches(ctx context.Context) (int, error)

	// LoadRecentMedia charge les médias récents du joueur (Q28).
	// Si la table media_files n'existe pas, retourne (nil, nil) sans erreur.
	LoadRecentMedia(ctx context.Context, limit int) ([]domain.HomeMediaRow, error)

	// LoadRecentPlaylistRanks retourne les 3 dernières playlists distinctes avec leur rang (Q26g).
	// Retourne (nil, nil) si aucune donnée disponible.
	LoadRecentPlaylistRanks(ctx context.Context, locale string) ([]domain.HomePlaylistRank, error)

	// LoadMatchMedals charge les médailles du joueur pour un lot de matchs (Q26h).
	// Retourne un map match_id → médailles triées count DESC. Dégradation silencieuse.
	LoadMatchMedals(ctx context.Context, matchIDs []string) (map[string][]domain.RecentMatchMedal, error)

	// LoadMatchCitations charge les citations progressées pour un lot de matchs (Q26i+Q26j).
	// Retourne un map match_id → []HomeMatchCitationRaw. Dégradation silencieuse.
	LoadMatchCitations(ctx context.Context, matchIDs []string) (map[string][]domain.HomeMatchCitationRaw, error)

	// LoadFavoriteWeapon retourne le nom localisé et les kills totaux de l'arme favorite (Q26k).
	// Dégradation silencieuse : retourne ("", 0, nil) si aucune donnée.
	LoadFavoriteWeapon(ctx context.Context, locale string) (string, int, error)

	// EnrichCanonicalAssetTranslations remplit Labels["fr"] des AssetReference
	// (Map, Playlist, GameVariant, PairMode) depuis metadata.asset_translations
	// + mode_name_tr quand absent. Mute les rows en place. Bug #2/#7 cascade.
	EnrichCanonicalAssetTranslations(ctx context.Context, rows []canonical.PlayerMatchRow) error
}

// Ensure compile-time check pour HomeRepository.
var _ HomeRepository = (*noopHomeRepo)(nil)

// noopHomeRepo — impl nulle pour le check de compilation uniquement.
type noopHomeRepo struct{}

func (n *noopHomeRepo) LoadSpartanIdentity(_ context.Context) (*domain.HomeSpartanIdentityRow, error) {
	return nil, nil
}
func (n *noopHomeRepo) CountPlayerMatches(_ context.Context) (int, error) {
	return 0, nil
}
func (n *noopHomeRepo) LoadRecentMedia(_ context.Context, _ int) ([]domain.HomeMediaRow, error) {
	return nil, nil
}
func (n *noopHomeRepo) LoadRecentPlaylistRanks(_ context.Context, _ string) ([]domain.HomePlaylistRank, error) {
	return nil, nil
}

func (n *noopHomeRepo) LoadMatchMedals(_ context.Context, _ []string) (map[string][]domain.RecentMatchMedal, error) {
	return map[string][]domain.RecentMatchMedal{}, nil
}

func (n *noopHomeRepo) LoadMatchCitations(_ context.Context, _ []string) (map[string][]domain.HomeMatchCitationRaw, error) {
	return map[string][]domain.HomeMatchCitationRaw{}, nil
}

func (n *noopHomeRepo) LoadFavoriteWeapon(_ context.Context, _ string) (string, int, error) {
	return "", 0, nil
}

func (n *noopHomeRepo) EnrichCanonicalAssetTranslations(_ context.Context, _ []canonical.PlayerMatchRow) error {
	return nil
}

// SquadRepository fournit les données pour la page Escouade.
// Implémenté par platform/duckdb.SquadRepo.
type SquadRepository interface {
	// LoadTopTeammates charge les coéquipiers les plus fréquents en escouade (Q29, top 50).
	LoadTopTeammates(ctx context.Context, xuid string) ([]domain.TopTeammateRow, error)

	// LookupXUIDByGamertag résout un gamertag (case-insensitive) vers son XUID
	// via shared.xuid_aliases. Fallback pour les gamertags hors top 50 sélectionnés
	// par le user (saisie libre dans la combobox). Retourne ("", false, nil) si
	// le gamertag n'existe pas dans les aliases.
	LookupXUIDByGamertag(ctx context.Context, gamertag string) (xuid string, found bool, err error)

	// LoadSquadMatches charge les matchs communs avec un coéquipier spécifique (Q30).
	LoadSquadMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error)

	// LoadTeammateMatches charge les stats d'un coéquipier sur les matchs communs (Q31).
	LoadTeammateMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.TeammateMatchRow, error)

	// LoadImpactEvents charge les événements highlight pour une liste de match_ids (Q32).
	LoadImpactEvents(ctx context.Context, matchIDs []string) ([]domain.ImpactEventRow, error)

	// LoadMainTeamParticipants charge tous les participants de l'équipe alliée
	// du joueur principal pour une liste de matchs (Q34, scoreboard impact
	// team-wide). Pour chaque match dans matchIDs, retourne les rows
	// match_participants où team_id = team_id du mainXUID dans ce match (le main
	// inclus). Permet à buildSquadImpactMatrix de calculer les badges sur
	// l'équipe alliée complète au lieu du squad sélectionné uniquement.
	LoadMainTeamParticipants(ctx context.Context, mainXUID string, matchIDs []string) ([]domain.AllyParticipant, error)

	// LoadSynthesisHeatmap charge les données heatmap carte × mode (Q33).
	LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error)

	// LoadAssetTranslationsFR retourne les traductions FR depuis metadata.asset_translations.
	// assetType : "map" | "playlist". Retourne nil sans erreur si table absente ou IDs vides.
	LoadAssetTranslationsFR(ctx context.Context, assetType string, assetIDs []string) (map[string]string, error)

	// LoadModeTranslationsFR retourne les traductions FR des modes EN (depuis metadata.mode_name_tr).
	// Les clés sont les noms EN normalisés (ex: "Slayer"), les valeurs les noms FR (ex: "Tueur").
	// Retourne nil sans erreur si la table est absente ou si modeENs est vide.
	LoadModeTranslationsFR(ctx context.Context, modeENs []string) (map[string]string, error)

	// LoadMapStatsForSquad retourne par map_id les stats historiques (wins,
	// total, perf moyenne) du joueur principal sur les matchs où TOUS les xuids
	// du squad sont participants. Aucun filtre temporel — c'est la référence
	// "avec cette escouade exacte" pour le chart Synergies et le tableau de
	// matchs squad. Retourne nil sans erreur si squadXUIDs est vide.
	LoadMapStatsForSquad(ctx context.Context, mainXUID string, squadXUIDs []string) (map[string]domain.MapSquadStats, error)

	// P4.3 finale : LoadSynthesisMatches retiré (squad/teammates chargent
	// canonical via PlayerMatchesRepository).
}

// SynthesisRepository fournit les données pour la page Synthèse (Sprint 55 D1).
// Sous-ensemble de SquadRepository — isolé pour l'injection ciblée.
//
// P4.3 finale : LoadSynthesisMatches retiré.
type SynthesisRepository interface {
	// LoadEncounters charge les encounters du joueur (Q_encounters).
	LoadEncounters(ctx context.Context, xuid string) ([]domain.EncounterRawRow, error)
	// LoadSynthesisHeatmap charge la heatmap carte×mode (Q33).
	LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error)
}

// CitationsRepository fournit les données pour les pages Citations et Commendations.
// Implémenté par platform/duckdb.CitationsRepo.
type CitationsRepository interface {
	// LoadCitationMappings charge les mappings de citations depuis metadata.duckdb (Q34).
	LoadCitationMappings(ctx context.Context) ([]domain.CitationMappingRow, error)

	// LoadCitationTotals charge les totaux agrégés depuis match_citations (Q35).
	LoadCitationTotals(ctx context.Context) ([]domain.CitationTotalRow, error)

	// LoadMedalTotals charge les totaux de médailles depuis shared.medals_earned (Q36a).
	LoadMedalTotals(ctx context.Context, xuid string) ([]domain.MedalEarnedRow, error)

	// LoadMedalCitationMappings charge les mappings médaille→citation depuis metadata (Q36b).
	LoadMedalCitationMappings(ctx context.Context) ([]domain.MedalCitationRow, error)

	// LoadCitationMedalMappings charge les règles citation→medal_id pour le moteur (Q39).
	LoadCitationMedalMappings(ctx context.Context) ([]domain.CitationMedalMapping, error)

	// LoadMatchCitationsForView charge les top citations d'un match pour la vue détail (Q38).
	LoadMatchCitationsForView(ctx context.Context, matchID string) ([]domain.CitationMatchViewRow, error)

	// WriteCitationsForMatch écrit les deltas calculés dans match_citations.
	WriteCitationsForMatch(ctx context.Context, matchID string, deltas []domain.CitationMatchDelta) error
}

// MatchExclusionRepository gère le flag is_excluded dans player_match_enrichment.
// Implémenté par platform/duckdb.MatchExclusionRepo.
type MatchExclusionRepository interface {
	// SetExclusion positionne is_excluded pour un match (UPSERT).
	SetExclusion(ctx context.Context, matchID string, excluded bool) error

	// ListExcluded retourne les matchs marqués is_excluded = TRUE.
	ListExcluded(ctx context.Context) ([]domain.ExcludedMatch, error)
}

// MediaRepository fournit les données pour la galerie médias.
// Implémenté par platform/duckdb.MediaRepo.
type MediaRepository interface {
	// LoadMediaFiles charge les médias actifs paginés avec filtres dynamiques.
	LoadMediaFiles(ctx context.Context, filters domain.MediaFilters, limit, offset int) ([]domain.MediaFileRow, error)

	// CountMediaFiles retourne le nombre total de médias actifs selon les filtres.
	CountMediaFiles(ctx context.Context, filters domain.MediaFilters) (int, error)

	// LoadMediaFilterOptions retourne les valeurs distinctes exposées dans les listes déroulantes.
	LoadMediaFilterOptions(ctx context.Context, filters domain.MediaFilters) (domain.MediaFilterOptions, error)

	// SetMediaLike persiste l'état liked d'un média. Retourne false si le média est introuvable.
	SetMediaLike(ctx context.Context, filePath string, liked bool) (bool, error)

	// ToggleSharedLike écrit/supprime un like dans la table media_likes partagée.
	ToggleSharedLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string, liked bool) error

	// GetMediaLikers retourne les likers (noms + total) pour une liste de media_path.
	GetMediaLikers(ctx context.Context, mediaPaths []string) (map[string]domain.MediaLikersInfo, error)

	// CurrentPlayerSlug retourne le slug du joueur dont on lit la galerie. Utilisé
	// pour distinguer "mine" vs "teammate" dans la section affichée.
	CurrentPlayerSlug() string

	// LoadMatchCandidatesForMedia retourne les matchs du joueur courant proches
	// temporellement du média (fenêtre ±windowMinutes). Utilisé pour la
	// réassociation manuelle quand l'algo automatique a deviné le mauvais match.
	LoadMatchCandidatesForMedia(ctx context.Context, filePath string, windowMinutes int) (domain.MediaMatchCandidatesResponse, error)

	// SetMediaMatchAssociation force l'association d'un média à un match précis.
	// Remplace l'association existante. Retourne (mapName, modeName) du nouveau match.
	SetMediaMatchAssociation(ctx context.Context, filePath, matchID string) (mapName, modeName *string, err error)
}

// SocialRepository gère les données sociales (favoris) dans shared_social.duckdb.
// Implémenté par platform/duckdb.SocialRepo.
type SocialRepository interface {
	// ToggleMatchFavorite bascule l'état favori d'un match pour un joueur.
	// Retourne le nouvel état.
	ToggleMatchFavorite(ctx context.Context, playerSlug, matchID string, favorited bool) error

	// IsMatchFavorite indique si un match est en favori pour un joueur.
	IsMatchFavorite(ctx context.Context, playerSlug, matchID string) (bool, error)

	// GetFavoriteMatchIDs retourne l'ensemble des match_id mis en favoris par un joueur.
	GetFavoriteMatchIDs(ctx context.Context, playerSlug string) (map[string]bool, error)
}

// Ensure compile-time checks pour les nouveaux repos Sprint 12+13.
var (
	_ SquadRepository     = (*noopSquadRepo)(nil)
	_ CitationsRepository = (*noopCitationsRepo)(nil)
	_ MediaRepository     = (*noopMediaRepo)(nil)
	_ SocialRepository    = (*noopSocialRepo)(nil)
)

// noopSquadRepo — impl nulle pour le check de compilation uniquement.
type noopSquadRepo struct{}

func (n *noopSquadRepo) LoadTopTeammates(_ context.Context, _ string) ([]domain.TopTeammateRow, error) {
	return nil, nil
}
func (n *noopSquadRepo) LookupXUIDByGamertag(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}
func (n *noopSquadRepo) LoadSquadMatches(_ context.Context, _, _ string) ([]domain.SquadMatchRow, error) {
	return nil, nil
}
func (n *noopSquadRepo) LoadTeammateMatches(_ context.Context, _, _ string) ([]domain.TeammateMatchRow, error) {
	return nil, nil
}
func (n *noopSquadRepo) LoadImpactEvents(_ context.Context, _ []string) ([]domain.ImpactEventRow, error) {
	return nil, nil
}
func (n *noopSquadRepo) LoadMainTeamParticipants(_ context.Context, _ string, _ []string) ([]domain.AllyParticipant, error) {
	return nil, nil
}
func (n *noopSquadRepo) LoadSynthesisHeatmap(_ context.Context, _ string) ([]domain.SynthesisHeatmapRow, error) {
	return nil, nil
}
func (n *noopSquadRepo) LoadAssetTranslationsFR(_ context.Context, _ string, _ []string) (map[string]string, error) {
	return nil, nil
}
func (n *noopSquadRepo) LoadModeTranslationsFR(_ context.Context, _ []string) (map[string]string, error) {
	return nil, nil
}
func (n *noopSquadRepo) LoadMapStatsForSquad(_ context.Context, _ string, _ []string) (map[string]domain.MapSquadStats, error) {
	return nil, nil
}

// noopCitationsRepo — impl nulle pour le check de compilation uniquement.
type noopCitationsRepo struct{}

func (n *noopCitationsRepo) LoadCitationMappings(_ context.Context) ([]domain.CitationMappingRow, error) {
	return nil, nil
}
func (n *noopCitationsRepo) LoadCitationTotals(_ context.Context) ([]domain.CitationTotalRow, error) {
	return nil, nil
}
func (n *noopCitationsRepo) LoadMedalTotals(_ context.Context, _ string) ([]domain.MedalEarnedRow, error) {
	return nil, nil
}
func (n *noopCitationsRepo) LoadMedalCitationMappings(_ context.Context) ([]domain.MedalCitationRow, error) {
	return nil, nil
}
func (n *noopCitationsRepo) LoadCitationMedalMappings(_ context.Context) ([]domain.CitationMedalMapping, error) {
	return nil, nil
}
func (n *noopCitationsRepo) LoadMatchCitationsForView(_ context.Context, _ string) ([]domain.CitationMatchViewRow, error) {
	return nil, nil
}
func (n *noopCitationsRepo) WriteCitationsForMatch(_ context.Context, _ string, _ []domain.CitationMatchDelta) error {
	return nil
}

// noopMediaRepo — impl nulle pour le check de compilation uniquement.
type noopMediaRepo struct{}

func (n *noopMediaRepo) LoadMediaFiles(_ context.Context, _ domain.MediaFilters, _, _ int) ([]domain.MediaFileRow, error) {
	return nil, nil
}
func (n *noopMediaRepo) CountMediaFiles(_ context.Context, _ domain.MediaFilters) (int, error) {
	return 0, nil
}
func (n *noopMediaRepo) LoadMediaFilterOptions(_ context.Context, _ domain.MediaFilters) (domain.MediaFilterOptions, error) {
	return domain.MediaFilterOptions{}, nil
}

// ─── Asset Drawer ────────────────────────────────────────────────────────────

// AssetMetaRepository fournit les métadonnées d'assets pour l'Asset Drawer.
// Implémenté par platform/duckdb.MetadataRepo.
type AssetMetaRepository interface {
	// ListMapsByTitle retourne les maps d'un titre filtrées par search (LIKE, vide = tout).
	ListMapsByTitle(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)

	// ListWeaponsByTitle retourne les armes filtrées par search (LIKE, vide = tout).
	// titleID est accepté pour l'interface mais weapon_labels n'est pas segmenté par titre en V1.
	ListWeaponsByTitle(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
}

var _ AssetMetaRepository = (*noopAssetMetaRepo)(nil)

type noopAssetMetaRepo struct{}

func (n *noopAssetMetaRepo) ListMapsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, nil
}
func (n *noopAssetMetaRepo) ListWeaponsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, nil
}

// ─── Sprint 54 : Metadata, Compare, Leaderboard ─────────────────────────────

// MetadataRepository fournit les saisons et snapshots de métadonnées.
// Implémenté par platform/duckdb.MetadataRepo.
type MetadataRepository interface {
	// GetCurrentSeason retourne la saison courante (last EndDate IS NULL ou MAX(StartDate)).
	GetCurrentSeason(ctx context.Context, titleID string) (*domain.SeasonCalendar, error)

	// ListSeasons retourne toutes les saisons connues pour un titre, triées
	// par StartDate ASC. Utilisé par SeasonsCatalog pour la fusion TOML+DB
	// (V2 saisons : pattern lazy-fetch + persist symétrique au battle pass).
	ListSeasons(ctx context.Context, titleID string) ([]domain.SeasonCalendar, error)

	// GetCSRSeasons retourne toutes les saisons CSR triées par StartDate DESC.
	GetCSRSeasons(ctx context.Context, titleID string) ([]domain.CSRSeasonCalendar, error)

	// GetSeasonByDate retourne la saison active à la date donnée.
	GetSeasonByDate(ctx context.Context, titleID string, date string) (*domain.SeasonCalendar, error)

	// UpsertSeason insère ou met à jour une saison (upsert sur season_id+title_id).
	UpsertSeason(ctx context.Context, s domain.SeasonCalendar) error

	// UpsertCSRSeason insère ou met à jour une saison CSR.
	UpsertCSRSeason(ctx context.Context, s domain.CSRSeasonCalendar) error

	// UpsertSnapshot enregistre un snapshot de ressource Waypoint.
	UpsertSnapshot(ctx context.Context, snap domain.WaypointResourceSnapshot) error

	// GetSnapshot retourne le dernier snapshot d'une ressource.
	GetSnapshot(ctx context.Context, titleID, resourceKey string) (*domain.WaypointResourceSnapshot, error)
}

// SeasonProvider fournit le calendrier des saisons depuis une source externe
// (Waypoint pour Halo Infinite). Implémenté par platform/halo.HaloProvider.
//
// Le contrat : retourne la liste des saisons + le payload brut (pour archive
// snapshot via UpsertSnapshot) + erreur. Les tokens d'auth sont lus depuis
// le contexte (cf. ctxkeys.HaloTokens) — pas dans la signature pour rester
// title-agnostic.
type SeasonProvider interface {
	FetchSeasonCalendar(ctx context.Context, titleID string) ([]domain.SeasonCalendar, []byte, error)
}

// CompareRepository fournit les stats normalisées d'un joueur local depuis DuckDB.
// Implémenté par platform/duckdb.CompareRepo.
type CompareRepository interface {
	// GetLocalStats retourne les stats normalisées depuis shared.match_participants.
	GetLocalStats(ctx context.Context, xuid, titleSlug string) (*domain.NormalizedPlayerStats, error)

	// ResolveXUID retourne le XUID pour un gamertag dans le registre local.
	ResolveXUID(ctx context.Context, gamertag string) (string, error)
}

// LeaderboardRepository fournit les données pour le classement CSR local.
// Implémenté par platform/duckdb.LeaderboardRepo.
type LeaderboardRepository interface {
	// GetLocalLeaderboard retourne les joueurs locaux triés par CSR DESC.
	GetLocalLeaderboard(ctx context.Context, titleSlug, season, playlist string) ([]domain.LeaderboardEntry, error)
}

// Ensure compile-time checks Sprint 54.
var (
	_ MetadataRepository    = (*noopMetadataRepo)(nil)
	_ CompareRepository     = (*noopCompareRepo)(nil)
	_ LeaderboardRepository = (*noopLeaderboardRepo)(nil)
)

// noopMetadataRepo — impl nulle pour le check de compilation uniquement.
type noopMetadataRepo struct{}

func (n *noopMetadataRepo) GetCurrentSeason(_ context.Context, _ string) (*domain.SeasonCalendar, error) {
	return nil, nil
}
func (n *noopMetadataRepo) ListSeasons(_ context.Context, _ string) ([]domain.SeasonCalendar, error) {
	return nil, nil
}
func (n *noopMetadataRepo) GetCSRSeasons(_ context.Context, _ string) ([]domain.CSRSeasonCalendar, error) {
	return nil, nil
}
func (n *noopMetadataRepo) GetSeasonByDate(_ context.Context, _, _ string) (*domain.SeasonCalendar, error) {
	return nil, nil
}
func (n *noopMetadataRepo) UpsertSeason(_ context.Context, _ domain.SeasonCalendar) error {
	return nil
}
func (n *noopMetadataRepo) UpsertCSRSeason(_ context.Context, _ domain.CSRSeasonCalendar) error {
	return nil
}
func (n *noopMetadataRepo) UpsertSnapshot(_ context.Context, _ domain.WaypointResourceSnapshot) error {
	return nil
}
func (n *noopMetadataRepo) GetSnapshot(_ context.Context, _, _ string) (*domain.WaypointResourceSnapshot, error) {
	return nil, nil
}

// noopCompareRepo — impl nulle pour le check de compilation uniquement.
type noopCompareRepo struct{}

func (n *noopCompareRepo) GetLocalStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return nil, nil
}
func (n *noopCompareRepo) ResolveXUID(_ context.Context, _ string) (string, error) {
	return "", nil
}

// noopLeaderboardRepo — impl nulle pour le check de compilation uniquement.
type noopLeaderboardRepo struct{}

func (n *noopLeaderboardRepo) GetLocalLeaderboard(_ context.Context, _, _, _ string) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

// PrivacyStateRepository persiste et charge l'état de privacy d'un joueur.
// Implémenté par platform/duckdb.PrivacyStateRepo.
// Sprint 55 E2-E4 : persistance durable du warning privacy (fallback Waypoint).
type PrivacyStateRepository interface {
	UpsertPrivacyState(ctx context.Context, state domain.PlayerPrivacyState) error
	LoadPrivacyState(ctx context.Context, xuid string) (*domain.PlayerPrivacyState, error)
}

// noopPrivacyStateRepo — impl nulle pour le check de compilation uniquement.
type noopPrivacyStateRepo struct{}

func (n *noopPrivacyStateRepo) UpsertPrivacyState(_ context.Context, _ domain.PlayerPrivacyState) error {
	return nil
}
func (n *noopPrivacyStateRepo) LoadPrivacyState(_ context.Context, _ string) (*domain.PlayerPrivacyState, error) {
	return nil, nil
}

var _ PrivacyStateRepository = (*noopPrivacyStateRepo)(nil)

// noopSocialRepo — impl nulle pour le check de compilation uniquement.
type noopSocialRepo struct{}

func (n *noopSocialRepo) ToggleMatchFavorite(_ context.Context, _, _ string, _ bool) error {
	return nil
}
func (n *noopSocialRepo) IsMatchFavorite(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (n *noopSocialRepo) GetFavoriteMatchIDs(_ context.Context, _ string) (map[string]bool, error) {
	return nil, nil
}
