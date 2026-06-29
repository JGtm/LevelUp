// Package port definit les interfaces (ports) que les services utilisent.
// Les implementations concretes vivent dans internal/platform/.
// Ce package n'importe que des types domaine - jamais de platform/.
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient les interfaces
// de configuration initiale, filtres, match history, career, match view,
// explorer + gamertag + leurs noops. Les autres responsabilites vivent
// dans :
//
//   - repository_sessions_home.go : SessionsRepository, StatsRepository,
//     ConfigProvider, HomeRepository + noops
//   - repository_data.go          : Squad, Synthesis, Citations,
//     MatchExclusion, Media, Social, AssetMeta,
//     Metadata, SeasonProvider, Compare,
//     Leaderboard, PrivacyState + noops
package port

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
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
	// GetHighlightMatchIDs : 15 best + 15 worst match_ids triés par perf+dominance.
	// `filters` applique les filtres Expérience + Saisons (côté SQL).
	GetHighlightMatchIDs(ctx context.Context, filters domain.CareerHighlightFilters) ([]domain.HighlightMatchIDRow, error)
	// GetHighlightPool : pool complet des matchs éligibles "marquants"
	// (light : match_id + is_ranked + start_time). Pour cascade counts.
	GetHighlightPool(ctx context.Context) ([]domain.HighlightMatchPoolRow, error)
	// LoadModeTranslationsFR : map EN→FR depuis metadata.mode_name_tr (lang='fr').
	// Best-effort : nil silencieux si table absente.
	LoadModeTranslationsFR(ctx context.Context, modeENs []string) (map[string]string, error)
	// LoadPlaylistAssetTranslationsFR : map playlist_id→nom FR depuis
	// metadata.asset_translations. Best-effort : nil silencieux si absent.
	LoadPlaylistAssetTranslationsFR(ctx context.Context, playlistIDs []string) (map[string]string, error)
	// GetTopEncountersGlobal : 10 joueurs les plus croisés au niveau carrière,
	// hors XUIDs présents dans excludeXUIDs (typiquement les amis).
	GetTopEncountersGlobal(ctx context.Context, excludeXUIDs []string) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error)
	// GetRivals : 10 némésis (deaths DESC) + 10 souffre-douleur (frags DESC).
	GetRivals(ctx context.Context) (nemeses, victims []domain.CareerRivalRawRow, err error)
	// GetCSRSnapshots : classements CSR par playlist depuis player_csr_snapshots
	// pour la saison demandée. seasonID vide → saison courante (configurée).
	// Retourne slice vide (pas d'erreur) si aucun snapshot disponible.
	GetCSRSnapshots(ctx context.Context, seasonID string) ([]domain.CareerPlaylistCSR, error)
	// AvailableCSRSeasons : saisons CSR proposables dans le menu (saisons ayant
	// des snapshots pour ce joueur + saison courante), triées récentes d'abord.
	AvailableCSRSeasons(ctx context.Context) ([]domain.CSRSeasonOption, error)
}

// RelationsRepository fournit les agrégats du hub Communauté > Relations.
// Implémenté par platform/duckdb.CareerRepo (méthode GetRelations).
type RelationsRepository interface {
	// GetRelations : tous les joueurs récurrents (>= 2 matchs communs) avec
	// agrégats allié/ennemi, KDA moyens, duel et bornes temporelles.
	//
	// scope restreint l'agrégation à un sous-ensemble de match_id (Phase 2,
	// segmentation serveur). scope == nil ⇒ aucun filtre (tous les matchs,
	// comportement Phase 1). scope non-nil et VIDE ⇒ aucun match en périmètre
	// ⇒ retour ([]) sans requête (un filtre qui ne matche rien).
	GetRelations(ctx context.Context, scope []string) ([]domain.RelationRawRow, error)

	// GetRelationsHeatmap : pour les TOP-N relations (les plus de matchs communs,
	// bots exclus), le nombre de matchs communs par heure UTC. Bucketing en
	// day-parts fait côté service. scope : même contrat que GetRelations.
	GetRelationsHeatmap(ctx context.Context, scope []string, topN int) ([]domain.RelationHeatmapRawRow, error)

	// GetRivalTimeline : séquence des `limit` derniers duels (matchs communs
	// joués EN ENNEMI) contre rivalXUID, ordonnée ancien→récent, frags
	// directionnels par match. scope : même contrat que GetRelations.
	GetRivalTimeline(ctx context.Context, rivalXUID string, scope []string, limit int) ([]domain.RelationDuelRawRow, error)

	// GetCoreEngagement : agrégats joueur-centriques de la carte résumé du
	// noyau dur — WR global du joueur (lift) + issues des `limit` derniers
	// matchs joués avec un membre du noyau (coreXUIDs). coreXUIDs vide ⇒
	// RecentForm vide (le WR reste calculé). scope : même contrat que
	// GetRelations (nil = tous ; vide = court-circuit).
	GetCoreEngagement(ctx context.Context, coreXUIDs []string, scope []string, limit int) (domain.CoreEngagement, error)

	// GetRelationRecentForm : issues ("win"|"loss"|"other") des `limit` derniers
	// matchs joués À CÔTÉS du joueur `xuid` (même équipe), ordonnées ancien→récent.
	// Pour la sparkline « Derniers matchs ensemble » de la carte binôme. xuid vide
	// ⇒ nil. scope : même contrat que GetRelations.
	GetRelationRecentForm(ctx context.Context, xuid string, scope []string, limit int) ([]string, error)
}

// FriendMatchExtras : enrichissement per-friend pour le panneau d'expander
// scoreboard (port 1:1 du Python `match_view_scoreboard_detail.py` section
// "Local"). Chargé depuis la player DB de l'ami (pas du joueur principal),
// donc disponible uniquement pour les amis configurés dans db_profiles.json
// avec une player DB existante.
type FriendMatchExtras struct {
	PerformanceScore *float64
	HadBotTeammate   bool
	SkillRank        *domain.MatchScoreboardSkillRank
	AssistsModel     *domain.PlayerAssistsModel
}

// FriendsExtrasResolver charge les FriendMatchExtras pour la liste de xuids
// fournie. Best-effort : un xuid sans player DB configurée ou sans données
// ne figure simplement pas dans la map de retour.
type FriendsExtrasResolver func(ctx context.Context, matchID string, gameVariantName string, xuids []string) map[string]FriendMatchExtras

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

	// GetMatchObjectiveScore retourne la somme des award_score (catégorie
	// 'objective') pour un joueur dans un match. Source : personal_score_awards.
	// Utilisé pour l'axe Objective du radar synergie (MatchView). Dégradation
	// silencieuse à 0 si la table est absente ou pas de données pour ce match.
	GetMatchObjectiveScore(ctx context.Context, xuid, matchID string) (int, error)

	// GetMatchMedals retourne les médailles du joueur dans ce match (Q14).
	GetMatchMedals(ctx context.Context, xuid, matchID string) ([]domain.MedalRaw, error)

	// GetMatchEvents retourne les events highlight du match (Q21).
	GetMatchEvents(ctx context.Context, matchID string) ([]domain.EventRaw, error)

	// GetMatchKVPairs retourne les paires killer→victim du match (Q20).
	GetMatchKVPairs(ctx context.Context, matchID string) ([]domain.KVPairRaw, error)

	// GetMatchNeighbors retourne les matchs précédent/suivant (Q25 chronologie globale).
	GetMatchNeighbors(ctx context.Context, xuid, matchID string) (*domain.MatchNeighbors, error)

	// GetMatchNeighborsFiltered : variante paramétrable de Q25. spec=nil
	// équivaut à GetMatchNeighbors. Phase 2b du rework header MatchView.
	GetMatchNeighborsFiltered(
		ctx context.Context,
		xuid, matchID string,
		spec *domain.MatchFilterSpec,
	) (*domain.MatchNeighbors, error)

	// GetMatchSkillRank retourne le rang compétitif pour ce match (Q22).
	GetMatchSkillRank(ctx context.Context, matchID string) (*domain.SkillRankRaw, error)

	// GetMatchSharedCSRs retourne le CSR de tous les participants d'un match
	// ranked depuis shared.match_csrs_latest. Map xuid → SkillRankRaw.
	// Dégradation gracieuse (nil, nil) si table absente ou match non-ranked.
	GetMatchSharedCSRs(ctx context.Context, matchID string) (map[string]*domain.SkillRankRaw, error)

	// GetMatchEncounters retourne l'historique de rencontres avec les participants (Q23).
	GetMatchEncounters(ctx context.Context, matchID, myXUID string) ([]domain.EncounterRaw, error)

	// GetMatchEncounterStats retourne les stats riches par encounter (Q23b,
	// chunk MV4.C'). Permet narrative.ComputeEncounterBadges (ally_plus +
	// tough_enemy). Optionnel : implémentations qui ne supportent pas peuvent
	// retourner (nil, nil) — le service dégrade gracieusement (badge ordinal
	// seul attribué).
	GetMatchEncounterStats(ctx context.Context, matchID, myXUID string) ([]domain.EncounterStatsRaw, error)

	// GetMatchMedia retourne les médias associés au match (Q24).
	// Cross-joueur : tous les auteurs sont retournés (un coéquipier peut avoir
	// uploadé un media pour ce match).
	GetMatchMedia(ctx context.Context, matchID string) ([]domain.MediaAssocRaw, error)

	// GetMatchExpectedStats retourne les stats attendues pour ce match (Q26).
	GetMatchExpectedStats(ctx context.Context, matchID, xuid string) (*domain.ExpectedStatsRaw, error)

	// GetMatchBulkMedals retourne les médailles de tous les joueurs du match (Q27).
	GetMatchBulkMedals(ctx context.Context, matchID string) ([]domain.BulkMedalRaw, error)

	// GetMatchBulkWeaponKills retourne les kills par arme de tous les joueurs (Q28).
	GetMatchBulkWeaponKills(ctx context.Context, matchID string) ([]domain.BulkWeaponKillRaw, error)

	// GetHistoryForAvg retourne les 50 derniers matchs du joueur pour le calcul
	// des moyennes historiques K/D/A + spree/headshots/perfect (Q29).
	GetHistoryForAvg(ctx context.Context, xuid string) ([]domain.MatchHistAvgRow, error)

	// GetPlayerAssistsModel retourne les coefs OLS expected_assists pour un mode.
	// Retourne nil si le modèle n'existe pas (joueur sans DB locale ou < 15 matchs
	// dans ce mode). Dégradation gracieuse vers le modèle populationnel.
	GetPlayerAssistsModel(ctx context.Context, gameVariantName string) (*domain.PlayerAssistsModel, error)
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

	// GetParticipantStatsForMatches agrège les stats brutes du joueur (xuid)
	// sur la liste de matchs fournie. Lecture sur shared.match_participants.
	// Retourne nil si matchIDs vide.
	GetParticipantStatsForMatches(ctx context.Context, xuid string, matchIDs []string) (*domain.ParticipantStatsAggregate, error)

	// GetMedalCountsForMatches retourne le total d'occurrences de médailles
	// et le nombre de types distincts gagnés par le joueur (xuid) sur la
	// liste de matchs fournie. Lecture sur shared.medals_earned.
	GetMedalCountsForMatches(ctx context.Context, xuid string, matchIDs []string) (*domain.MedalCountsAggregate, error)

	// GetMatchStartTimesForXUID retourne les start_time (UTC) de tous les matchs
	// du joueur dans shared.match_participants. Local = historique complet ;
	// adversaire = matchs observés. Sert au bucketing "matchs par saison".
	GetMatchStartTimesForXUID(ctx context.Context, xuid string) ([]time.Time, error)

	// GetTargetRecentMatches retourne les `limit` derniers matchs PvP (firefight
	// exclu) du joueur, du plus récent au plus ancien, pour les graphes "profil
	// de combat" de l'Explorer. Lecture shared.match_participants + match_registry
	// + medals_earned (perfect kills). Retourne nil si xuid vide ou limit <= 0.
	GetTargetRecentMatches(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error)

	// TranslateModeUIsFR traduit EN PLACE les ModeUI (sous-modes EN normalisés) en
	// FR via metadata.mode_name_tr — même résolution que GetTargetRecentMatches.
	// Sert à homogénéiser la source LIVE du profil de combat (fetchée hors DB) avec
	// la source locale. Best-effort : no-op si metadata absent.
	TranslateModeUIsFR(ctx context.Context, rows []domain.ExplorerTargetRecentMatch)

	// GetTopWeaponsForMatches retourne le top `limit` armes (par kills) du joueur
	// sur les matchs donnés (shared.v_weapon_kills, COUNT(*) par effective_weapon_id)
	// + labels metadata.weapon_labels. Retourne nil si entrée vide — best-effort.
	GetTopWeaponsForMatches(ctx context.Context, xuid string, matchIDs []string, limit int) ([]domain.WeaponHighlight, error)
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
func (n *noopCareerRepo) GetHighlightMatchIDs(_ context.Context, _ domain.CareerHighlightFilters) ([]domain.HighlightMatchIDRow, error) {
	return nil, nil
}
func (n *noopCareerRepo) GetHighlightPool(_ context.Context) ([]domain.HighlightMatchPoolRow, error) {
	return nil, nil
}
func (n *noopCareerRepo) LoadModeTranslationsFR(_ context.Context, _ []string) (map[string]string, error) {
	return nil, nil
}
func (n *noopCareerRepo) LoadPlaylistAssetTranslationsFR(_ context.Context, _ []string) (map[string]string, error) {
	return nil, nil
}
func (n *noopCareerRepo) GetTopEncountersGlobal(_ context.Context, _ []string) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error) {
	return nil, nil, nil
}
func (n *noopCareerRepo) GetRivals(_ context.Context) ([]domain.CareerRivalRawRow, []domain.CareerRivalRawRow, error) {
	return nil, nil, nil
}
func (n *noopCareerRepo) GetCSRSnapshots(_ context.Context, _ string) ([]domain.CareerPlaylistCSR, error) {
	return nil, nil
}
func (n *noopCareerRepo) AvailableCSRSeasons(_ context.Context) ([]domain.CSRSeasonOption, error) {
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
func (n *noopMatchViewRepo) GetMatchObjectiveScore(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (n *noopMatchViewRepo) GetMatchMedals(_ context.Context, _, _ string) ([]domain.MedalRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchEvents(_ context.Context, _ string) ([]domain.EventRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchKVPairs(_ context.Context, _ string) ([]domain.KVPairRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchNeighbors(_ context.Context, _, _ string) (*domain.MatchNeighbors, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchNeighborsFiltered(_ context.Context, _, _ string, _ *domain.MatchFilterSpec) (*domain.MatchNeighbors, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchSkillRank(_ context.Context, _ string) (*domain.SkillRankRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchSharedCSRs(_ context.Context, _ string) (map[string]*domain.SkillRankRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchEncounters(_ context.Context, _, _ string) ([]domain.EncounterRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchEncounterStats(_ context.Context, _, _ string) ([]domain.EncounterStatsRaw, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetMatchMedia(_ context.Context, _ string) ([]domain.MediaAssocRaw, error) {
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
func (n *noopMatchViewRepo) GetHistoryForAvg(_ context.Context, _ string) ([]domain.MatchHistAvgRow, error) {
	return nil, nil
}
func (n *noopMatchViewRepo) GetPlayerAssistsModel(_ context.Context, _ string) (*domain.PlayerAssistsModel, error) {
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
func (n *noopExplorerRepo) GetParticipantStatsForMatches(_ context.Context, _ string, _ []string) (*domain.ParticipantStatsAggregate, error) {
	return nil, nil
}
func (n *noopExplorerRepo) GetMedalCountsForMatches(_ context.Context, _ string, _ []string) (*domain.MedalCountsAggregate, error) {
	return nil, nil
}
func (n *noopExplorerRepo) GetMatchStartTimesForXUID(_ context.Context, _ string) ([]time.Time, error) {
	return nil, nil
}
func (n *noopExplorerRepo) GetTargetRecentMatches(_ context.Context, _ string, _ int) ([]domain.ExplorerTargetRecentMatch, error) {
	return nil, nil
}
func (n *noopExplorerRepo) TranslateModeUIsFR(_ context.Context, _ []domain.ExplorerTargetRecentMatch) {
}
func (n *noopExplorerRepo) GetTopWeaponsForMatches(_ context.Context, _ string, _ []string, _ int) ([]domain.WeaponHighlight, error) {
	return nil, nil
}

// SessionsRepository fournit les données brutes pour le calcul des sessions.
// Implémenté par platform/duckdb.SessionsRepo.
