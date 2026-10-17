// Package skill — skill_rating.go : état glissant d'un joueur et helpers de rendement de
// combat, partagés par le simulateur de formules (skill_formula_sim.go) et le calcul des
// composantes de performance du LUSR v2 (skill_v2_components.go).
//
// Le LUSR est le modèle v2 (ADR 0024), seul à écrire match_skill_rank ; ce qui reste de
// TrueSkill 2 à plat est le simulateur de formules (skill_formula_sim.go), qui n'écrit rien.
package skill

import (
	"time"

	"levelup/go-api/internal/games"
)

// ── PlayerState — état glissant entre deux matchs ───────────────────────────

// PlayerState contient l'état mu/sigma d'un joueur pour un playlist_group.
type PlayerState struct {
	MU                   float64
	Sigma                float64
	MatchCount           int
	LastMatchTime        *time.Time
	AccuracyHistory      []float64
	DamageEffHistory     []float64
	MedalExploitHistory  []float64
	OffConversionHistory []float64
	DefResistanceHistory []float64
}

// NewPlayerState crée un état initial.
func NewPlayerState() *PlayerState {
	return &PlayerState{MU: InitialMU, Sigma: InitialSigma}
}

// rollingAvgPtr retourne la moyenne d'une slice si elle a au moins
// MinMatchesForAccuracyDelta éléments, nil sinon. Le seuil est toujours
// MinMatchesForAccuracyDelta (constant unique) — inline pour eviter le bruit
// unparam.
func rollingAvgPtr(hist []float64) *float64 {
	if len(hist) < MinMatchesForAccuracyDelta {
		return nil
	}
	sum := 0.0
	for _, v := range hist {
		sum += v
	}
	avg := sum / float64(len(hist))
	return &avg
}

// appendToHistory ajoute v à *hist et tronque a AccuracyHistorySize elements.
// La taille max est toujours AccuracyHistorySize (constant unique) — inline
// pour eviter le bruit unparam.
func appendToHistory(hist *[]float64, v float64) {
	*hist = append(*hist, v)
	if len(*hist) > AccuracyHistorySize {
		*hist = (*hist)[len(*hist)-AccuracyHistorySize:]
	}
}

// ComputeCombatYield calcule offensive_conversion et defensive_resistance depuis un match.
//
// Le baseline PV-pour-tuer est SCALE-INVARIANT ici : offConv/defRes sont ensuite
// comparés à la moyenne glissante DU MÊME joueur (avgOffConv/avgDefRes dans
// computeCompositeScoreWithBreakdown), tous calculés avec la même constante → le
// facteur s'annule dans le ratio et n'affecte PAS le classement LUSR. On utilise
// donc le baseline Infinite par défaut (games.DefaultEffectiveHpToKill) comme
// simple constante d'échelle nommée — title-agnostic par construction, sans avoir
// à threader le slug sur ce hot path. Le baseline title-aware (225 Infinite / 115
// Halo 5) ne concerne que le KPI Rendement/Résistance AFFICHÉ (match-view), résolu
// séparément via games.EffectiveHpToKill(slug).
func ComputeCombatYield(m LusrMatchData) (offConv, defRes float64) {
	const hpToKill = games.DefaultEffectiveHpToKill
	if m.DamageDealt > 0 {
		offConv = hpToKill * (m.Kills + m.Assists/3.0) / m.DamageDealt
	}
	if m.DamageTaken > 0 && m.Deaths > 0 {
		defRes = m.DamageTaken / (hpToKill * m.Deaths)
	}
	return
}
