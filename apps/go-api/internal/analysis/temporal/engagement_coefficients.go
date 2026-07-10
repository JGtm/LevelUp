// Package temporal — engagement_coefficients.go : calcul des coefficients
// d'engagement personnels (coef_team_share, coef_lobby_share).
//
// Reference plan : .ai/V7/PLAN_ENGAGEMENT_IMPLEMENTATION.md §4.4 et §6.7.
//
// Concept : coef_team_share = mediane glissante des ratios pace_joueur/pace_team
// sur les N derniers matchs PvP du joueur dans une categorie (PvP_ranked /
// PvP_unranked). Caracterise "a quel point le joueur fait sa part" face a son
// equipe sur la duree.
//
// Le coef_team_share est ensuite utilise par ComputeEngagementScore pour
// construire pace_attendu = coef × pace_team. Quand le coef vaut 1.0
// (cold-start), l'attendu = team observee → courbes superposees a l'ecran.
// L'objectif de ce module est de remplacer ce 1.0 cold-start par une mediane
// stable et personnalisee.
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

	// PaceTeamMinThreshold : pace de l'equipe (events/min/joueur) en dessous
	// duquel le match est exclu de la mediane. Un lobby quasi-inactif (lag,
	// quitters massifs, custom mode AFK) genere des ratios extremes qui
	// polluent la mediane. 1.0 event/min/joueur = 1 kill/death toutes les 60s
	// par joueur d'equipe : seuil tres bas, exclus seulement les vrais cas
	// degeneres.
	PaceTeamMinThreshold = 1.0

	// PaceLobbyMinThreshold : meme idee pour le ratio lobby. Seuil identique.
	PaceLobbyMinThreshold = 1.0

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
	CoefTeamShare  float64
	CoefLobbyShare float64
	NMatches       int
	NRejected      int
}

// ComputeEngagementCoefficient calcule la mediane glissante des ratios
// pace_joueur/pace_team et pace_joueur/pace_lobby sur les echantillons
// fournis.
//
// Filtre applique avant calcul :
//   - PlayerActivity < PlayerActivityMin → exclus (quitter/AFK)
//   - PaceTeam < PaceTeamMinThreshold → exclus pour le ratio team (lobby AFK)
//   - PaceLobby < PaceLobbyMinThreshold → exclus pour le ratio lobby (idem)
//
// Le filtre team et lobby sont independants : un sample peut contribuer au
// coef_team mais pas au coef_lobby (ou inversement). NMatches refere au
// nombre de samples valides pour le coef_team_share (le plus restrictif en
// pratique car PaceTeam <= PaceLobby).
//
// Si moins de MinMatchesForCoef samples valides → ErrInsufficientCoefHistory.
//
// Le resultat est borne [CoefMin, CoefMax] pour eviter les aberrations.
func ComputeEngagementCoefficient(samples []RatioSample) (*CoefficientResult, error) {
	if len(samples) == 0 {
		return nil, ErrInsufficientCoefHistory
	}

	teamRatios := make([]float64, 0, len(samples))
	lobbyRatios := make([]float64, 0, len(samples))
	rejected := 0

	for _, s := range samples {
		if s.PlayerActivity < PlayerActivityMin {
			rejected++
			continue
		}
		// Le sample est conserve s'il contribue a au moins l'un des 2 ratios.
		// Sinon (lobby AFK total), on le compte comme rejected.
		teamOK := s.PaceTeam >= PaceTeamMinThreshold && s.PaceTeam > 0
		lobbyOK := s.PaceLobby >= PaceLobbyMinThreshold && s.PaceLobby > 0
		if !teamOK && !lobbyOK {
			rejected++
			continue
		}
		if teamOK {
			teamRatios = append(teamRatios, s.PaceJoueur/s.PaceTeam)
		}
		if lobbyOK {
			lobbyRatios = append(lobbyRatios, s.PaceJoueur/s.PaceLobby)
		}
	}

	// On exige MinMatchesForCoef samples valides pour le coef team (le plus
	// utilise en pratique pour les modes en equipe).
	if len(teamRatios) < MinMatchesForCoef {
		return nil, ErrInsufficientCoefHistory
	}

	coefTeam := clampCoef(analysis.MedianFloat(teamRatios))
	coefLobby := 1.0 // fallback neutre si pas assez de samples lobby valides
	if len(lobbyRatios) >= MinMatchesForCoef {
		coefLobby = clampCoef(analysis.MedianFloat(lobbyRatios))
	}

	return &CoefficientResult{
		CoefTeamShare:  coefTeam,
		CoefLobbyShare: coefLobby,
		NMatches:       len(teamRatios),
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
