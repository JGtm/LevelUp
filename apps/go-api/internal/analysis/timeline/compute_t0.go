package timeline

import (
	"sort"
	"time"
)

// T0Quality qualifie la fiabilité du T0 calculé pour un match. Stockée en DB
// (colonne t0_quality) pour tracer les cas dégradés et permettre des audits.
type T0Quality string

const (
	// T0QualityOK : ≥2 sources concordantes (spread ≤ seuil), T0 retenu = min.
	T0QualityOK T0Quality = "ok"
	// T0QualitySingleSource : une seule source present_at_beginning exploitable.
	T0QualitySingleSource T0Quality = "single_source"
	// T0QualitySpreadHigh : sources divergentes (spread > seuil), T0 = médiane.
	T0QualitySpreadHigh T0Quality = "spread_high"
	// T0QualityNoData : aucune source exploitable → T0 non calculable (NULL).
	T0QualityNoData T0Quality = "no_data"
	// T0QualityNegative : T0 calculé < 0 (incohérence timezone/données) → rejet.
	T0QualityNegative T0Quality = "negative"
	// T0QualitySuspiciousHigh : T0 > seuil max plausible → rejet.
	T0QualitySuspiciousHigh T0Quality = "suspicious_high"
)

// Computed indique si la qualité correspond à un T0 exploitable (stockable) ou
// à un rejet (T0 doit rester NULL, fallback T0=0 au runtime).
func (q T0Quality) Computed() bool {
	switch q {
	case T0QualityOK, T0QualitySingleSource, T0QualitySpreadHigh:
		return true
	default:
		return false
	}
}

// Seuils du filet de sécurité multi-joueurs.
const (
	// t0SpreadThresholdMs : au-delà, les first_joined_time des joueurs présents
	// au début divergent trop (latence réseau hétérogène, données douteuses) →
	// on bascule sur la médiane plutôt que le min.
	t0SpreadThresholdMs int64 = 2000
	// t0MaxPlausibleMs : countdown pré-match plausible max. Au-delà, on suspecte
	// un bug (timezone, données aberrantes) et on rejette.
	t0MaxPlausibleMs int64 = 120_000
)

// ParticipationT0Input est la donnée par joueur nécessaire au calcul de T0.
// Les bots et les joueurs absents au début sont filtrés par l'appelant OU via
// les flags ci-dessous.
type ParticipationT0Input struct {
	FirstJoinedTime    time.Time
	PresentAtBeginning bool
	IsBot              bool // gamertag absent / xuid de bot
}

// ComputeT0 estime le décalage T0 (countdown pré-match, en millisecondes) d'un
// match à partir des first_joined_time des joueurs présents au début.
//
// startTimeUTC doit être le début du film en UTC (pattern canonical :
// via analysis.SQLStartTimeCanonical cote SQL).
//
// Filet de sécurité multi-joueurs :
//   - exclut bots et joueurs non présents au début ;
//   - 0 source → NoData (T0 NULL) ;
//   - 1 source → SingleSource (T0 = source − start) ;
//   - ≥2 sources, spread ≤ 2s → OK (T0 = min − start, le premier arrivé) ;
//   - ≥2 sources, spread > 2s → SpreadHigh (T0 = médiane − start) ;
//   - T0 < 0 → Negative (rejet) ; T0 > 120s → SuspiciousHigh (rejet).
//
// Retourne (t0Ms, quality). Si quality.Computed() est faux, t0Ms vaut 0 et ne
// doit pas être stocké comme valeur valide (fallback runtime T0=0).
func ComputeT0(parts []ParticipationT0Input, startTimeUTC time.Time) (int64, T0Quality) {
	joinMs := make([]int64, 0, len(parts))
	startMs := startTimeUTC.UnixMilli()
	for _, p := range parts {
		if p.IsBot || !p.PresentAtBeginning || p.FirstJoinedTime.IsZero() {
			continue
		}
		joinMs = append(joinMs, p.FirstJoinedTime.UnixMilli())
	}

	if len(joinMs) == 0 {
		return 0, T0QualityNoData
	}

	sort.Slice(joinMs, func(i, j int) bool { return joinMs[i] < joinMs[j] })

	var t0 int64
	var quality T0Quality
	switch {
	case len(joinMs) == 1:
		t0 = joinMs[0] - startMs
		quality = T0QualitySingleSource
	default:
		spread := joinMs[len(joinMs)-1] - joinMs[0]
		if spread > t0SpreadThresholdMs {
			t0 = medianInt64(joinMs) - startMs
			quality = T0QualitySpreadHigh
		} else {
			t0 = joinMs[0] - startMs
			quality = T0QualityOK
		}
	}

	if t0 < 0 {
		return 0, T0QualityNegative
	}
	if t0 > t0MaxPlausibleMs {
		return 0, T0QualitySuspiciousHigh
	}
	return t0, quality
}

// medianInt64 retourne la médiane d'un slice trié non vide (moyenne basse des
// deux centraux si longueur paire).
func medianInt64(sorted []int64) int64 {
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
