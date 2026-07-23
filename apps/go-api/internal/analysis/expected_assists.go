// Package analysis — expected_assists.go : arithmétique PURE du modèle d'assists
// attendus (OLS personnel + fallback populationnel). Centralisée ici (source
// unique) car partagée par le package service (is_me : Match View, Timeseries,
// Sessions) et le sous-package teammates (par membre d'escouade), qui ne peuvent
// pas partager de helper sans passer par analysis (règle CLAUDE.md n°6 ≤2 copies).
//
// La RÉSOLUTION du modèle (lecture player DB / metadata) reste côté service —
// seule l'arithmétique, testable sans IO, vit ici.
package analysis

import (
	"math"

	"levelup/go-api/internal/domain"
)

// ApplyPersonalAssistsModel applique le modèle OLS personnel d'assists attendus
// (intercept + Σ coef·feature), arrondi à 2 décimales.
func ApplyPersonalAssistsModel(m *domain.PlayerAssistsModel, kills, deaths, damageDealt, damageTaken, mmrDelta float64) float64 {
	raw := m.Intercept +
		m.CoefKills*kills +
		m.CoefDeaths*deaths +
		m.CoefDamageDealt*damageDealt +
		m.CoefDamageTaken*damageTaken +
		m.CoefMMRDelta*mmrDelta
	return math.Round(raw*100) / 100
}

// ApplyPopulationalAssists applique le modèle populationnel
// slope × (personal_score + shots_hit) + intercept, arrondi à 2 décimales.
func ApplyPopulationalAssists(slope, intercept, personalScore, shotsHit float64) float64 {
	return math.Round((slope*(personalScore+shotsHit)+intercept)*100) / 100
}
