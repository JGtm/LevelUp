// Package skill — skill_rating.go : état glissant d'un joueur et helpers de rendement de
// combat, partagés par le simulateur de formules (skill_formula_sim.go) et le calcul des
// composantes de performance du LUSR v2 (skill_v2_components.go).
//
// Le LUSR est le modèle v2 (ADR 0024), seul à écrire match_skill_rank ; ce qui reste de
// TrueSkill 2 à plat est le simulateur de formules (skill_formula_sim.go), qui n'écrit rien.
package skill

import (
	"time"

	"levelup/go-api/internal/domain/title"
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
// La vie effective est celle du titre par défaut (games.EffectiveHpToKill(title.DefaultSlug)), quel que
// soit le titre du match : la fonction n'a pas son slug sur ce chemin chaud. Halo 5 déclare pourtant la
// capacité `lusr` avec sa propre vie (config/titles/halo_5/constants.toml) : un match de ce titre qui
// passerait ici serait calculé avec la vie du titre par défaut. Les deux grandeurs sont ensuite
// comparées à la moyenne glissante du MÊME joueur, calculée avec la même vie : le facteur s'annule dans le
// ratio, et ne pèse que sur la référence absolue des premiers matchs.
func ComputeCombatYield(m LusrMatchData) (offConv, defRes float64) {
	hpToKill := games.EffectiveHpToKill(title.DefaultSlug)
	if m.DamageDealt > 0 {
		offConv = hpToKill * (m.Kills + m.Assists/3.0) / m.DamageDealt
	}
	if m.DamageTaken > 0 && m.Deaths > 0 {
		defRes = m.DamageTaken / (hpToKill * m.Deaths)
	}
	return
}
