// Package port — services.go : interfaces des services métier.
//
// Sprint 37 : injection de dépendances — les handlers dépendent de ces
// interfaces, pas des implémentations concrètes de internal/service/.
package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Services joueur (per-player, créés par PlayerServiceFactory)
// ---------------------------------------------------------------------------

// CareerService construit les réponses de la page Carrière.
type CareerService interface {
	GetCareerPage(ctx context.Context) (domain.CareerPageResponse, error)
	GetTopMatches(ctx context.Context) (domain.CareerTopMatchesResponse, error)
	GetEncounters(ctx context.Context) (domain.CareerEncountersResponse, error)
}

// CitationsService construit les réponses Citations et Commendations.
type CitationsService interface {
	GetCitationsPage(ctx context.Context) (*domain.CitationsPageResponse, error)
	GetCommendationsPage(ctx context.Context, playerXUID string) (*domain.CommendationsPageResponse, error)
}

// ExplorerService orchestre les requêtes de l'Explorer (matchs communs).
type ExplorerService interface {
	GetCommonMatches(ctx context.Context, otherGamertag string, limit int) (domain.ExplorerPlayerQueryResponse, error)
}

// FiltersService résout le contexte de filtres d'un joueur.
type FiltersService interface {
	Resolve(ctx context.Context, input domain.FilterContextInput) (domain.FilterContextResolved, error)
}

// HomeService construit les réponses de la page d'accueil Mission Control.
type HomeService interface {
	GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error)
	GetBattlePass(ctx context.Context) domain.BattlePassResponse
	GetChallenges(ctx context.Context) domain.ChallengesResponse
}

// SeasonPassService construit la réponse de la page Season Pass (palmares).
type SeasonPassService interface {
	GetSeasonPassPage(ctx context.Context) (domain.SeasonPassPageResponse, error)
}

// MatchHistoryService construit les réponses d'historique de matchs.
type MatchHistoryService interface {
	GetPage(ctx context.Context, req domain.MatchHistoryQueryRequest) (domain.MatchHistoryPageResponse, error)
	ExportCSV(ctx context.Context, req domain.MatchHistoryQueryRequest) ([]domain.MatchHistoryRow, error)
}

// MatchViewService construit la vue détaillée d'un match.
type MatchViewService interface {
	GetMatchView(ctx context.Context, matchID string) (domain.MatchViewResponse, error)
}

// MatchExclusionService gère le marquage et la liste des matchs non pertinents.
type MatchExclusionService interface {
	// SetExclusion marque ou démarque un match comme non pertinent.
	SetExclusion(ctx context.Context, matchID string, excluded bool) error

	// ListExcluded retourne les matchs exclus du joueur.
	ListExcluded(ctx context.Context) ([]domain.ExcludedMatch, error)
}

// MediaService construit la page de la galerie médias et gère l'upload.
type MediaService interface {
	GetMediaPage(ctx context.Context, req domain.MediaPageRequest) (*domain.MediaPageResponse, error)
	SetMediaLike(ctx context.Context, req domain.MediaLikeRequest) (*domain.MediaLikeResponse, error)
	UploadMedia(ctx context.Context, req domain.UploadRequest) (*domain.UploadResult, error)
}

// SocialService gère les interactions sociales (favoris de matchs).
type SocialService interface {
	ToggleMatchFavorite(ctx context.Context, req domain.MatchFavoriteRequest) error
}

// SessionsService construit la page des sessions.
type SessionsService interface {
	GetSessions(ctx context.Context, opts domain.SessionComputeOptions) (domain.SessionsResponse, error)
}

// SessionCompareService compare deux sessions.
type SessionCompareService interface {
	Compare(ctx context.Context, req domain.SessionCompareRequest) (domain.SessionCompareResponse, error)
}

// SessionPageService construit la page détail de session.
type SessionPageService interface {
	GetPage(ctx context.Context, req domain.SessionPageRequest) (domain.SessionPageResponse, error)
}

// SquadService orchestre les pages Escouade et Synthèse.
type SquadService interface {
	GetSquadPage(ctx context.Context, playerXUID, gamertag, teammateXUID string) (*domain.SquadPageResponse, error)
	GetSynthesisPage(ctx context.Context, playerXUID string) (*domain.SynthesisPageResponse, error)
}

// SynthesisService orchestre la page Synthèse (Sprint 55 D1 — autonome).
type SynthesisService interface {
	GetSynthesisPage(ctx context.Context, playerXUID string, req domain.SynthesisRequest) (*domain.SynthesisPageV2Response, error)
}

// StatsService construit la page Stats/Séries.
type StatsService interface {
	GetPage(ctx context.Context, req domain.StatsQueryRequest) (domain.StatsPageResponse, error)
}

// TeammatesService construit la page Coéquipiers.
type TeammatesService interface {
	GetPage(ctx context.Context, playerXUID string, req domain.TeammatesQueryRequest) (domain.TeammatesPageResponse, error)
}

// TimeseriesService construit la page Séries temporelles.
type TimeseriesService interface {
	GetPage(ctx context.Context, req domain.TimeseriesQueryRequest) (domain.TimeseriesPageResponse, error)
}

// ---------------------------------------------------------------------------
// Services globaux (singletons, résolution non liée à un joueur)
// ---------------------------------------------------------------------------

// BootstrapService construit le bootstrap et la liste des joueurs.
type BootstrapService interface {
	Build(ctx context.Context, sess *domain.SessionData) (*domain.BootstrapResponse, error)
	BuildPlayersList(ctx context.Context) (*domain.PlayersListResponse, error)
}

// GamertagSearchService cherche des gamertags dans la base partagée.
type GamertagSearchService interface {
	Search(ctx context.Context, query string) ([]domain.GamertagSearchResult, error)
}

// ProfileService gère la création de profils joueur (extrait de setup.go).
type ProfileService interface {
	CreatePlayer(req domain.CreatePlayerProfileRequest) (playerKey string, warnings []string, err error)
}

// ─── Sprint 54 : Compare, Leaderboard ────────────────────────────────────────

// CompareService construit la page Compare joueur vs joueur.
type CompareService interface {
	GetPage(ctx context.Context, req domain.CompareRequest) (domain.CompareResponse, error)
}

// LeaderboardService construit la page Classement CSR.
type LeaderboardService interface {
	GetPage(ctx context.Context, req domain.LeaderboardRequest) (domain.LeaderboardResponse, error)
}

// PlayerStatsProvider fournit les stats d'un joueur distant via Waypoint.
// Implémenté par platform/halo.CompareProvider.
type PlayerStatsProvider interface {
	FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error)
}

// PrivacyProvider interroge la privacy d'un compte Halo via Waypoint.
// Implémenté par platform/halo.HaloProvider.
type PrivacyProvider interface {
	GetMatchPrivacy(ctx context.Context, xuid string) (*domain.MatchPrivacyInfo, error)
}
