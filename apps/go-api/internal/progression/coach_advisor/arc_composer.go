package coach_advisor

import (
	"sort"

	"levelup/go-api/internal/prestige"
)

// arc_composer.go — composition d'un Arc dynamique à partir de signaux
// convergents sur un même radar_axis (cf. ADR 0028).
//
// Fonction PURE — pas d'I/O, pas d'accès au catalogue Prestige ni au
// synthesizer. Le service orchestrateur (Phase 7) résout les template_ids
// après la composition (matcher + synthesizer si nécessaire).

// ArcSpec décrit un arc dynamique prêt à être matérialisé par
// prestige.CreateArc + N CreateChallenge à l'acceptance.
//
// Aucune information de palier absolu — les SuggestedTier des Steps sont
// indicatifs (cf. invariant I1 ADR 0020).
type ArcSpec struct {
	TitleEN         string
	TitleFR         string
	DescriptionEN   string
	DescriptionFR   string
	RadarAxis       string  // l'axis pivot
	AverageStrength float64 // moyenne des strength des signaux retenus
	Steps           []ArcStep
}

// ArcStep est une étape de l'arc — un Signal (à résoudre en template_id par
// le service via matcher ou synthesizer) + un tier suggéré (indicatif UI).
type ArcStep struct {
	Position      int // 1-indexed
	Signal        Signal
	SuggestedTier prestige.Tier
}

// ArcComposerConfig regroupe les seuils numériques de la composition.
type ArcComposerConfig struct {
	// MinSignalsForArc : nombre minimal de signaux convergents sur un même
	// radar_axis pour qu'un arc soit proposé. Défaut 2.
	MinSignalsForArc int
	// MaxArcSteps : nombre maximal d'étapes dans l'arc (cap dur). Défaut 4.
	MaxArcSteps int
	// MinStrengthForArcSignal : seuil par signal pour qu'il puisse intégrer
	// un arc (filtre faible avant composition). Défaut 0.5.
	MinStrengthForArcSignal float64
	// RequireOneStrongSignal : si true, au moins un signal de l'arc doit
	// avoir strength >= synthesis_min_strength (0.6) — sinon l'arc est rejeté
	// (évite arcs uniquement constitués de signaux faibles). Défaut true.
	RequireOneStrongSignal bool
}

// DefaultArcComposerConfig retourne les valeurs canoniques de l'ADR.
func DefaultArcComposerConfig() ArcComposerConfig {
	return ArcComposerConfig{
		MinSignalsForArc:        2,
		MaxArcSteps:             4,
		MinStrengthForArcSignal: 0.5,
		RequireOneStrongSignal:  true,
	}
}

// TryCompose retourne (ArcSpec, true) si les conditions de composition sont
// remplies, (zero, false) sinon. Pas d'erreur — composition silencieuse.
//
// Algorithme :
//  1. Filtre les signaux avec strength >= cfg.MinStrengthForArcSignal et
//     RadarAxis non vide.
//  2. Regroupe par RadarAxis.
//  3. Choisit l'axis avec le plus de signaux (tie-break par ordre
//     alphabétique pour déterminisme).
//  4. Vérifie qu'il y a >= cfg.MinSignalsForArc signaux sur cet axis.
//  5. Si cfg.RequireOneStrongSignal : vérifie qu'au moins un signal a
//     strength >= 0.6 (synthesis_min_strength).
//  6. Trie les signaux par strength décroissante, prend max cfg.MaxArcSteps.
//  7. Construit l'ArcSpec avec titres/descriptions paramétrés par l'axis et
//     SuggestedTier croissants (Normal → Mythic).
func TryCompose(signals []Signal, cfg ArcComposerConfig) (ArcSpec, bool) {
	if cfg.MinSignalsForArc < 2 {
		cfg.MinSignalsForArc = 2
	}
	if cfg.MaxArcSteps < 2 {
		cfg.MaxArcSteps = 4
	}

	// 1. Filtre
	filtered := make([]Signal, 0, len(signals))
	for _, s := range signals {
		if s.Strength >= cfg.MinStrengthForArcSignal && s.RadarAxis != "" {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) < cfg.MinSignalsForArc {
		return ArcSpec{}, false
	}

	// 2. Group by axis
	byAxis := map[string][]Signal{}
	for _, s := range filtered {
		byAxis[s.RadarAxis] = append(byAxis[s.RadarAxis], s)
	}

	// 3. Pick axis with most signals (tie-break alphabetical)
	pivotAxis, pivotSignals := pickPivotAxis(byAxis)

	// 4. Min signals check
	if len(pivotSignals) < cfg.MinSignalsForArc {
		return ArcSpec{}, false
	}

	// 5. RequireOneStrongSignal
	if cfg.RequireOneStrongSignal && !hasStrongSignal(pivotSignals, 0.6) {
		return ArcSpec{}, false
	}

	// 6. Sort by strength desc, cap to MaxArcSteps
	sort.SliceStable(pivotSignals, func(i, j int) bool {
		if pivotSignals[i].Strength != pivotSignals[j].Strength {
			return pivotSignals[i].Strength > pivotSignals[j].Strength
		}
		// Tie-break stable par Kind alphabétique
		return string(pivotSignals[i].Kind) < string(pivotSignals[j].Kind)
	})
	if len(pivotSignals) > cfg.MaxArcSteps {
		pivotSignals = pivotSignals[:cfg.MaxArcSteps]
	}

	// 7. Build ArcSpec
	titleEN, titleFR := arcTitleForAxis(pivotAxis)
	descEN, descFR := arcDescriptionForAxis(pivotAxis, len(pivotSignals))
	tiers := suggestedTierProgression(len(pivotSignals))

	steps := make([]ArcStep, len(pivotSignals))
	avgStrength := 0.0
	for i, s := range pivotSignals {
		steps[i] = ArcStep{
			Position:      i + 1,
			Signal:        s,
			SuggestedTier: tiers[i],
		}
		avgStrength += s.Strength
	}
	avgStrength /= float64(len(pivotSignals))

	return ArcSpec{
		TitleEN:         titleEN,
		TitleFR:         titleFR,
		DescriptionEN:   descEN,
		DescriptionFR:   descFR,
		RadarAxis:       pivotAxis,
		AverageStrength: avgStrength,
		Steps:           steps,
	}, true
}

// pickPivotAxis retourne (axis, signals) pour l'axis avec le plus de
// signaux. Tie-break alphabétique sur le nom de l'axis pour déterminisme.
func pickPivotAxis(byAxis map[string][]Signal) (string, []Signal) {
	type axisCount struct {
		axis  string
		count int
	}
	axes := make([]axisCount, 0, len(byAxis))
	for a, ss := range byAxis {
		axes = append(axes, axisCount{axis: a, count: len(ss)})
	}
	sort.SliceStable(axes, func(i, j int) bool {
		if axes[i].count != axes[j].count {
			return axes[i].count > axes[j].count
		}
		return axes[i].axis < axes[j].axis
	})
	if len(axes) == 0 {
		return "", nil
	}
	picked := axes[0].axis
	return picked, byAxis[picked]
}

// hasStrongSignal retourne true si au moins un signal a Strength >= threshold.
func hasStrongSignal(signals []Signal, threshold float64) bool {
	for _, s := range signals {
		if s.Strength >= threshold {
			return true
		}
	}
	return false
}

// suggestedTierProgression retourne les tiers indicatifs pour N étapes.
//   - 2 étapes : Normal, Heroic
//   - 3 étapes : Normal, Heroic, Legendary
//   - 4 étapes : Normal, Heroic, Legendary, Mythic
//
// Prestige recalcule à l'acceptance via baseline (cf. I1).
func suggestedTierProgression(n int) []prestige.Tier {
	all := []prestige.Tier{
		prestige.TierNormal,
		prestige.TierHeroic,
		prestige.TierLegendary,
		prestige.TierMythic,
	}
	if n > len(all) {
		n = len(all)
	}
	out := make([]prestige.Tier, n)
	copy(out, all[:n])
	return out
}
