// Package port - repository_sessions_home.go : interfaces SessionsRepository,
// StatsRepository, ConfigProvider, HomeRepository + noops. Decoupe de
// repository.go (god-file split, refactor 2026-05-27).
package port

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

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

func (n *noopMediaRepo) MediaExists(_ context.Context, _ string) (bool, error) {
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

	// LoadMatchCommendations charge les commendations NATIVES gagnées sur un lot de
	// matchs (Halo 5 : shared.match_commendations ⨝ commendation_definitions). Retourne
	// un map match_id → top commendations (count DESC). Dégradation silencieuse (table
	// absente / titre sans commendations natives → map vide). Sert le slot TopCitations
	// de la MatchCard pour les titres sans moteur de citations dérivé.
	LoadMatchCommendations(ctx context.Context, matchIDs []string) (map[string][]domain.HomeMatchCommendationRaw, error)

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

func (n *noopHomeRepo) LoadMatchCommendations(_ context.Context, _ []string) (map[string][]domain.HomeMatchCommendationRaw, error) {
	return map[string][]domain.HomeMatchCommendationRaw{}, nil
}

func (n *noopHomeRepo) LoadFavoriteWeapon(_ context.Context, _ string) (string, int, error) {
	return "", 0, nil
}

func (n *noopHomeRepo) EnrichCanonicalAssetTranslations(_ context.Context, _ []canonical.PlayerMatchRow) error {
	return nil
}

// SquadRepository fournit les données pour la page Escouade.
// Implémenté par platform/duckdb.SquadRepo.
