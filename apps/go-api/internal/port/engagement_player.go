package port

import (
	"context"
	"errors"

	"levelup/go-api/internal/domain"
)

// Erreurs d'un calcul d'engagement par match ; leur identité fixe le code d'API rendu.
var (
	// ErrEngagementMatchNotFound signale un match inconnu pour le joueur cible.
	ErrEngagementMatchNotFound = errors.New("engagement: match not found for this player")

	// ErrEngagementPvENotSupported signale qu'on tente de calculer sur un match PvE
	// (non couvert v1, cf doc reflexion §3.4 perimetre).
	ErrEngagementPvENotSupported = errors.New("engagement: PvE not supported in v1")

	// ErrEngagementInsufficient signale que la courbe n'a pas pu etre calculee pour
	// ce match precis (match trop court, ou aucun event exploitable pour le joueur
	// et ses coequipiers). Distinct de ErrEngagementUnavailable (migration absente) :
	// c'est une indisponibilite PAR MATCH, pas un probleme de schema -> 422, pas 503.
	// Le front affiche un message neutre ("indisponible pour ce match"), jamais
	// "migration en cours".
	ErrEngagementInsufficient = errors.New("engagement: insufficient data for this match")
)

// PlayerEngagementService calcule l'engagement d'UN joueur (le service est construit pour son xuid) :
// score par match, série d'un joueur sur ses derniers matchs, session d'escouade, profil long terme
// et recalcul admin des coefficients. Implémenté par service.PlayerEngagementService.
type PlayerEngagementService interface {
	// GetMatchEngagement : ErrEngagementMatchNotFound, ErrEngagementPvENotSupported,
	// ErrEngagementInsufficient, ErrEngagementUnavailable.
	GetMatchEngagement(ctx context.Context, matchID string) (*domain.EngagementScoreResult, error)
	GetEngagementProfile(ctx context.Context) ([]domain.EngagementProfile, error)
	GetTimeseries(ctx context.Context, filters domain.FilterContextInput, limit int) (*domain.EngagementTimeseriesResponse, error)
	GetSquadSession(ctx context.Context, matchIDs []string, teammates []domain.EngagementCoefficient) (*domain.SquadEngagementSession, error)
	// RecomputeCoefficients force le recalcul des coefficients du joueur ; ErrEngagementUnavailable si la
	// migration d'engagement n'est pas appliquée.
	RecomputeCoefficients(ctx context.Context) (*domain.RecomputeReport, error)
}
