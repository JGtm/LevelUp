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

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
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

	// LoadResponseBins charge les bins de reponse (terciles d'intensite) du
	// joueur pour une categorie de mode (modele lobby-anchored v2). Retourne
	// (nil, nil) si aucun bin persiste. ErrEngagementUnavailable si la table
	// engagement_response_bins est absente (migration non appliquee).
	LoadResponseBins(ctx context.Context, xuid, modeCategory string) (*domain.EngagementResponseBins, error)

	// SaveResponseBins persiste / met a jour les bins de reponse d'un joueur
	// pour une categorie de mode (SELECT-then-UPDATE-or-INSERT par bin, sous
	// lease). ErrEngagementUnavailable si la table est absente.
	SaveResponseBins(ctx context.Context, bins domain.EngagementResponseBins) error

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

	// LoadMatchEngagementContext charge les metadonnees d'un match necessaires
	// au calcul du score (start/end ms, mode flags, team_id, NTeam, NHumansLobby,
	// kills/assists/personal_score du joueur).
	//
	// Retourne (nil, nil) si match introuvable. ErrEngagementUnavailable si
	// les colonnes/tables requises sont absentes.
	LoadMatchEngagementContext(ctx context.Context, matchID, xuid string) (*MatchEngagementContext, error)

	// LoadEventsForMatch charge tous les events highlight_events d'un match.
	// Liste vide (non nil) si aucun event.
	LoadEventsForMatch(ctx context.Context, matchID string) ([]canonical.HighlightEvent, error)

	// LoadTeamXUIDs retourne le set des XUIDs des coequipiers humains du
	// joueur cible (joueur cible exclu).
	LoadTeamXUIDs(ctx context.Context, matchID string, teamID int, targetXUID string) (map[string]bool, error)

	// LoadAllCoefficients charge tous les coefficients du joueur (toutes
	// categories de mode confondues). Pour endpoint engagement_profile.
	LoadAllCoefficients(ctx context.Context, xuid string) ([]domain.EngagementCoefficient, error)

	// LoadRatioSamples charge les paces moyennes des N derniers matchs PvP
	// du joueur sur une categorie de mode, sous forme de RatioSample
	// (algo `temporal.ComputeEngagementCoefficient`). Utilise par le pipeline
	// de recompute des coefficients en post-sync.
	//
	// Filtre cote SQL :
	//   - mode_category = ?
	//   - engagement_pace_team IS NOT NULL (skip cold-start non encore renseigne)
	// Le filtrage outliers (PaceTeamMin, PlayerActivityMin) est fait cote algo.
	//
	// Retourne une slice vide (non nil) si aucun sample. ErrEngagementUnavailable
	// si les colonnes paces ne sont pas presentes (migration non appliquee).
	LoadRatioSamples(ctx context.Context, xuid, modeCategory string, limit int) ([]temporal.RatioSample, error)
}

// MatchEngagementContext regroupe les metadonnees d'un match necessaires au
// calcul du score d'engagement. Source : shared.match_registry +
// shared.match_participants.
type MatchEngagementContext struct {
	MatchID       string
	StartTimeMS   int64
	EndTimeMS     int64
	IsRanked      bool
	IsPvE         bool
	TargetTeamID  int
	NTeam         int  // taille equipe alliee humains (joueur cible inclus)
	NHumansLobby  int  // taille lobby humains
	IsTeamMode    bool // false si NTeam == 1 (FFA-like)
	PersonalScore int
	Kills         int
	Assists       int
	MapName       *string
}

// Note historique : MatchEngagementParams + EngagementScoreService interface
// ont ete supprimes en Phase 6 du plan engagement long-term — code mort
// (NewEngagementScoreService jamais appele en production). Les chemins
// production utilisent service.PlayerEngagementService directement.
