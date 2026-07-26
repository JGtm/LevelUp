// Package analysis — overtime.go : détection « Prolongation » d'un match.
//
// Un match dont la durée de jeu dépasse nettement le temps réglementaire de sa
// variante (table data-driven config/titles/{slug}/mappings/regulation.toml) est
// parti en prolongation. Algo PUR : aucune DB, aucun HTTP, aucun slug.
package analysis

import "strings"

// OvertimeMarginSeconds est la MARGE unique au-delà du temps réglementaire à
// partir de laquelle un match est considéré en prolongation.
//
// Valeur mesurée (investigation 1793 matchs, 2026-07) :
//   - seuil T_reg + 40 s : 0 faux positif sur les 724 matchs Slayer de contrôle
//     (aucun match terminé dans le temps n'est flagué) ;
//   - seuil T_reg + 20 s : 3 faux positifs — le bruit d'horloge de fin de match
//     (arrêt du chrono, dernier frag, écran de fin) suffit à franchir 20 s ;
//   - coût assumé du choix : 2 prolongations COURTES observées (+19 s et +24 s)
//     restent sous le seuil et ne sont donc jamais signalées. Un faux négatif
//     est acceptable, un faux positif ne l'est pas (le badge doit être fiable).
//
// SOURCE UNIQUE : ne jamais recopier ce 40 ailleurs (garde-rail
// overtime_guard_test.go). Toute évolution du seuil se fait ici, avec la mesure
// qui la justifie.
const OvertimeMarginSeconds = 40

// ComputeOvertime indique si un match est parti en prolongation et de combien.
//
//   - elapsedSeconds    : durée de jeu observée du match (médiane du temps joué
//     des participants présents du début à la fin, cf. repo).
//   - regulationSeconds : temps réglementaire de la variante (table du titre).
//     0 ou négatif = variante inconnue / titre sans table → JAMAIS de flag.
//
// Retourne (isOvertime, overtimeSeconds). overtimeSeconds est le dépassement
// RÉEL du temps réglementaire (elapsed − regulation), pas le dépassement du
// seuil : un match flagué à 763 s sur 720 s de règlement a 43 s de prolongation.
// Il vaut 0 quand isOvertime est faux (rien à afficher).
func ComputeOvertime(elapsedSeconds, regulationSeconds int) (isOvertime bool, overtimeSeconds int) {
	if regulationSeconds <= 0 || elapsedSeconds <= 0 {
		return false, 0
	}
	// Marge INCLUSE : elapsed == regulation + marge reste du temps réglementaire
	// (le bruit d'horloge de fin de match vit dans cette fenêtre).
	if elapsedSeconds <= regulationSeconds+OvertimeMarginSeconds {
		return false, 0
	}
	return true, elapsedSeconds - regulationSeconds
}

// ResolveOvertime est le CHOKEPOINT unique « variante de jeu → prolongation » :
// il résout le temps réglementaire dans la table du titre (chargée depuis
// config/titles/{slug}/mappings/regulation.toml) puis délègue à ComputeOvertime.
//
// Toutes les surfaces (Match View, Explorateur) passent par ici — aucune ne
// refait le lookup. Titre sans table, variante inconnue ou durée non estimable
// → (false, 0), jamais d'erreur. Aucun slug n'entre dans cette fonction : le
// titre est déjà porté par la table injectée.
func ResolveOvertime(gameVariantName string, elapsedSeconds *int, regulationSeconds map[string]int) (isOvertime bool, overtimeSeconds int) {
	if len(regulationSeconds) == 0 || elapsedSeconds == nil {
		return false, 0
	}
	reg, ok := regulationSeconds[strings.TrimSpace(gameVariantName)]
	if !ok {
		return false, 0
	}
	return ComputeOvertime(*elapsedSeconds, reg)
}
