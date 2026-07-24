// Package port - repository_data.go : interfaces SquadRepository,
// SynthesisRepository, CitationsRepository, MatchExclusionRepository,
// MediaRepository, SocialRepository, AssetMetaRepository, MetadataRepository,
// SeasonProvider, CompareRepository, LeaderboardRepository,
// PrivacyStateRepository + noops. Decoupe de repository.go (god-file split,
// refactor 2026-05-27).
package port

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

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
	//
	// Title-agnostic : si highlight_events ne porte aucun kill/death sur le lot (ex.
	// Halo 5 = médailles seules), l'implémentation synthétise les kill/death depuis
	// killer_victim_pairs (LoadKVPairs) via analysis.SynthesizeKillEventsFromKVPairs
	// et les fusionne, triés par TimeMS. NO-OP sur Infinite (kills déjà présents).
	LoadImpactEvents(ctx context.Context, matchIDs []string) ([]domain.ImpactEventRow, error)

	// LoadKVPairs charge les paires killer→victim horodatées (killer_victim_pairs)
	// pour une liste de match_ids (lecture batch, shared DB). Source du fallback
	// title-agnostic de synthèse d'events kill/death utilisé par LoadImpactEvents
	// (et, côté solo, par HighlightEventsRepo). Retourne nil si matchIDs est vide.
	LoadKVPairs(ctx context.Context, matchIDs []string) ([]domain.KVPairRaw, error)

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
	//
	// excludeXUIDs (composition exacte) : coéquipiers connus HORS sélection —
	// anti-join qui écarte les matchs où l'un d'eux figure sur l'équipe du main
	// (parité avec le filtre allSquadRows). nil/vide = pas d'exclusion.
	LoadMapStatsForSquad(ctx context.Context, mainXUID string, squadXUIDs, excludeXUIDs []string) (map[string]domain.MapSquadStats, error)

	// P4.3 finale : LoadSynthesisMatches retiré (squad/teammates chargent
	// canonical via PlayerMatchesRepository).
}

// SynthesisRepository fournit les données pour la page Synthèse (Sprint 55 D1).
// Sous-ensemble de SquadRepository — isolé pour l'injection ciblée.
//
// P4.3 finale : LoadSynthesisMatches retiré.
type SynthesisRepository interface {
	// LoadSynthesisHeatmap charge la heatmap carte×mode (Q33).
	LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error)
	// EnrichCanonicalAssetTranslations remplit Labels["fr"] sur les AssetReference
	// (Map, Playlist, GameVariant, PairMode) des rows canoniques depuis
	// metadata.asset_translations + mode_name_tr quand match_registry.{...}_name_fr
	// est NULL en DB. Même helper que HomeRepo — réutilisé pour cohérence cross-page.
	// Best-effort : erreurs loggées, rows mutées en place.
	EnrichCanonicalAssetTranslations(ctx context.Context, rows []canonical.PlayerMatchRow) error
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

	// LoadMatchCitationsRich charge les citations riches d'un match (Q41) : delta + cumul + tiers.
	// Utilisé par le Summary tab pour les progress rings et le filtrage mastery.
	LoadMatchCitationsRich(ctx context.Context, matchID string) ([]domain.HomeMatchCitationRaw, error)

	// LoadMatchCommendationsRich charge les commendations NATIVES (Halo 5) gagnées
	// par un joueur (xuid) sur UN match : ID + nom + icône + count (delta) + progress
	// (cumul à vie) + tier_targets. Match-scoped (toutes les commendations du match,
	// PAS de top-N : le filtrage mastery + la sélection sont faits au build côté
	// service). Source = shared.match_commendations ⨝ commendation_definitions.
	// Dégradation silencieuse : titre sans table (Infinite) / SharedReader indispo →
	// slice vide. Le viewer doit fournir son xuid (les commendations sont par joueur).
	LoadMatchCommendationsRich(ctx context.Context, matchID, xuid string) ([]domain.HomeMatchCommendationRaw, error)
}

// MatchExclusionRepository gère le flag is_excluded dans player_match_enrichment.
// Implémenté par platform/duckdb.MatchExclusionRepo.
type MatchExclusionRepository interface {
	// SetExclusion positionne is_excluded pour un match (UPSERT).
	SetExclusion(ctx context.Context, matchID string, excluded bool) error

	// ListExcluded retourne les matchs marqués is_excluded = TRUE.
	ListExcluded(ctx context.Context) ([]domain.ExcludedMatch, error)

	// GetMatchRegistryInfo retourne les métadonnées du match depuis
	// shared.match_registry. Utilisé pour la garde "match classé" et la
	// détermination de la chaîne perf / playlist_group LUSR. Retourne
	// domain.ErrMatchNotFound si le match_id n'existe pas.
	GetMatchRegistryInfo(ctx context.Context, matchID string) (domain.MatchRegistryInfo, error)
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

	// ListMediaAuthors retourne les player_slug distincts ayant des médias (+ leur
	// compte), depuis shared_social.media_files. Gamertag/is_self résolus par le caller.
	ListMediaAuthors(ctx context.Context) ([]domain.MediaAuthor, error)

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
func (n *noopSquadRepo) LoadKVPairs(_ context.Context, _ []string) ([]domain.KVPairRaw, error) {
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
func (n *noopSquadRepo) LoadMapStatsForSquad(_ context.Context, _ string, _, _ []string) (map[string]domain.MapSquadStats, error) {
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
func (n *noopCitationsRepo) LoadMatchCitationsRich(_ context.Context, _ string) ([]domain.HomeMatchCitationRaw, error) {
	return nil, nil
}
func (n *noopCitationsRepo) LoadMatchCommendationsRich(_ context.Context, _, _ string) ([]domain.HomeMatchCommendationRaw, error) {
	return nil, nil
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
func (n *noopMediaRepo) ListMediaAuthors(_ context.Context) ([]domain.MediaAuthor, error) {
	return nil, nil
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

	// ListMedalsByTitle retourne les médailles d'un titre filtrées par search.
	// Implémentation in-memory (StaticAssetMetaRepo) ; vide pour les titres sans seed médailles.
	ListMedalsByTitle(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
}

var _ AssetMetaRepository = (*noopAssetMetaRepo)(nil)

type noopAssetMetaRepo struct{}

func (n *noopAssetMetaRepo) ListMapsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, nil
}
func (n *noopAssetMetaRepo) ListWeaponsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, nil
}
func (n *noopAssetMetaRepo) ListMedalsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
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

	// GetAssistsCoef retourne les coefs de régression pour expected_assists.
	// Retourne les coefs du mode si disponible, sinon le fallback '__global__'.
	GetAssistsCoef(ctx context.Context, gameVariantName string) (slope, intercept float64, err error)
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

	// GetPlayerATH retourne les métriques all-time depuis la player DB (stats.duckdb).
	// N'appeler que pour le joueur A — lit depuis pdb.Player (connexion du joueur principal).
	GetPlayerATH(ctx context.Context) (*domain.PlayerATH, error)

	// GetPlayerATHFor retourne les métriques all-time pour n'importe quel joueur local.
	// Lookup dans le pool global par {titleSlug}:{gamertag} — best-effort (nil si non ouvert).
	GetPlayerATHFor(ctx context.Context, gamertag, titleSlug string) (*domain.PlayerATH, error)

	// GetEncounterStats retourne les stats de rencontres historiques entre xuidA et xuidB.
	// Retourne nil si aucun match commun ou en cas d'erreur (best-effort).
	GetEncounterStats(ctx context.Context, xuidA, xuidB string) (*domain.CompareEncounterStats, error)

	// GetCrossMatchSample agrège les 4 métriques locale-only (max_killing_spree,
	// avg_life_secs, perfect_kills_per_game, headshot_kills_per_game) du joueur
	// xuidB calculées sur les matchs où xuidA et xuidB sont tous deux participants.
	// Retourne (nil, nil) si aucun match croisé exploitable — best-effort.
	GetCrossMatchSample(ctx context.Context, xuidA, xuidB string) (*domain.CrossMatchSample, error)
}

// LeaderboardRepository fournit les données pour la page Classement.
// Implémenté par platform/duckdb.LeaderboardRepo.
type LeaderboardRepository interface {
	// GetLocalLeaderboard retourne les joueurs locaux triés par CSR DESC (legacy).
	GetLocalLeaderboard(ctx context.Context, titleSlug, season, playlist string) ([]domain.LeaderboardEntry, error)
	// GetCSRWorldLeaderboard lit le dernier snapshot du classement CSR mondial
	// (scrapé depuis Halo Waypoint) pour un titre + saison + playlist (PMT-7).
	GetCSRWorldLeaderboard(ctx context.Context, titleSlug, season, playlist string, limit int) ([]domain.LeaderboardEntry, error)
	// GetStatLeaderboard agrège les stats mondiales par xuid pour un titre + une
	// catégorie de stat. playlist + season : filtres optionnels (season au format
	// interne "CsrSeasonN", cf. match_registry.season_id).
	GetStatLeaderboard(ctx context.Context, titleSlug string, category domain.LeaderboardCategory, playlist, season string, limit int) ([]domain.LeaderboardEntry, error)
	// GetWorldLeaderboardCatalog liste saisons + playlists ayant des snapshots
	// CSR mondiaux pour un titre (sélecteurs dynamiques).
	GetWorldLeaderboardCatalog(ctx context.Context, titleSlug string) (domain.LeaderboardCatalog, error)
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
func (n *noopMetadataRepo) GetAssistsCoef(_ context.Context, _ string) (float64, float64, error) {
	return 0, 0, nil
}

// noopCompareRepo — impl nulle pour le check de compilation uniquement.
type noopCompareRepo struct{}

func (n *noopCompareRepo) GetLocalStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return nil, nil
}
func (n *noopCompareRepo) ResolveXUID(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (n *noopCompareRepo) GetPlayerATH(_ context.Context) (*domain.PlayerATH, error) {
	return nil, nil
}
func (n *noopCompareRepo) GetPlayerATHFor(_ context.Context, _, _ string) (*domain.PlayerATH, error) {
	return nil, nil
}
func (n *noopCompareRepo) GetEncounterStats(_ context.Context, _, _ string) (*domain.CompareEncounterStats, error) {
	return nil, nil
}
func (n *noopCompareRepo) GetCrossMatchSample(_ context.Context, _, _ string) (*domain.CrossMatchSample, error) {
	return nil, nil
}

// noopLeaderboardRepo — impl nulle pour le check de compilation uniquement.
type noopLeaderboardRepo struct{}

func (n *noopLeaderboardRepo) GetLocalLeaderboard(_ context.Context, _, _, _ string) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}
func (n *noopLeaderboardRepo) GetCSRWorldLeaderboard(_ context.Context, _, _, _ string, _ int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}
func (n *noopLeaderboardRepo) GetStatLeaderboard(_ context.Context, _ string, _ domain.LeaderboardCategory, _, _ string, _ int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}
func (n *noopLeaderboardRepo) GetWorldLeaderboardCatalog(_ context.Context, _ string) (domain.LeaderboardCatalog, error) {
	return domain.LeaderboardCatalog{}, nil
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
