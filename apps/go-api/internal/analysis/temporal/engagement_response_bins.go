// Package temporal — engagement_response_bins.go : bins de reponse d'engagement
// (modele lobby-anchored v2, cf. .ai/V7/PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07.md).
//
// Concept : l'attendu du joueur n'est pas une part relative a son equipe mais
// « sa reponse habituelle a un match d'intensite similaire ». On classe les
// matchs du joueur en 3 bins d'intensite (terciles de pace_lobby, adaptatifs par
// joueur et par mode) et on calcule, pour chaque bin, la mediane de
// pace_joueur / pace_lobby (le coef de reponse lobby-anchored). Au serving,
// l'intensite du match courant (mean pace_lobby de sa courbe) selectionne le bin.
//
// Pur : aucun acces DB, aucun log, aucun side-effect. Le caller (recompute)
// charge les RatioSample puis appelle ComputeEngagementResponseBins.
package temporal

import (
	"errors"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// Libelles des bins d'intensite (terciles de pace_lobby). Vocabulaire produit
// (calme / standard / chaotique) aligne sur la classification d'intensite de
// match. cf plan D2.
const (
	IntensityBinCalme     = "calme"
	IntensityBinStandard  = "standard"
	IntensityBinChaotique = "chaotique"
)

// MinMatchesForBin est le nombre minimal d'echantillons valides DANS UN BIN pour
// que son coefficient soit exploitable a l'affichage (ExpectedBasis "bin"). En
// dessous, l'attendu retombe sur le coef lobby global. Aligne sur
// MinMatchesForCoef (10) — meme seuil que le cold-start global.
const MinMatchesForBin = 10

// ErrInsufficientBinHistory : moins de MinMatchesForCoef echantillons valides au
// total → impossible de former des terciles significatifs. Le caller ne persiste
// aucun bin (l'attendu retombera sur global / cold-start).
var ErrInsufficientBinHistory = errors.New("engagement: insufficient valid samples for response bins")

// ResponseBinsResult est le retour de ComputeEngagementResponseBins.
//
// Bins porte TOUJOURS les 3 terciles (calme/standard/chaotique) quand assez
// d'echantillons valides existent, meme si un bin est sous MinMatchesForBin :
// le NMatches par bin permet au serving de decider (bin exploitable vs fallback
// global). NRejected = samples ecartes par le filtre (AFK / lobby inactif).
type ResponseBinsResult struct {
	Bins      []domain.EngagementIntensityBin
	NRejected int
}

// binSample est un echantillon retenu apres filtrage : intensite (pace_lobby) et
// ratio de reponse (pace_joueur / pace_lobby).
type binSample struct {
	lobby float64
	ratio float64
}

// ComputeEngagementResponseBins calcule les 3 bins d'intensite (terciles de
// pace_lobby) et, pour chacun, la mediane de pace_joueur/pace_lobby.
//
// Filtre (identique a ComputeEngagementCoefficient) : PlayerActivity >=
// PlayerActivityMin, PaceLobby >= PaceLobbyMinThreshold, PaceLobby > 0.
//
// Emet systematiquement les 3 cles (bornes contiguës) : un jeu de cles constant
// evite les lignes orphelines a la persistence (SELECT-then-UPDATE-or-INSERT).
// Un bin vide (tercile degenere t1==t2) recoit coef 1.0 et NMatches 0.
//
// Renvoie ErrInsufficientBinHistory si moins de MinMatchesForCoef samples valides.
func ComputeEngagementResponseBins(samples []RatioSample) (*ResponseBinsResult, error) {
	valid := make([]binSample, 0, len(samples))
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
		valid = append(valid, binSample{lobby: s.PaceLobby, ratio: s.PaceJoueur / s.PaceLobby})
	}
	if len(valid) < MinMatchesForCoef {
		return nil, ErrInsufficientBinHistory
	}

	// Terciles de la distribution des intensites (pace_lobby).
	lobbies := make([]float64, len(valid))
	for i, v := range valid {
		lobbies[i] = v.lobby
	}
	sort.Float64s(lobbies)
	t1 := quantileSorted(lobbies, 1.0/3.0)
	t2 := quantileSorted(lobbies, 2.0/3.0)
	maxLobby := lobbies[len(lobbies)-1]

	// Partition des ratios en 3 groupes selon l'intensite du match.
	var calme, standard, chaotique []float64
	for _, v := range valid {
		switch {
		case v.lobby < t1:
			calme = append(calme, v.ratio)
		case v.lobby < t2:
			standard = append(standard, v.ratio)
		default:
			chaotique = append(chaotique, v.ratio)
		}
	}

	bins := []domain.EngagementIntensityBin{
		buildResponseBin(IntensityBinCalme, 0, t1, calme),
		buildResponseBin(IntensityBinStandard, t1, t2, standard),
		buildResponseBin(IntensityBinChaotique, t2, maxLobby, chaotique),
	}
	return &ResponseBinsResult{Bins: bins, NRejected: rejected}, nil
}

// buildResponseBin assemble un bin : mediane des ratios (clampee [CoefMin,
// CoefMax]) ou 1.0 si le bin est vide.
func buildResponseBin(label string, lower, upper float64, ratios []float64) domain.EngagementIntensityBin {
	coef := 1.0
	if len(ratios) > 0 {
		coef = clampCoef(analysis.MedianFloat(ratios))
	}
	return domain.EngagementIntensityBin{
		Bin:        label,
		LowerBound: lower,
		UpperBound: upper,
		CoefLobby:  coef,
		NMatches:   len(ratios),
	}
}

// quantileSorted retourne le quantile q (0..1) d'une slice TRIEE non vide
// (interpolation lineaire entre les deux rangs encadrants).
func quantileSorted(sorted []float64, q float64) float64 {
	n := len(sorted)
	if n == 1 {
		return sorted[0]
	}
	pos := q * float64(n-1)
	lo := int(pos)
	if lo >= n-1 {
		return sorted[n-1]
	}
	frac := pos - float64(lo)
	return sorted[lo] + frac*(sorted[lo+1]-sorted[lo])
}
