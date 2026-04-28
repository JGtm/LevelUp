// Package port — engagement_score.go : interfaces de persistence et service
// pour la metrique EngagementScore.
//
// Reference conceptuelle : .ai/REFLEXION_ENGAGEMENT_SCORE_INTRA_MATCH.md
// Plan d'implementation : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md
//
// Phase 1.4 du plan : decouplage handler/service via interface, et
// service/repository via interface. Le service ne connait pas l'implementation
// DuckDB ; le handler ne connait pas le service concret.
package port

import (
	"context"
	"errors"

	"levelup/go-api/internal/domain"
)

// ErrEngagementUnavailable signale que la fonctionnalite EngagementScore
// n'est pas disponible pour ce titre / cette base. Generalement parce que
// la migration Phase 2 n'a pas encore ete appliquee (colonnes manquantes
// dans player_match_enrichment ou table engagement_coefficients absente).
//
// Les services / handlers doivent degrader gracieusement : retourner une
// reponse partielle sans champ engagement plutot que faire planter l'API.
var ErrEngagementUnavailable = errors.New("port: engagement persistence unavailable (migration not applied)")

// EngagementHistoryFilter parametre la lecture de l'historique d'un joueur
// pour le calcul du percentile.
//
// Garde-fou : XUID requis, ModeCategory requis (rejet de scan complet).
type EngagementHistoryFilter struct {
	// XUID du joueur cible. Requis.
	XUID string

	// ModeCategory normalise (PvP_ranked, PvP_unranked, ...). Requis.
	ModeCategory string

	// Limit borne le nombre de matchs renvoyes. Le service utilise typiquement
	// 200 (cf. doc reflexion §6.2). Si 0, le repo retourne une erreur.
	Limit int

	// ExcludeMatchID permet d'exclure un match precis de l'historique
	// (utile lors du calcul d'un score : on ne veut pas inclure le match
	// courant dans sa propre baseline). Vide = pas d'exclusion.
	ExcludeMatchID string
}

// Validate verifie la coherence des filtres.
func (f EngagementHistoryFilter) Validate() error {
	if f.XUID == "" {
		return errors.New("port: EngagementHistoryFilter.XUID required")
	}
	if f.ModeCategory == "" {
		return errors.New("port: EngagementHistoryFilter.ModeCategory required")
	}
	if f.Limit <= 0 {
		return errors.New("port: EngagementHistoryFilter.Limit must be > 0")
	}
	return nil
}

// EngagementScoreRepository gere la persistence des metriques d'engagement
// (score par match, courbe vivable depuis events, coefficients perso, intensite
// match).
//
// Implemente par internal/platform/duckdb.EngagementScoreRepo (Phase 1.5).
//
// Capability gating : toute methode peut retourner ErrEngagementUnavailable
// si la migration Phase 2 n'a pas ete appliquee. Le service amont doit
// degrader gracieusement (cf doc arch-rules : pas de panic).
type EngagementScoreRepository interface {
	// LoadPlayerHistory charge les residus bruts des N derniers matchs du
	// joueur sur la categorie de mode demandee. Utilise pour normaliser le
	// score courant en percentile.
	//
	// Retourne une slice vide (non nil) si aucun historique trouve. Le
	// service interpretera cela comme "insufficient_history".
	LoadPlayerHistory(ctx context.Context, filter EngagementHistoryFilter) ([]domain.HistoricalEngagementBrut, error)

	// LoadEngagementCoefficient charge le couple (CoefTeamShare, CoefLobbyShare)
	// stocke pour le joueur sur une categorie de mode. Retourne (nil, nil)
	// si aucun coefficient stocke (cas cold start).
	LoadEngagementCoefficient(ctx context.Context, xuid, modeCategory string) (*domain.EngagementCoefficient, error)

	// SaveEngagementScore persiste le score 0-100, le residu brut et la
	// confidence pour un (xuid, match_id). Idempotent : un appel ulterieur
	// avec les memes valeurs n'a pas d'effet (UPSERT).
	SaveEngagementScore(ctx context.Context, xuid, matchID string, result domain.EngagementScoreResult) error

	// SaveEngagementCoefficient persiste / met a jour les coefficients perso
	// pour le couple (XUID, ModeCategory). UPSERT.
	SaveEngagementCoefficient(ctx context.Context, coef domain.EngagementCoefficient) error

	// SaveMatchIntensity persiste l'intensite calculee pour un match. Cette
	// caracteristique est independante du joueur (1 valeur par match).
	SaveMatchIntensity(ctx context.Context, matchID string, intensity float64) error

	// LoadMatchIntensity lit l'intensite stockee. Retourne (0, false, nil)
	// si non encore calculee. Permet au service d'eviter de recalculer.
	LoadMatchIntensity(ctx context.Context, matchID string) (intensity float64, found bool, err error)

	// HasEngagementScore retourne true si un score est deja persiste pour
	// (xuid, match_id). Utilise par le pipeline sync pour skip les matchs
	// deja traites (sauf si force=true).
	HasEngagementScore(ctx context.Context, xuid, matchID string) (bool, error)
}

// MatchEngagementParams regroupe les inputs pour calculer le score d'un match.
//
// Le service amont charge les events (via HighlightEventsRepository), connait
// le mode et la composition du lobby/equipe, et passe le tout. Le calcul
// proprement dit est delegue a internal/analysis/temporal.ComputeEngagementScore.
type MatchEngagementParams struct {
	XUID         string
	MatchID      string
	ModeCategory string
	IsTeamMode   bool
	NTeam        int
	NHumansLobby int
	MatchStartMS int64
	MatchEndMS   int64

	// PersonalScore, Kills, Assists permettent de calculer les events
	// objectif estimes (modes asymetriques). Cf temporal.EventsObjectifEstimes.
	PersonalScore int
	Kills         int
	Assists       int
}

// EngagementScoreService orchestre les operations metier sur EngagementScore.
//
// Implemente par internal/service.EngagementScoreService (Phase 1.6).
type EngagementScoreService interface {
	// ComputeAndPersist calcule le score d'engagement pour un match et le
	// persiste. Appele par le pipeline de sync apres ingestion des events.
	//
	// Si force=false et qu'un score existe deja pour (xuid, match_id),
	// retourne le score existant sans recalcul. Si force=true, recalcule
	// et ecrase.
	ComputeAndPersist(ctx context.Context, params MatchEngagementParams, force bool) (*domain.EngagementScoreResult, error)

	// GetMatchEngagement reconstruit le score + la courbe pour un match
	// (lecture cote handler Match View). La courbe est recalculee a la
	// volee depuis les events (cf plan §9.4 stockage hybride).
	GetMatchEngagement(ctx context.Context, params MatchEngagementParams) (*domain.EngagementScoreResult, error)

	// GetEngagementProfile retourne les coefficients perso d'un joueur,
	// par categorie de mode. Utilise par l'endpoint dedie
	// /players/{slug}/engagement_profile.
	GetEngagementProfile(ctx context.Context, xuid string) ([]domain.EngagementCoefficient, error)
}
