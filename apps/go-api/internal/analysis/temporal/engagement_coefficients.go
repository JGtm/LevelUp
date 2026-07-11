// Package temporal — engagement_coefficients.go : calcul du coefficient
// d'engagement lobby global (coef_lobby_share).
//
// Reference plan : .ai/V7/PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07.md (modele v2).
//
// Concept : coef_lobby_share = mediane glissante des ratios pace_joueur/pace_lobby
// sur les N derniers matchs PvP du joueur dans une categorie (PvP_ranked /
// PvP_unranked). Caracterise la reponse habituelle GLOBALE du joueur a l'action
// totale du match (toutes intensites confondues).
//
// Il sert de FALLBACK a l'attendu (ExpectedBasis "global") quand aucun bin
// d'intensite n'est exploitable (cf. engagement_response_bins.go pour les bins).
// coef_team_share (part intra-equipe) a ete abandonne (D5, modele lobby-anchored) :
// l'attendu n'est plus une part relative a l'equipe.
//
// Pure : aucun acces DB, aucun log, aucun side-effect. Le caller charge les
// echantillons (RatioSample) puis appelle ComputeEngagementCoefficient.

package temporal

import (
	"errors"

	"levelup/go-api/internal/analysis"
)

// Constantes de calibration pour le calcul du coefficient.
//
// Les seuils sont alignes avec la doc REFLEXION_ENGAGEMENT_SCORE §6 (cold
// start raisonnable >= 10 matchs) et §16 (filtrage outliers : matchs ou le
// lobby est quasi-AFK ne portent pas d'information sur le joueur).
const (
	// MinMatchesForCoef est le nombre minimal d'echantillons valides pour
	// calculer un coefficient stable. En dessous, on retourne
	// ErrInsufficientCoefHistory et le caller garde le cold-start 1.0.
	//
	// Aligne sur HistoryMinPartial (10) — meme seuil que pour le score
	// percentile, pour rester coherent dans la dichotomie "cold vs warm".
	MinMatchesForCoef = 10

	// PaceLobbyMinThreshold : pace du lobby (events/min/joueur) en dessous duquel
	// le match est exclu de la mediane. Un lobby quasi-inactif (lag, quitters
	// massifs, custom mode AFK) genere des ratios extremes qui polluent la
	// mediane. Seuil tres bas, exclus seulement les vrais cas degeneres.
	//
	// 1.0 → 0.75 (2026-07-07, modele lobby-anchored) : la suppression du poids
	// mort (death 0.4 → 0.0, cf. engagement_weights.go) baisse mecaniquement
	// tous les paces de ~25 % sur un mix kills≈morts. Abaisser le seuil dans la
	// meme proportion evite de rejeter des matchs auparavant valides. Gate
	// empirique : taux de rejets hors-AFK < 5 % apres re-backfill, sinon 0.6.
	// Partage avec le calcul des bins (engagement_response_bins.go).
	PaceLobbyMinThreshold = 0.75

	// PlayerActivityMin : activite minimale du joueur (kills+assists+deaths)
	// pour que le match contribue a la mediane. < 3 = quitter / AFK / disco
	// rapide. Le ratio observe sur ces matchs n'est pas representatif.
	PlayerActivityMin = 3

	// DefaultRatioSampleLimit : taille de la fenetre glissante (200 derniers
	// matchs par categorie). Au-dela, l'evolution du joueur (skill, meta game)
	// fausse la mediane.
	DefaultRatioSampleLimit = 200

	// CoefMin / CoefMax : bornes physiques du coefficient. Un coef hors de
	// cette plage signale un bug en amont (mauvaise partition team/lobby,
	// match degenere). On le clamp pour eviter de propager une aberration.
	// 0.1 = joueur fait 10x moins que l'equipe (peu realiste sauf bug)
	// 5.0 = joueur fait 5x plus que l'equipe (super-carry, possible mais rare)
	CoefMin = 0.1
	CoefMax = 5.0
)

// Erreurs sentinelles. Toutes wrappables avec errors.Is.
var (
	// ErrInsufficientCoefHistory : moins de MinMatchesForCoef echantillons
	// valides apres filtrage. Le caller doit garder le cold-start 1.0/1.0.
	ErrInsufficientCoefHistory = errors.New("engagement: insufficient valid samples for coefficient")
)

// RatioSample est un echantillon (1 match) utilise pour calculer le
// coefficient median. Charge en amont par le repo via LoadRatioSamples.
//
// Tous les paces sont en events/min/joueur (cf engagement_curve.go).
type RatioSample struct {
	MatchID string

	// Paces moyennes du match (3 traces de la courbe d'engagement).
	PaceJoueur float64
	PaceTeam   float64
	PaceLobby  float64

	// PlayerActivity = kills + assists + deaths. Sert a detecter les
	// quitters/AFK qui faussent la mediane.
	PlayerActivity int
}

// CoefficientResult est le retour de ComputeEngagementCoefficient.
//
// NMatches reflete le nombre d'echantillons valides effectivement utilises
// dans la mediane (apres filtrage outliers). NRejected = total fourni - NMatches.
type CoefficientResult struct {
	CoefLobbyShare float64
	NMatches       int
	NRejected      int
}

// ComputeEngagementCoefficient calcule la mediane glissante du ratio
// pace_joueur/pace_lobby (coef lobby global) sur les echantillons fournis.
//
// Filtre applique avant calcul :
//   - PlayerActivity < PlayerActivityMin → exclus (quitter/AFK)
//   - PaceLobby < PaceLobbyMinThreshold (ou <= 0) → exclus (lobby AFK)
//
// Si moins de MinMatchesForCoef samples valides → ErrInsufficientCoefHistory.
//
// Le resultat est borne [CoefMin, CoefMax] pour eviter les aberrations.
func ComputeEngagementCoefficient(samples []RatioSample) (*CoefficientResult, error) {
	if len(samples) == 0 {
		return nil, ErrInsufficientCoefHistory
	}

	lobbyRatios := make([]float64, 0, len(samples))
	rejected := 0

	for _, s := range samples {
		if s.PlayerActivity < PlayerActivityMin {
			rejected++
			continue
		}
		if s.PaceLobby < PaceLobbyMinThreshold || s.PaceLobby <= 0 {
			rejected++
			continue
		}
		lobbyRatios = append(lobbyRatios, s.PaceJoueur/s.PaceLobby)
	}

	if len(lobbyRatios) < MinMatchesForCoef {
		return nil, ErrInsufficientCoefHistory
	}

	return &CoefficientResult{
		CoefLobbyShare: clampCoef(analysis.MedianFloat(lobbyRatios)),
		NMatches:       len(lobbyRatios),
		NRejected:      rejected,
	}, nil
}

// clampCoef borne le coefficient dans [CoefMin, CoefMax] pour eviter de
// propager une aberration (bug partition events, match degenere non filtre,
// etc.). Les bornes sont larges (0.1, 5.0) — un coef "normal" se situe
// entre 0.7 et 1.5.
func clampCoef(c float64) float64 {
	if c < CoefMin {
		return CoefMin
	}
	if c > CoefMax {
		return CoefMax
	}
	return c
}
